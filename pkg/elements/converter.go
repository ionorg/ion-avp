package elements

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"

	avp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/log"
)

const (
	// IDConverter .
	IDConverter = "converter"
)

// ConverterConfig .
type ConverterConfig struct {
	ID   string `json:"id"`
	Type int    `json:"type"`
}

// Converter instance
type Converter struct {
	id       string
	typ      int
	children map[string]avp.Element
}

// NewConverter instance. Converter converts between
// media types.
//
// Currently supports:
//     - YCbCR -> JPEG
func NewConverter(config ConverterConfig) *Converter {
	w := &Converter{
		id:       config.ID,
		typ:      config.Type,
		children: make(map[string]avp.Element),
	}

	log.Infof("NewConverter with config: %+v", config)

	return w
}

// ID for Converter
func (d *Converter) ID() string {
	return IDConverter
}

func (d *Converter) Write(sample *avp.Sample) error {

	var out []byte
	switch sample.Type {
	case TypeYCbCr:
		payload := sample.Payload.(image.Image)
		switch d.typ {
		case TypeJPEG:
			buf := new(bytes.Buffer)
			if err := jpeg.Encode(buf, payload, nil); err != nil {
				return err
			}
			out = buf.Bytes()
		default:
			return errors.New("unsupported dest type")
		}
	default:
		return errors.New("unsupported source type")
	}

	for _, e := range d.children {
		sample := &avp.Sample{
			Type:    d.typ,
			Payload: out,
		}
		err := e.Write(sample)
		if err != nil {
			return (err)
		}
	}

	return nil
}

// Attach attach a child element
func (d *Converter) Attach(e avp.Element) error {
	if d.children[e.ID()] == nil {
		log.Infof("Converter.Attach element => %s", e.ID())
		d.children[e.ID()] = e
		return nil
	}
	return ErrElementAlreadyAttached
}

// Close Converter
func (d *Converter) Close() {
	log.Infof("Converter.Close() %s", d.id)
}
