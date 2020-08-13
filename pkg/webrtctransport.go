package avp

import (
	"fmt"
	"sync"

	"github.com/pion/ion-avp/pkg/log"
	"github.com/pion/ion-avp/pkg/samples"

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
	builders  map[string]*samples.Builder
	pending   map[string]string
	pipelines map[string]*Pipeline
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

	t := &WebRTCTransport{
		id:        id,
		pc:        pc,
		builders:  make(map[string]*samples.Builder),
		pending:   make(map[string]string),
		pipelines: make(map[string]*Pipeline),
	}

	pc.OnTrack(func(track *webrtc.Track, recv *webrtc.RTPReceiver) {
		log.Infof("Got track: %s", track.ID())
		builder := samples.NewBuilder(track, 200)
		t.mu.Lock()
		t.builders[track.ID()] = builder
		if pipeline := t.pending[track.ID()]; pipeline != "" {
			t.pipelines[pipeline].AddTrack(builder)
			delete(t.pending, track.ID())
		}
		t.mu.Unlock()
	})

	return t
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

// Process creates a pipeline
func (t *WebRTCTransport) Process(pid, tid, eid string) {
	log.Infof("WebRTCTransport.Process id=%s", pid)
	t.mu.Lock()
	defer t.mu.Unlock()
	p := t.pipelines[pid]
	if p == nil {
		e := registry.GetElement(eid)
		p = NewPipeline(e(t.id, pid, tid))
		t.pipelines[pid] = p
	}

	b := t.builders[tid]
	if b == nil {
		log.Debugf("builder not found for track %s. queuing.", tid)
		t.pending[tid] = pid
		return
	}

	p.AddTrack(b)
}

func (t *WebRTCTransport) stats() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	info := fmt.Sprintf("  session: %s\n", t.id)
	for _, pipeline := range t.pipelines {
		info += pipeline.stats()
	}

	if len(t.pending) > 0 {
		info += "  pending tracks:\n"
		for tid, pipeline := range t.pending {
			info += fmt.Sprintf("    track id: %s for pipeline: %s\n", tid, pipeline)
		}
	}

	return info
}
