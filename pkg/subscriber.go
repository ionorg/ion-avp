package avp

import (
	"sync"

	log "github.com/pion/ion-log"
	"github.com/pion/webrtc/v3"
)

type Subscriber struct {
	pc             *webrtc.PeerConnection
	candidates     []webrtc.ICECandidateInit
	candidatesLock sync.Mutex

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
	s.addPendingICE()
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
	s.candidatesLock.Lock()
	s.candidates = append(s.candidates, candidate)
	s.candidatesLock.Unlock()
	return nil
}

func (s *Subscriber) addPendingICE() {
	s.candidatesLock.Lock()
	defer s.candidatesLock.Unlock()
	for _, c := range s.candidates {
		if err := s.pc.AddICECandidate(c); err != nil {
			log.Errorf("AddICECandidate from pending: %s. %v", err, c)
		}
	}
	s.candidates = s.candidates[:0]
}
