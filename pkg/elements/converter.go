package elements

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"

	avp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/log"
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
	children []avp.Element
}

// NewConverter instance. Converter converts between
// media types.
//
// Currently supports:
//     - YCbCR -> JPEG
func NewConverter(config ConverterConfig) *Converter {
	w := &Converter{
		id:  config.ID,
		typ: config.Type,
	}

	log.Infof("NewConverter with config: %+v", config)

	return w
}

func (c *Converter) Write(sample *avp.Sample) error {
	var out []byte
	switch sample.Type {
	case TypeYCbCr:
		payload := sample.Payload.(image.Image)
		switch c.typ {
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

	for _, e := range c.children {
		sample := &avp.Sample{
			Type:    c.typ,
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
func (c *Converter) Attach(e avp.Element) {
	c.children = append(c.children, e)
}

// Close Converter
func (c *Converter) Close() {
	log.Infof("Converter.Close() %s", c.id)
}
