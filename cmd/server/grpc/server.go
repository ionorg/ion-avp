package server

import (
	"io"
	"net"

	pb "github.com/pion/ion-avp/cmd/server/grpc/proto"
	avp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedAVPServer
	avp *avp.AVP
}

// NewServer creates a new grpc avp server
func NewServer(addr string, conf avp.Config) *grpc.Server {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Panicf("failed to listen: %v", err)
	}
	s := grpc.NewServer()
	pb.RegisterAVPServer(s, &server{
		avp: avp.NewAVP(conf),
	})

	log.Infof("--- AVP Node Listening at %s ---", addr)

	if err := s.Serve(lis); err != nil {
		log.Panicf("failed to serve: %v", err)
	}

	return s
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

		switch payload := in.Payload.(type) {
		case *pb.SignalRequest_Join:
			go s.avp.Join(stream.Context(), payload.Join.Sfu, payload.Join.Sid)

		case *pb.SignalRequest_Process:
			s.avp.Process(payload.Process.Pid, payload.Process.Sid, payload.Process.Tid, payload.Process.Eid)
		}
	}
}
