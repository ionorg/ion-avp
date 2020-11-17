package avp

import (
	log "github.com/pion/ion-log"

	"github.com/pion/webrtc/v3"
)

type Publisher struct {
	pc         *webrtc.PeerConnection
	candidates []webrtc.ICECandidateInit
}

// NewPublisher creates a new Publisher
func NewPublisher(cfg WebRTCTransportConfig) (*Publisher, error) {
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

	return &Publisher{
		pc: pc,
	}, nil
}

func (p *Publisher) CreateOffer() (webrtc.SessionDescription, error) {
	offer, err := p.pc.CreateOffer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	err = p.pc.SetLocalDescription(offer)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}

	return offer, nil
}

// OnICECandidate handler
func (p *Publisher) OnICECandidate(f func(c *webrtc.ICECandidate)) {
	p.pc.OnICECandidate(f)
}

// AddICECandidate to peer connection
func (p *Publisher) AddICECandidate(candidate webrtc.ICECandidateInit) error {
	if p.pc.RemoteDescription() != nil {
		return p.pc.AddICECandidate(candidate)
	}
	p.candidates = append(p.candidates, candidate)
	return nil
}

// SetRemoteDescription sets the SessionDescription of the remote peer
func (p *Publisher) SetRemoteDescription(desc webrtc.SessionDescription) error {
	err := p.pc.SetRemoteDescription(desc)
	if err != nil {
		log.Errorf("SetRemoteDescription error: %v", err)
		return err
	}

	return nil
}

// Close peer
func (p *Publisher) Close() error {
	return p.pc.Close()
}
