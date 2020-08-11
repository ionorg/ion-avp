package avp

import (
	"context"
	"encoding/json"
	"io"
	"sync"
	"time"

	"github.com/pion/ion-avp/pkg/log"
	sfu "github.com/pion/ion-sfu/cmd/server/grpc/proto"
	"github.com/pion/webrtc/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	statCycle = 5 * time.Second
)

var registry *Registry

// AVP represents an avp instance
type AVP struct {
	config     Config
	transports map[string]*WebRTCTransport
	mu         sync.RWMutex
}

// Init avp with a registry of elements
func Init(r *Registry) {
	registry = r
}

// NewAVP creates a new avp instance
func NewAVP(c Config) *AVP {
	log.Init(c.Log.Level)

	a := &AVP{
		config:     c,
		transports: make(map[string]*WebRTCTransport),
	}

	go a.stats()

	return a
}

// NewWebRTCTransport creates a new webrtctransport for a session
func (a *AVP) NewWebRTCTransport(id string, config Config) *WebRTCTransport {
	t := NewWebRTCTransport(id, config)
	a.mu.Lock()
	a.transports[id] = t
	a.mu.Unlock()
	return t
}

// Join creates an sfu client and join the session.
// All tracks will be relayed to the avp.
func (a *AVP) Join(ctx context.Context, addr, sid string) {
	log.Infof("Joining sfu: %s session: %s", addr, sid)
	// Set up a connection to the sfu server.
	conn, err := grpc.Dial(addr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Errorf("did not connect: %v", err)
		return
	}
	c := sfu.NewSFUClient(conn)

	sfustream, err := c.Signal(ctx)

	if err != nil {
		log.Errorf("error creating sfu stream: %s", err)
		return
	}

	t := a.NewWebRTCTransport(sid, a.config)

	offer, err := t.CreateOffer()
	if err != nil {
		log.Errorf("Error creating offer: %v", err)
		return
	}

	if err = t.SetLocalDescription(offer); err != nil {
		log.Errorf("Error setting local description: %v", err)
		return
	}

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
		return
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

	// Handle sfu stream messages
	for {
		res, err := sfustream.Recv()

		if err != nil {
			if err == io.EOF {
				// WebRTC Transport closed
				log.Infof("WebRTC Transport Closed")
				err := sfustream.CloseSend()
				if err != nil {
					log.Errorf("error sending close: %s", err)
				}
				return
			}

			errStatus, _ := status.FromError(err)
			if errStatus.Code() == codes.Canceled {
				err := sfustream.CloseSend()
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
			if err = t.SetRemoteDescription(webrtc.SessionDescription{
				Type: webrtc.SDPTypeAnswer,
				SDP:  string(payload.Join.Answer.Sdp),
			}); err != nil {
				log.Errorf("join error %s", err)
				return
			}

		case *sfu.SignalReply_Negotiate:
			if payload.Negotiate.Type == webrtc.SDPTypeOffer.String() {
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

				answer, err := t.CreateAnswer()
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
}

// Process starts a process for a track.
func (a *AVP) Process(pid, sid, tid, eid string) {
	a.mu.RLock()
	t := a.transports[sid]
	a.mu.RUnlock()
	e := registry.GetElement(eid)
	if e == nil {
		log.Errorf("process err: element not found")
		return
	}
	t.Process(pid, tid, eid)
}

// show all avp stats
func (a *AVP) stats() {
	t := time.NewTicker(statCycle)
	for range t.C {
		info := "\n----------------stats-----------------\n"

		a.mu.RLock()
		if len(a.transports) == 0 {
			a.mu.RUnlock()
			continue
		}

		for _, transport := range a.transports {
			info += transport.stats()
		}
		a.mu.RUnlock()
		log.Infof(info)
		log.Infof(info)
	}
}
