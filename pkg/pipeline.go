package avp

import (
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
	element Element
	mu      sync.RWMutex
	stop    bool
}

// NewPipeline return a new Pipeline
func NewPipeline(e Element) *Pipeline {
	p := &Pipeline{
		element: e,
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
