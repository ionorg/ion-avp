package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"os"

	pb "github.com/pion/ion-avp/cmd/server/grpc/proto"
	avp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/log"
	sfu "github.com/pion/ion-sfu/cmd/server/grpc/proto"
	"github.com/pion/webrtc/v3"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	conf = avp.Config{}
	file string
)

type server struct {
	pb.UnimplementedAVPServer
	avp *avp.AVP
}

// func getDefaultElements(id string) map[string]avp.Element {
// 	de := make(map[string]avp.Element)
// 	if conf.Pipeline.WebmSaver.Enabled && conf.Pipeline.WebmSaver.DefaultOn {
// 		filewriter := elements.NewFileWriter(elements.FileWriterConfig{
// 			ID:   id,
// 			Path: path.Join(conf.Pipeline.WebmSaver.Path, fmt.Sprintf("%s.webm", id)),
// 		})
// 		webm := elements.NewWebmSaver(elements.WebmSaverConfig{
// 			ID: id,
// 		})
// 		err := webm.Attach(filewriter)
// 		if err != nil {
// 			log.Errorf("error attaching filewriter to webm %s", err)
// 		} else {
// 			de[elements.TypeWebmSaver] = webm
// 		}
// 	}
// 	return de
// }

// func getTogglableElement(e *pb.Element) (avp.Element, error) {
// 	switch e.Type {
// 	case elements.TypeWebmSaver:
// 		filewriter := elements.NewFileWriter(elements.FileWriterConfig{
// 			ID:   e.Mid,
// 			Path: path.Join(conf.Pipeline.WebmSaver.Path, fmt.Sprintf("%s.webm", e.Mid)),
// 		})
// 		webm := elements.NewWebmSaver(elements.WebmSaverConfig{
// 			ID: e.Mid,
// 		})
// 		err := webm.Attach(filewriter)
// 		if err != nil {
// 			log.Errorf("error attaching filewriter to webm %s", err)
// 			return nil, err
// 		}
// 		return webm, nil
// 	}

// 	return nil, errors.New("element not found")
// }

func showHelp() {
	fmt.Printf("Usage:%s {params}\n", os.Args[0])
	fmt.Println("      -c {config file}")
	fmt.Println("      -h (show help info)")
}

func load() bool {
	_, err := os.Stat(file)
	if err != nil {
		return false
	}

	viper.SetConfigFile(file)
	viper.SetConfigType("toml")

	err = viper.ReadInConfig()
	if err != nil {
		fmt.Printf("config file %s read failed. %v\n", file, err)
		return false
	}
	err = viper.GetViper().Unmarshal(&conf)
	if err != nil {
		fmt.Printf("config file %s loaded failed. %v\n", file, err)
		return false
	}

	fmt.Printf("config %s load ok!\n", file)
	return true
}

func parse() bool {
	flag.StringVar(&file, "c", "config.toml", "config file")
	help := flag.Bool("h", false, "help info")
	flag.Parse()
	if !load() {
		return false
	}

	if *help {
		showHelp()
		return false
	}
	return true
}

func main() {
	if !parse() {
		showHelp()
		os.Exit(-1)
	}

	lis, err := net.Listen("tcp", conf.GRPC.Port)
	if err != nil {
		log.Panicf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterAVPServer(s, &server{
		avp: avp.NewAVP(conf),
	})

	log.Infof("--- AVP Node Listening at %s ---", conf.GRPC.Port)

	if err := s.Serve(lis); err != nil {
		log.Panicf("failed to serve: %v", err)
	}
	select {}
}

func (s *server) Signal(stream pb.AVP_SignalServer) error {
	for {
		in, err := stream.Recv()

		if err != nil {
			if err == io.EOF {
				return nil
			}

			errStatus, _ := status.FromError(err)
			if errStatus.Code() == codes.Canceled {
				return nil
			}

			log.Errorf("signal error %v %v", errStatus.Message(), errStatus.Code())
			return err
		}

		switch payload := in.Payload.(type) {
		case *pb.SignalRequest_Join:
			go s.join(stream.Context(), payload.Join.Sfu, payload.Join.Sid)

		case *pb.SignalRequest_Element:
			// yo
		}
	}
}

func (s *server) join(ctx context.Context, addr, sid string) {
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

	t := s.avp.NewWebRTCTransport(sid, conf)

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
			continue
		}

		switch payload := res.Payload.(type) {
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
