package server

import (
	"io"

	pb "github.com/pion/ion-avp/cmd/signal/grpc/proto"
	avp "github.com/pion/ion-avp/pkg"
	log "github.com/pion/ion-log"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedAVPServer
	avp *AVP
}

func NewAVPServer(conf avp.Config, elems map[string]avp.ElementFun) pb.AVPServer {
	return &server{
		avp: NewAVP(conf, elems),
	}
}

// Signal handler for avp server
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

		if payload, ok := in.Payload.(*pb.SignalRequest_Process); ok {
			if err = s.avp.Process(
				stream.Context(),
				payload.Process.Sfu,
				payload.Process.Pid,
				payload.Process.Sid,
				payload.Process.Tid,
				payload.Process.Eid,
				payload.Process.Config,
			); err != nil {
				log.Errorf("process error: %v", err)
			}
		}
	}
}
