package elements

import (
	"errors"

	avp "github.com/pion/ion-avp/pkg"
	log "github.com/pion/ion-log"
)

// Types for samples
const (
	TypeMetadata = 100
	TypeBinary   = 101
	TypeRGB24    = 102
	TypeWebM     = 103
	TypeYCbCr    = 104
	TypeJPEG     = 105
	TypeRGBA     = 106
)

var (
	// ErrAttachNotSupported returned when attaching elements is not supported
	ErrAttachNotSupported = errors.New("attach not supported")
	// ErrElementAlreadyAttached returned when attaching an element that is already attached
	ErrElementAlreadyAttached = errors.New("element already attached")
)

type Node struct {
	children []avp.Element
}

func (e *Node) Write(sample *avp.Sample) error {
	for _, el := range e.children {
		err := el.Write(sample)
		if err != nil {
			return (err)
		}
	}
	return nil
}

func (e *Node) Attach(el avp.Element) {
	e.children = append(e.children, el)
}

func (e *Node) Close() {
	for _, el := range e.children {
		el.Close()
	}
}

type Leaf struct{}

func (e *Leaf) Write(sample *avp.Sample) error {
	log.Warnf("Write not implemented")
	return nil
}

func (e *Leaf) Attach(el avp.Element) {
	log.Warnf("Leaf nodes do not supported Attach()")
}

func (e *Leaf) Close() {}

type Pipeline struct {
	head avp.Element
	tail avp.Element
}

func NewPipeline(elements []avp.Element) *Pipeline {
	cur := elements[0]
	p := &Pipeline{head: cur}

	for i := 1; i < len(elements); i++ {
		cur.Attach(elements[i])
		cur = elements[i]
	}

	p.tail = cur

	return p
}

func (p *Pipeline) Write(sample *avp.Sample) error {
	return p.head.Write(sample)
}

func (p *Pipeline) Attach(el avp.Element) {
	p.tail.Attach(el)
}

func (p *Pipeline) Close() {
	p.head.Close()
}
