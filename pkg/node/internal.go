package avp

import (
	"context"
	"errors"

	"github.com/pion/ion-avp/pkg/log"
	"github.com/pion/ion-avp/pkg/process"

	pb "github.com/pion/ion-avp/pkg/proto/avp"
)

func (s *server) StartProcess(ctx context.Context, in *pb.StartProcessRequest) (*pb.StartProcessReply, error) {
	log.Infof("process einfo=%v", einfo)
	pipeline := process.GetPipeline(einfo.MID)
	if pipeline == nil {
		return nil, errors.New("process: pipeline not found")
	}
	pipeline.AddElement(einfo)
	return &pb.StartProcessReply{}, nil
}

func (s *server) StopProcess(ctx context.Context, in *pb.StopProcessRequest) (*pb.StopProcessReply, error) {
	log.Infof("publish unprocess=%v", msg)
	return &pb.StopProcessReply{}, nil
}
