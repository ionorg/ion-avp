package avp

import (
	pb "github.com/pion/ion-avp/cmd/server/grpc/proto"
	"github.com/pion/ion-avp/pkg/log"
	"github.com/pion/ion-avp/pkg/samples"
)

// AVP represents an avp instance
type AVP struct {
}

// NewAVP creates a new avp instance
func NewAVP(c Config, getDefaultElements func(id string) map[string]Element, getTogglableElement func(e *pb.Element) (Element, error)) *AVP {
	log.Init(c.Log.Level)

	if err := InitRTP(c.Rtp.Port, c.Rtp.KcpKey, c.Rtp.KcpSalt); err != nil {
		panic(err)
	}

	InitPipeline(PipelineConfig{
		SampleBuilder: samples.BuilderConfig{
			AudioMaxLate: c.Pipeline.SampleBuilder.AudioMaxLate,
			VideoMaxLate: c.Pipeline.SampleBuilder.VideoMaxLate,
		},
		GetDefaultElements:  getDefaultElements,
		GetTogglableElement: getTogglableElement,
	})

	return &AVP{}
}
