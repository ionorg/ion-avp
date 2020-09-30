package elements

import (
	"time"

	avp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/log"
	"github.com/xlab/libvpx-go/vpx"
)

// Decoder instance
type Decoder struct {
	Node
	ctx   *vpx.CodecCtx
	iface *vpx.CodecIface
	async bool
}

// NewDecoder instance. Decoder takes as input VPX streams
// and decodes it into a YCbCr image.
func NewDecoder(fps uint) *Decoder {
	dec := &Decoder{
		ctx: vpx.NewCodecCtx(),
	}

	if fps > 0 {
		dec.async = true
		go dec.producer(fps)
	}

	return dec
}

func (dec *Decoder) Write(sample *avp.Sample) error {
	if sample.Type == avp.TypeVP8 {
		if dec.iface == nil {
			dec.iface = vpx.DecoderIfaceVP8()
			err := vpx.Error(vpx.CodecDecInitVer(dec.ctx, dec.iface, nil, 0, vpx.DecoderABIVersion))
			if err != nil {
				log.Errorf("%s", err)
			}
		}

		payload := sample.Payload.([]byte)
		err := vpx.Error(vpx.CodecDecode(dec.ctx, string(payload), uint32(len(payload)), nil, 0))
		if err != nil {
			return err
		}

		if !dec.async {
			return dec.write()
		}
	}

	return nil
}

func (dec *Decoder) Close() {
	dec.async = false
}

func (dec *Decoder) write() error {
	var iter vpx.CodecIter
	img := vpx.CodecGetFrame(dec.ctx, &iter)
	for img != nil {
		img.Deref()
		return dec.Node.Write(&avp.Sample{
			Type:    TypeYCbCr,
			Payload: img.ImageYCbCr(),
		})
	}
	return nil
}

func (dec *Decoder) producer(fps uint) {
	ticker := time.NewTicker(time.Duration(1/fps) * time.Second)
	for range ticker.C {
		if !dec.async {
			return
		}

		err := dec.write()
		if err != nil {
			log.Errorf("%s", err)
		}
	}
}
