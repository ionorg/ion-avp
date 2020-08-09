package avp

import (
	"sync"
	"time"

	pb "github.com/pion/ion-avp/cmd/server/grpc/proto"
	"github.com/pion/ion-avp/pkg/log"
	"github.com/pion/ion-avp/pkg/samples"
	"github.com/pion/ion-sfu/pkg/rtc/transport"
)

const (
	statCycle = 3 * time.Second
)

// AVP represents an avp instance
type AVP struct {
	config       PipelineConfig
	pipelines    map[string]*Pipeline
	pipelineLock sync.RWMutex

	stop bool
}

// NewAVP creates a new avp instance
func NewAVP(c Config, getDefaultElements func(id string) map[string]Element, getTogglableElement func(e *pb.Element) (Element, error)) *AVP {
	log.Init(c.Log.Level)

	return &AVP{
		config: PipelineConfig{
			SampleBuilder: samples.BuilderConfig{
				AudioMaxLate: c.Pipeline.SampleBuilder.AudioMaxLate,
				VideoMaxLate: c.Pipeline.SampleBuilder.VideoMaxLate,
			},
			GetDefaultElements:  getDefaultElements,
			GetTogglableElement: getTogglableElement,
		},
		pipelines: make(map[string]*Pipeline),
	}
}

func (a *AVP) addPipeline(id string, pub transport.Transport) *Pipeline {
	log.Infof("process.addPipeline id=%s", id)
	a.pipelineLock.Lock()
	defer a.pipelineLock.Unlock()
	a.pipelines[id] = NewPipeline(id, a.config, pub)
	return a.pipelines[id]
}

// GetPipeline get pipeline from map
func (a *AVP) GetPipeline(id string) *Pipeline {
	log.Infof("process.GetPipeline id=%s", id)
	a.pipelineLock.RLock()
	defer a.pipelineLock.RUnlock()
	return a.pipelines[id]
}

// DelPipeline delete pub
func (a *AVP) DelPipeline(id string) {
	log.Infof("DelPipeline id=%s", id)
	pipeline := a.GetPipeline(id)
	if pipeline == nil {
		return
	}
	pipeline.Close()
	a.pipelineLock.Lock()
	defer a.pipelineLock.Unlock()
	delete(a.pipelines, id)
}

// Close close all pipelines
func (a *AVP) Close() {
	if a.stop {
		return
	}
	a.stop = true
	a.pipelineLock.Lock()
	defer a.pipelineLock.Unlock()
	for id, pipeline := range a.pipelines {
		if pipeline != nil {
			pipeline.Close()
			delete(a.pipelines, id)
		}
	}
}

// check show all pipelines' stat
func (a *AVP) check() {
	t := time.NewTicker(statCycle)
	for range t.C {
		info := "\n----------------process-----------------\n"
		a.pipelineLock.Lock()
		if len(a.pipelines) == 0 {
			a.pipelineLock.Unlock()
			continue
		}

		for id, pipeline := range a.pipelines {
			if !pipeline.Alive() {
				pipeline.Close()
				delete(a.pipelines, id)
				log.Infof("Stat delete %v", id)
			}
			info += "pipeline: " + id + "\n"
		}
		a.pipelineLock.Unlock()
		log.Infof(info)
	}
}
