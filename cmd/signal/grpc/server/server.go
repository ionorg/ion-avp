package server

import (
	"io"

	pb "github.com/pion/ion-avp/cmd/signal/grpc/proto"
	avp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/elements"
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

		switch payload := in.Payload.(type) {
		case *pb.SignalRequest_Process:
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

		case *pb.SignalRequest_RecordStart:
			cfg := payload.RecordStart.Cfg
			switch cfg.GetFormat() {
			case pb.RecordConfig_WEBM:
				webm := elements.NewWebmSaver(
					&elements.WebmSaverConfig{
						// TODO MONO vs STEREO
						Audio: cfg.GetAudio() != pb.RecordConfig_AUDIO_OFF,
						Video: cfg.GetVideo() == pb.RecordConfig_VIDEO_ON,
					},
				)
				filewriter := elements.NewFileWriter(
					cfg.GetFilename(),
					int(cfg.GetBuffersize()),
				)
				webm.Attach(filewriter)

				if err = s.avp.Run(
					payload.RecordStart.Sfu,
					payload.RecordStart.Sid,
					payload.RecordStart.Tid,
					webm,
				); err != nil {
					log.Errorf("RecordStart Run error: %v", err)
				}
			default:
				log.Errorf("RecordStart: unknown format %s", cfg.GetFormat())
			}

		case *pb.SignalRequest_RecordStop:
			err := s.avp.Stop(
				payload.RecordStop.Sfu,
				payload.RecordStop.Sid,
				payload.RecordStop.Tid,
			)
			if err != nil {
				log.Errorf("RecordStop error: %v", err)
			}
		}
	}
}
