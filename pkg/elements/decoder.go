// +build libvpx

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
	typ   int
	run   bool
	async bool
}

// NewDecoder instance. Decoder takes as input VPX streams
// and decodes it into a YCbCr image.
func NewDecoder(fps uint, outType int) *Decoder {
	dec := &Decoder{
		ctx: vpx.NewCodecCtx(),
		typ: outType,
	}

	if fps > 0 {
		dec.async = true
		go dec.producer(fps)
	}

	return dec
}

func (dec *Decoder) Write(sample *avp.Sample) error {
	if sample.Type == avp.TypeVP8 {
		payload := sample.Payload.([]byte)

		if !dec.run {
			videoKeyframe := (payload[0]&0x1 == 0)
			if !videoKeyframe {
				return nil
			}
			dec.run = true
		}

		if dec.iface == nil {
			dec.iface = vpx.DecoderIfaceVP8()
			err := vpx.Error(vpx.CodecDecInitVer(dec.ctx, dec.iface, nil, 0, vpx.DecoderABIVersion))
			if err != nil {
				log.Errorf("%s", err)
			}
		}

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
	dec.run = false
	dec.Node.Close()
}

func (dec *Decoder) write() error {
	var iter vpx.CodecIter
	img := vpx.CodecGetFrame(dec.ctx, &iter)
	for img != nil {
		img.Deref()

		if dec.typ == TypeYCbCr {
			return dec.Node.Write(&avp.Sample{
				Type:    TypeYCbCr,
				Payload: img.ImageYCbCr(),
			})
		} else if dec.typ == TypeRGBA {
			return dec.Node.Write(&avp.Sample{
				Type:    TypeRGBA,
				Payload: img.ImageRGBA(),
			})
		}
	}
	return nil
}

func (dec *Decoder) producer(fps uint) {
	ticker := time.NewTicker(time.Duration((1/fps)*1000) * time.Millisecond)
	for range ticker.C {
		if !dec.run {
			return
		}

		err := dec.write()
		if err != nil {
			log.Errorf("%s", err)
		}
	}
}
