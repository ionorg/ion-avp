package elements

import (
	avp "github.com/pion/ion-avp/pkg"
)

// Filter instance
type Filter struct {
	Node
	condition func(*avp.Sample) bool
}

// NewFilter instance. Filter contitionally forwards
// a payload based on the return of the provided function.
func NewFilter(condition func(*avp.Sample) bool) *Filter {
	return &Filter{
		condition: condition,
	}
}

func (f *Filter) Write(sample *avp.Sample) error {
	if f.condition(sample) {
		return f.Node.Write(sample)
	}
	return nil
}
