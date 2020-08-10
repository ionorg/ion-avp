package avp

import (
	"fmt"
	"sync"

	"github.com/pion/ion-avp/pkg/log"

	"github.com/pion/webrtc/v3"
)

// WebRTCTransport represents a webrtc transport
type WebRTCTransport struct {
	id        string
	pc        *webrtc.PeerConnection
	mu        sync.RWMutex
	pipelines map[uint32]*Pipeline
}

// NewWebRTCTransport creates a new webrtc transport
func NewWebRTCTransport(id string, config Config) *WebRTCTransport {
	// Create peer connection
	pc, err := webrtc.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	})

	if err != nil {
		log.Errorf("Error creating peer connection: %s", err)
		return nil
	}

	t := &WebRTCTransport{
		id:        id,
		pc:        pc,
		pipelines: make(map[uint32]*Pipeline),
	}

	pc.OnTrack(func(track *webrtc.Track, recv *webrtc.RTPReceiver) {
		pipeline := NewPipeline(track)
		t.mu.Lock()
		t.pipelines[track.SSRC()] = pipeline
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

// AddElement add a processing element for track
func (t *WebRTCTransport) AddElement(ssrc uint32, e Element) {
	log.Infof("WebRTCTransport.AddElement type=%s", e.Type())
	t.mu.Lock()
	defer t.mu.Unlock()
	err := t.pipelines[ssrc].AddElement(e)
	if err != nil {
		log.Errorf("WebRTCTransport.AddElement err: %s", err)
	}
}

func (t *WebRTCTransport) stats() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	info := fmt.Sprintf("  peer: %s\n", t.id)
	// for _, router := range t.routers {
	// 	info += router.stats()
	// }

	return info
}
