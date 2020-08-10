package avp

import (
	"errors"
	"sync"

	"github.com/pion/ion-avp/pkg/log"
	"github.com/pion/ion-avp/pkg/samples"
	"github.com/pion/webrtc/v3"
)

// Pipeline constructs a processing graph
//
//          +--->elementCh--->element
//          |
// builder--+--->elementCh--->element
//          |
//          +--->elementCh--->element
type Pipeline struct {
	builder      *samples.Builder
	elements     map[string]Element
	elementLock  sync.RWMutex
	elementChans map[string]chan *samples.Sample
	stop         bool
}

// NewPipeline return a new Pipeline
func NewPipeline(track *webrtc.Track) *Pipeline {
	log.Infof("NewPipeline for track ssrc: %d", track.SSRC())

	p := &Pipeline{
		builder:      samples.NewBuilder(track, 200),
		elements:     make(map[string]Element),
		elementChans: make(map[string]chan *samples.Sample),
	}

	go p.start()

	return p
}

func (p *Pipeline) start() {
	for {
		if p.stop {
			return
		}

		sample := p.builder.Read()

		p.elementLock.RLock()
		// Push to client send queues
		for _, element := range p.elements {
			go func(element Element) {
				err := element.Write(sample)
				if err != nil {
					log.Errorf("element.Write err=%v", err)
				}
			}(element)
		}
		p.elementLock.RUnlock()
	}
}

// AddElement add a element to pipeline
func (p *Pipeline) AddElement(e Element) error {
	if p.elements[e.Type()] != nil {
		return errors.New("element already exists")
	}

	p.elementLock.Lock()
	defer p.elementLock.Unlock()
	p.elements[e.Type()] = e
	p.elementChans[e.Type()] = make(chan *samples.Sample, 100)
	log.Infof("Pipeline.AddElement type=%s", e.Type())
	return nil
}

// DelElement del node by id
func (p *Pipeline) DelElement(id string) {
	log.Infof("Pipeline.DelElement id=%s", id)
	p.elementLock.Lock()
	defer p.elementLock.Unlock()
	if p.elements[id] != nil {
		p.elements[id].Close()
	}
	if p.elementChans[id] != nil {
		close(p.elementChans[id])
	}
	delete(p.elements, id)
	delete(p.elementChans, id)
}

func (p *Pipeline) delElements() {
	p.elementLock.RLock()
	ids := make([]string, 0, len(p.elements))
	for id := range p.elements {
		ids = append(ids, id)
	}
	p.elementLock.RUnlock()

	for _, id := range ids {
		p.DelElement(id)
	}
}

// Close release all
func (p *Pipeline) Close() {
	if p.stop {
		return
	}
	log.Infof("Pipeline.Close")
	p.stop = true
	p.delElements()
}
