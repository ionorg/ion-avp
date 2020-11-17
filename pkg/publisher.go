package avp

import (
	"sync"

	log "github.com/pion/ion-log"

	"github.com/pion/webrtc/v3"
)

type Publisher struct {
	sync.RWMutex

	id string
	pc *webrtc.PeerConnection

	candidates []webrtc.ICECandidateInit

	closeOnce sync.Once
}

// NewPublisher creates a new Publisher
func NewPublisher(id string, cfg WebRTCTransportConfig) (*Publisher, error) {
	api := webrtc.NewAPI(webrtc.WithSettingEngine(cfg.setting))
	pc, err := api.NewPeerConnection(cfg.configuration)

	if err != nil {
		log.Errorf("NewPeer error: %v", err)
		return nil, errPeerConnectionInitFailed
	}

	_, err = pc.CreateDataChannel("ion-sfu", &webrtc.DataChannelInit{})

	if err != nil {
		log.Errorf("error creating data channel: %v", err)
		return nil, errPeerConnectionInitFailed
	}

	s := &Publisher{
		id: id,
		pc: pc,
	}

	pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		log.Debugf("ice connection state: %s", connectionState)
		switch connectionState {
		case webrtc.ICEConnectionStateFailed:
			fallthrough
		case webrtc.ICEConnectionStateClosed:
			s.closeOnce.Do(func() {
				log.Debugf("webrtc ice closed for peer: %s", s.id)
				if err := s.Close(); err != nil {
					log.Errorf("webrtc transport close err: %v", err)
				}
			})
		}
	})

	return s, nil
}

func (s *Publisher) CreateOffer() (webrtc.SessionDescription, error) {
	offer, err := s.pc.CreateOffer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	err = s.pc.SetLocalDescription(offer)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	return offer, nil
}

// OnICECandidate handler
func (s *Publisher) OnICECandidate(f func(c *webrtc.ICECandidate)) {
	s.pc.OnICECandidate(f)
}

// AddICECandidate to peer connection
func (s *Publisher) AddICECandidate(candidate webrtc.ICECandidateInit) error {
	if s.pc.RemoteDescription() != nil {
		return s.pc.AddICECandidate(candidate)
	}
	s.candidates = append(s.candidates, candidate)
	return nil
}

// SetRemoteDescription sets the SessionDescription of the remote peer
func (s *Publisher) SetRemoteDescription(desc webrtc.SessionDescription) error {
	err := s.pc.SetRemoteDescription(desc)
	if err != nil {
		log.Errorf("SetRemoteDescription error: %v", err)
		return err
	}

	return nil
}

// Close peer
func (s *Publisher) Close() error {
	return s.pc.Close()
}
