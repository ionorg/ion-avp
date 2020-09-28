package avp

import (
	"context"
	"encoding/json"
	"io"
	"sync"

	"github.com/pion/ion-avp/pkg/log"
	sfu "github.com/pion/ion-sfu/cmd/server/grpc/proto"
	"github.com/pion/webrtc/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SFU client
type SFU struct {
	ctx        context.Context
	cancel     context.CancelFunc
	client     sfu.SFUClient
	config     Config
	mu         sync.RWMutex
	onCloseFn  func()
	transports map[string]*WebRTCTransport
}

// NewSFU intializes a new SFU client
func NewSFU(addr string, config Config) *SFU {
	log.Infof("Connecting to sfu: %s", addr)
	// Set up a connection to the sfu server.
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Errorf("did not connect: %v", err)
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	return &SFU{
		ctx:        ctx,
		cancel:     cancel,
		client:     sfu.NewSFUClient(conn),
		config:     config,
		transports: make(map[string]*WebRTCTransport),
	}
}

// GetTransport returns a webrtc transport for a session
func (s *SFU) GetTransport(sid string) *WebRTCTransport {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := s.transports[sid]

	// no transport yet, create one
	if t == nil {
		t = s.join(sid)
		t.OnClose(func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			delete(s.transports, sid)
			if len(s.transports) == 0 && s.onCloseFn != nil {
				s.cancel()
				s.onCloseFn()
			}
		})
		s.transports[sid] = t
	}

	return t
}

// OnClose handler called when sfu client is closed
func (s *SFU) OnClose(f func()) {
	s.onCloseFn = f
}

// Join creates an sfu client and join the session.
// All tracks will be relayed to the avp.
func (s *SFU) join(sid string) *WebRTCTransport {
	log.Infof("Joining sfu session: %s", sid)

	sfustream, err := s.client.Signal(s.ctx)

	if err != nil {
		log.Errorf("error creating sfu stream: %s", err)
		return nil
	}

	t := NewWebRTCTransport(sid, s.config)

	offer, err := t.CreateOffer()
	if err != nil {
		log.Errorf("Error creating offer: %v", err)
		return nil
	}

	if err = t.SetLocalDescription(offer); err != nil {
		log.Errorf("Error setting local description: %v", err)
		return nil
	}

	log.Debugf("Send offer:\n %s", offer.SDP)
	err = sfustream.Send(
		&sfu.SignalRequest{
			Payload: &sfu.SignalRequest_Join{
				Join: &sfu.JoinRequest{
					Sid: sid,
					Offer: &sfu.SessionDescription{
						Type: offer.Type.String(),
						Sdp:  []byte(offer.SDP),
					},
				},
			},
		},
	)

	if err != nil {
		log.Errorf("Error sending publish request: %v", err)
		return nil
	}

	t.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			// Gathering done
			return
		}
		bytes, err := json.Marshal(c.ToJSON())
		if err != nil {
			log.Errorf("OnIceCandidate error %s", err)
		}
		err = sfustream.Send(&sfu.SignalRequest{
			Payload: &sfu.SignalRequest_Trickle{
				Trickle: &sfu.Trickle{
					Init: string(bytes),
				},
			},
		})
		if err != nil {
			log.Errorf("OnIceCandidate error %s", err)
		}
	})

	go func() {
		// Handle sfu stream messages
		for {
			res, err := sfustream.Recv()

			if err != nil {
				if err == io.EOF {
					// WebRTC Transport closed
					log.Infof("WebRTC Transport Closed")
					err = sfustream.CloseSend()
					if err != nil {
						log.Errorf("error sending close: %s", err)
					}
					return
				}

				errStatus, _ := status.FromError(err)
				if errStatus.Code() == codes.Canceled {
					err = sfustream.CloseSend()
					if err != nil {
						log.Errorf("error sending close: %s", err)
					}
					return
				}

				log.Errorf("Error receiving signal response: %v", err)
				return
			}

			switch payload := res.Payload.(type) {
			case *sfu.SignalReply_Join:
				// Set the remote SessionDescription
				log.Debugf("got answer: %s", string(payload.Join.Answer.Sdp))
				if err = t.SetRemoteDescription(webrtc.SessionDescription{
					Type: webrtc.SDPTypeAnswer,
					SDP:  string(payload.Join.Answer.Sdp),
				}); err != nil {
					log.Errorf("join error %s", err)
					return
				}

			case *sfu.SignalReply_Negotiate:
				log.Debugf("got negotiate %s", payload.Negotiate.Type)
				if payload.Negotiate.Type == webrtc.SDPTypeOffer.String() {
					log.Debugf("got offer: %s", string(payload.Negotiate.Sdp))
					offer := webrtc.SessionDescription{
						Type: webrtc.SDPTypeOffer,
						SDP:  string(payload.Negotiate.Sdp),
					}

					// Peer exists, renegotiating existing peer
					err = t.SetRemoteDescription(offer)
					if err != nil {
						log.Errorf("negotiate error %s", err)
						continue
					}

					var answer webrtc.SessionDescription
					answer, err = t.CreateAnswer()
					if err != nil {
						log.Errorf("negotiate error %s", err)
						continue
					}

					err = t.SetLocalDescription(answer)
					if err != nil {
						log.Errorf("negotiate error %s", err)
						continue
					}

					err = sfustream.Send(&sfu.SignalRequest{
						Payload: &sfu.SignalRequest_Negotiate{
							Negotiate: &sfu.SessionDescription{
								Type: answer.Type.String(),
								Sdp:  []byte(answer.SDP),
							},
						},
					})

					if err != nil {
						log.Errorf("negotiate error %s", err)
						continue
					}
				} else if payload.Negotiate.Type == webrtc.SDPTypeAnswer.String() {
					log.Debugf("got answer: %s", string(payload.Negotiate.Sdp))
					err = t.SetRemoteDescription(webrtc.SessionDescription{
						Type: webrtc.SDPTypeAnswer,
						SDP:  string(payload.Negotiate.Sdp),
					})

					if err != nil {
						log.Errorf("negotiate error %s", err)
						continue
					}
				}
			case *sfu.SignalReply_Trickle:
				var candidate webrtc.ICECandidateInit
				_ = json.Unmarshal([]byte(payload.Trickle.Init), &candidate)
				err := t.AddICECandidate(candidate)
				if err != nil {
					log.Errorf("error adding ice candidate: %e", err)
				}
			}
		}
	}()

	return t
}

// show all sfu client stats
func (s *SFU) stats() string {
	info := "\n  sfu\n"

	s.mu.RLock()
	if len(s.transports) == 0 {
		s.mu.RUnlock()
		return info
	}

	for _, transport := range s.transports {
		info += transport.stats()
	}
	s.mu.RUnlock()
	return info
}
