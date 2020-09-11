package avp

import (
	"fmt"
	"sync"

	"github.com/pion/ion-avp/pkg/log"

	"github.com/pion/webrtc/v3"
)

// WebRTCTransportConfig represents configuration options
type WebRTCTransportConfig struct {
	configuration webrtc.Configuration
	setting       webrtc.SettingEngine
}

// WebRTCTransport represents a webrtc transport
type WebRTCTransport struct {
	id        string
	pc        *webrtc.PeerConnection
	mu        sync.RWMutex
	builders  map[string]*Builder         // one builder per track
	pending   map[string][]func() Element // maps track id to pending element constructors
	onCloseFn func()
}

// NewWebRTCTransport creates a new webrtc transport
func NewWebRTCTransport(id string, cfg WebRTCTransportConfig) *WebRTCTransport {
	// Create peer connection
	me := webrtc.MediaEngine{}
	me.RegisterDefaultCodecs()
	api := webrtc.NewAPI(webrtc.WithMediaEngine(me), webrtc.WithSettingEngine(cfg.setting))
	pc, err := api.NewPeerConnection(cfg.configuration)

	if err != nil {
		log.Errorf("Error creating peer connection: %s", err)
		return nil
	}

	_, err = pc.CreateDataChannel("feedback", nil)
	if err != nil {
		log.Errorf("Error creating peer data channel: %s", err)
		return nil
	}

	t := &WebRTCTransport{
		id:       id,
		pc:       pc,
		builders: make(map[string]*Builder),
		pending:  make(map[string][]func() Element),
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
				builder.AttachElement(p())
			}
			delete(t.pending, id)
		}

		builder.OnStop(func() {
			t.mu.Lock()
			b := t.builders[id]
			if b != nil {
				log.Debugf("stop builder %s", id)
				delete(t.builders, id)
			}

			if len(t.builders) == 0 && len(t.pending) == 0 {
				// No more tracks, cleanup transport
				t.mu.Unlock()
				t.Close()
				return
			}
			t.mu.Unlock()
		})
	})

	return t
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
func (t *WebRTCTransport) Process(pid, tid, eid string, config []byte) {
	log.Infof("WebRTCTransport.Process id=%s", pid)
	t.mu.Lock()
	defer t.mu.Unlock()

	e := registry.GetElement(eid)
	b := t.builders[tid]
	if b == nil {
		log.Debugf("builder not found for track %s. queuing.", tid)
		t.pending[tid] = append(t.pending[tid], func() Element { return e(t.id, pid, tid, config) })
		return
	}

	b.AttachElement(e(t.id, pid, tid, config))
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

func (t *WebRTCTransport) stats() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	info := fmt.Sprintf("    session: %s\n", t.id)
	for _, builder := range t.builders {
		info += builder.stats()
	}

	if len(t.pending) > 0 {
		info += "    pending tracks:\n"
		for tid := range t.pending {
			info += fmt.Sprintf("      track id: %s\n", tid)
		}
	}

	return info
}
