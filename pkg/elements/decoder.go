package elements

import (
	"bytes"

	avp "github.com/pion/ion-avp/pkg"
	"golang.org/x/image/vp8"
)

// Decoder instance
type Decoder struct {
	Node
	decoder *vp8.Decoder
}

// NewDecoder instance. Decoder takes as input VP8 keyframes
// and decodes it into a YCbCr image.
func NewDecoder() *Decoder {
	return &Decoder{
		decoder: vp8.NewDecoder(),
	}
}

func (d *Decoder) Write(sample *avp.Sample) error {
	if sample.Type == avp.TypeVP8 {
		payload := sample.Payload.([]byte)

		d.decoder.Init(bytes.NewReader(payload), len(payload))

		// Decode header
		if _, err := d.decoder.DecodeFrameHeader(); err != nil {
			return err
		}

		// Decode Frame
		img, err := d.decoder.DecodeFrame()
		if err != nil {
			return err
		}

		return d.Node.Write(&avp.Sample{
			Type:    TypeYCbCr,
			Payload: img,
		})
	}

	return nil
}
