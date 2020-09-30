package elements

import (
	"errors"

	avp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/log"
)

// Types for samples
const (
	TypeMetadata = 100
	TypeBinary   = 101
	TypeRGB24    = 102
	TypeWebM     = 103
	TypeYCbCr    = 104
	TypeJPEG     = 105
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
