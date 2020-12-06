package avp

import (
	log "github.com/pion/ion-log"
	"github.com/pion/webrtc/v3"
)

type Subscriber struct {
	pc         *webrtc.PeerConnection
	candidates []webrtc.ICECandidateInit

	onTrackFn func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver)
}

// NewSubscriber creates a new Subscriber
func NewSubscriber(cfg WebRTCTransportConfig) (*Subscriber, error) {
	me := webrtc.MediaEngine{}
	err := me.RegisterDefaultCodecs()
	if err != nil {
		log.Errorf("NewSubscriber error: %v", err)
		return nil, errPeerConnectionInitFailed
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(&me), webrtc.WithSettingEngine(cfg.setting))
	pc, err := api.NewPeerConnection(cfg.configuration)

	if err != nil {
		log.Errorf("NewSubscriber error: %v", err)
		return nil, errPeerConnectionInitFailed
	}

	s := &Subscriber{
		pc: pc,
	}

	pc.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		if s.onTrackFn != nil {
			s.onTrackFn(track, receiver)
		}
	})

	return s, nil
}

func (s *Subscriber) OnTrack(f func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver)) {
	s.onTrackFn = f
}

// Close the webrtc transport
func (s *Subscriber) Close() error {
	return s.pc.Close()
}

func (s *Subscriber) Answer(offer webrtc.SessionDescription) (webrtc.SessionDescription, error) {
	if err := s.pc.SetRemoteDescription(offer); err != nil {
		return webrtc.SessionDescription{}, err
	}
	answer, err := s.pc.CreateAnswer(nil)
	if err != nil {
		return webrtc.SessionDescription{}, err
	}
	if err := s.pc.SetLocalDescription(answer); err != nil {
		return webrtc.SessionDescription{}, err
	}
	return answer, nil
}

// OnICECandidate handler
func (s *Subscriber) OnICECandidate(f func(c *webrtc.ICECandidate)) {
	s.pc.OnICECandidate(f)
}

// AddICECandidate to peer connection
func (s *Subscriber) AddICECandidate(candidate webrtc.ICECandidateInit) error {
	if s.pc.RemoteDescription() != nil {
		return s.pc.AddICECandidate(candidate)
	}
	s.candidates = append(s.candidates, candidate)
	return nil
}
