package elements

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"

	avp "github.com/pion/ion-avp/pkg"
)

// Converter instance
type Converter struct {
	Node
	typ int
}

// NewConverter instance. Converter converts between
// media types.
//
// Currently supports:
//     - YCbCR -> JPEG
func NewConverter(typ int) *Converter {
	return &Converter{
		typ: typ,
	}
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

	return c.Node.Write(&avp.Sample{
		Type:    c.typ,
		Payload: out,
	})
}
