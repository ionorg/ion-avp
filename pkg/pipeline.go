package avp

import (
	"fmt"
	"sync"

	"github.com/pion/ion-avp/pkg/log"
	"github.com/pion/ion-avp/pkg/samples"
)

// Pipeline constructs a processing graph
//
//          +--->elementCh--->element
//          |
// builder--+--->elementCh--->element
//          |
//          +--->elementCh--->element
type Pipeline struct {
	element  Element
	builders map[string]*samples.Builder
	mu       sync.RWMutex
	stop     bool
}

// NewPipeline return a new Pipeline
func NewPipeline(e Element) *Pipeline {
	p := &Pipeline{
		element:  e,
		builders: make(map[string]*samples.Builder),
	}

	return p
}

func (p *Pipeline) start(builder *samples.Builder) {
	for {
		p.mu.RLock()
		if p.stop {
			p.mu.RUnlock()
			return
		}
		p.mu.RUnlock()

		sample := builder.Read()
		err := p.element.Write(sample)
		if err != nil {
			log.Errorf("error writing sample: %s", err)
		}
	}
}

// AddTrack to pipeline
func (p *Pipeline) AddTrack(builder *samples.Builder) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.builders[builder.Track().ID()] = builder
	go p.start(builder)
}

// Stop a pipeline
func (p *Pipeline) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.stop {
		return
	}
	p.stop = true
}

func (p *Pipeline) stats() string {
	info := fmt.Sprintf("    element: %s\n", p.element.ID())
	for id := range p.builders {
		info += fmt.Sprintf("      track id: %s", id)
	}
	return info
}
