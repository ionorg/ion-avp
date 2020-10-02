package elements

import (
	avp "github.com/pion/ion-avp/pkg"
)

// Map instance
type Map struct {
	Node
	fn func(*avp.Sample) *avp.Sample
}

// NewMap creates a new sample mapper which
// maps samples using the provided function
func NewMap(fn func(*avp.Sample) *avp.Sample) *Map {
	return &Map{
		fn: fn,
	}
}

func (m *Map) Write(sample *avp.Sample) error {
	return m.Node.Write(m.fn(sample))
}
