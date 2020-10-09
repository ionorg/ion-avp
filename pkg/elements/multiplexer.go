package elements

import (
	"github.com/lucsky/cuid"
	avp "github.com/pion/ion-avp/pkg"
)

type Multiplexer struct {
	id    string
	el    avp.Element
	demux avp.Element
}

func NewMultiplexer(el avp.Element) avp.Element {
	id := cuid.New()
	demux := NewFilter(func(sample *avp.Sample) bool {
		return sample.ID == id
	})
	el.Attach(demux)
	return &Multiplexer{
		id:    id,
		demux: demux,
		el:    el,
	}
}

func (m *Multiplexer) Write(sample *avp.Sample) error {
	sample.ID = m.id
	err := m.el.Write(sample)
	if err != nil {
		return (err)
	}
	return nil
}

func (m *Multiplexer) Attach(el avp.Element) {
	m.demux.Attach(el)
}

func (m *Multiplexer) Close() {
	m.demux.Close()
}
