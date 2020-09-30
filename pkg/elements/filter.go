package elements

import (
	avp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/log"
)

// Filter instance
type Filter struct {
	id        string
	condition func(*avp.Sample) bool
	children  []avp.Element
}

// NewFilter instance. Filter contitionally forwards
// a payload based on the return of the provided function.
func NewFilter(id string, condition func(*avp.Sample) bool) *Filter {
	f := &Filter{
		id:        id,
		condition: condition,
	}

	log.Infof("NewFilter with id: %s", id)

	return f
}

func (f *Filter) Write(sample *avp.Sample) error {
	if f.condition(sample) {
		for _, e := range f.children {
			err := e.Write(sample)
			if err != nil {
				return (err)
			}
		}
	}

	return nil
}

// Attach attach a child element
func (f *Filter) Attach(e avp.Element) {
	f.children = append(f.children, e)
}

// Close Filter
func (f *Filter) Close() {
	for _, e := range f.children {
		e.Close()
	}
}
