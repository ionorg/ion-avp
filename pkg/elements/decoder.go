package elements

import (
	"bytes"

	avp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/log"
	"golang.org/x/image/vp8"
)

// DecoderConfig .
type DecoderConfig struct {
	ID string `json:"id"`
}

// Decoder instance
type Decoder struct {
	id       string
	decoder  *vp8.Decoder
	children []avp.Element
}

// NewDecoder instance. Decoder takes as input VP8 keyframes
// and decodes it into a YCbCr image.
func NewDecoder(config DecoderConfig) *Decoder {
	w := &Decoder{
		id:      config.ID,
		decoder: vp8.NewDecoder(),
	}

	log.Infof("NewDecoder with config: %+v", config)

	return w
}

func (d *Decoder) Write(sample *avp.Sample) error {
	if sample.Type == avp.TypeVP8 {
		payload := sample.Payload.([]byte)
		videoKeyframe := (payload[0]&0x1 == 0)
		if !videoKeyframe {
			return nil
		}

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

		for _, e := range d.children {
			sample := &avp.Sample{
				Type:    TypeYCbCr,
				Payload: img,
			}
			err := e.Write(sample)
			if err != nil {
				return (err)
			}
		}
	}

	return nil
}

// Attach attach a child element
func (d *Decoder) Attach(e avp.Element) {
	d.children = append(d.children, e)
}

// Close Decoder
func (d *Decoder) Close() {
	log.Infof("Decoder.Close() %s", d.id)
}
