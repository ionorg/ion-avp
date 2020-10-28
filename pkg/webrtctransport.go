package avp

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	log "github.com/pion/ion-log"
	"github.com/pion/rtcp"
	"github.com/pion/sdp/v2"

	"github.com/pion/webrtc/v3"
)

type SFUFeedback struct {
	StreamID string `json:"streamId"`
	Video    string `json:"video"`
	Audio    bool   `json:"audio"`
}

type PendingProcess struct {
	pid string
	fn  func() Element
}

// WebRTCTransport represents a webrtc transport
type WebRTCTransport struct {
	id        string
	pc        *webrtc.PeerConnection
	mu        sync.RWMutex
	feedback  *webrtc.DataChannel
	builders  map[string]*Builder         // one builder per track
	pending   map[string][]PendingProcess // maps track id to pending element constructors
	processes map[string]Element          // existing processes
	onCloseFn func()
}

// NewWebRTCTransport creates a new webrtc transport
func NewWebRTCTransport(id string, c Config) *WebRTCTransport {
	conf := webrtc.Configuration{}
	se := webrtc.SettingEngine{}

	var icePortStart, icePortEnd uint16

	if len(c.WebRTC.ICEPortRange) == 2 {
		icePortStart = c.WebRTC.ICEPortRange[0]
		icePortEnd = c.WebRTC.ICEPortRange[1]
	}

	if icePortStart != 0 || icePortEnd != 0 {
		if err := se.SetEphemeralUDPPortRange(icePortStart, icePortEnd); err != nil {
			panic(err)
		}
	}

	var iceServers []webrtc.ICEServer
	for _, iceServer := range c.WebRTC.ICEServers {
		s := webrtc.ICEServer{
			URLs:       iceServer.URLs,
			Username:   iceServer.Username,
			Credential: iceServer.Credential,
		}
		iceServers = append(iceServers, s)
	}

	conf.ICEServers = iceServers

	// Create peer connection
	me := webrtc.MediaEngine{}
	me.RegisterDefaultCodecs()
	api := webrtc.NewAPI(webrtc.WithMediaEngine(me), webrtc.WithSettingEngine(se))
	pc, err := api.NewPeerConnection(conf)

	if err != nil {
		log.Errorf("Error creating peer connection: %s", err)
		return nil
	}

	feedback, err := pc.CreateDataChannel("ion-sfu", nil)
	if err != nil {
		log.Errorf("Error creating peer data channel: %s", err)
		return nil
	}

	t := &WebRTCTransport{
		id:        id,
		feedback:  feedback,
		pc:        pc,
		builders:  make(map[string]*Builder),
		pending:   make(map[string][]PendingProcess),
		processes: make(map[string]Element),
	}

	pc.OnTrack(func(track *webrtc.Track, recv *webrtc.RTPReceiver) {
		id := track.ID()
		log.Infof("Got track: %s", id)
		builder := NewBuilder(track, 200)
		t.mu.Lock()
		defer t.mu.Unlock()
		t.builders[id] = builder

		// If there is a pending pipeline for this track,
		// initialize the pipeline.
		if pending := t.pending[id]; len(pending) != 0 {
			for _, p := range pending {
				process := t.processes[p.pid]
				if process == nil {
					process = p.fn()
					t.processes[p.pid] = process
				}
				builder.AttachElement(process)
			}
			delete(t.pending, id)
		}

		if track.Kind() == webrtc.RTPCodecTypeVideo {
			err := pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{SenderSSRC: track.SSRC(), MediaSSRC: track.SSRC()}})
			if err != nil {
				log.Errorf("error writing pli %s", err)
			}
		}

		builder.OnStop(func() {
			t.mu.Lock()
			b := t.builders[id]
			if b != nil {
				log.Debugf("stop builder %s", id)
				delete(t.builders, id)
			}
			t.mu.Unlock()

			if t.isEmpty() {
				// No more tracks, cleanup transport
				t.Close()
			}
		})
	})

	go t.pliLoop(c.WebRTC.PLICycle)

	return t
}

func (t *WebRTCTransport) pliLoop(cycle uint) {
	if cycle == 0 {
		return
	}

	ticker := time.NewTicker(time.Duration(cycle) * time.Millisecond)
	for range ticker.C {
		t.mu.RLock()
		builders := t.builders
		t.mu.RUnlock()

		if len(builders) == 0 {
			return
		}

		var pkts []rtcp.Packet
		for _, b := range builders {
			pkts = append(pkts, &rtcp.PictureLossIndication{SenderSSRC: b.Track().SSRC(), MediaSSRC: b.Track().SSRC()})
		}

		err := t.pc.WriteRTCP(pkts)
		if err != nil {
			log.Errorf("error writing pli %s", err)
		}
	}
}

func (t *WebRTCTransport) isEmpty() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.builders) == 0 && len(t.pending) == 0
}

// OnClose sets a handler that is called when the webrtc transport is closed
func (t *WebRTCTransport) OnClose(f func()) {
	t.onCloseFn = f
}

// Close the webrtc transport
func (t *WebRTCTransport) Close() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.onCloseFn != nil {
		t.onCloseFn()
	}
	return t.pc.Close()
}

// Process creates a pipeline
func (t *WebRTCTransport) Process(pid, tid, eid string, config []byte) error {
	log.Infof("WebRTCTransport.Process id=%s", pid)
	t.mu.Lock()
	defer t.mu.Unlock()

	e := registry.GetElement(eid)
	if e == nil {
		log.Errorf("element not found: %s", eid)
		return errors.New("element not found")
	}

	b := t.builders[tid]
	if b == nil {
		log.Debugf("builder not found for track %s. queuing.", tid)
		t.pending[tid] = append(t.pending[tid], PendingProcess{
			pid: pid,
			fn:  func() Element { return e(t.id, pid, tid, config) },
		})
		return nil
	}

	process := t.processes[pid]
	if process == nil {
		process = e(t.id, pid, tid, config)
		t.processes[pid] = process
	}

	b.AttachElement(process)

	return nil
}

// CreateOffer starts the PeerConnection and generates the localDescription
func (t *WebRTCTransport) CreateOffer() (webrtc.SessionDescription, error) {
	return t.pc.CreateOffer(nil)
}

// CreateAnswer starts the PeerConnection and generates the localDescription
func (t *WebRTCTransport) CreateAnswer() (webrtc.SessionDescription, error) {
	return t.pc.CreateAnswer(nil)
}

// SetLocalDescription sets the SessionDescription of the local peer
func (t *WebRTCTransport) SetLocalDescription(desc webrtc.SessionDescription) error {
	return t.pc.SetLocalDescription(desc)
}

// SetRemoteDescription sets the SessionDescription of the remote peer
func (t *WebRTCTransport) SetRemoteDescription(desc webrtc.SessionDescription) error {
	parsed := sdp.SessionDescription{}
	if err := parsed.Unmarshal([]byte(desc.SDP)); err == nil {
		for _, m := range parsed.MediaDescriptions {
			if msid, ok := m.Attribute(sdp.AttrKeyMsid); ok {
				split := strings.Split(msid, " ")
				if len(split) != 2 {
					log.Debugf("Invalid msid: %+v", msid)
					continue
				}

				mid := split[0]

				// Request unmute from sfu
				req := SFUFeedback{
					StreamID: mid,
					Video:    "high",
					Audio:    true,
				}
				msg, err := json.Marshal(req)
				if err != nil {
					log.Errorf("error marshalling feedback json")
				}
				err = t.feedback.Send(msg)
				if err != nil {
					log.Errorf("error sending feedback request")
				}
			}
		}
	} else {
		log.Errorf("error parsing sdp media")
	}

	return t.pc.SetRemoteDescription(desc)
}

// AddICECandidate accepts an ICE candidate string and adds it to the existing set of candidates
func (t *WebRTCTransport) AddICECandidate(candidate webrtc.ICECandidateInit) error {
	return t.pc.AddICECandidate(candidate)
}

// OnICECandidate sets an event handler which is invoked when a new ICE candidate is found.
// Take note that the handler is gonna be called with a nil pointer when gathering is finished.
func (t *WebRTCTransport) OnICECandidate(f func(c *webrtc.ICECandidate)) {
	t.pc.OnICECandidate(f)
}
