package elements

import (
	"time"

	avp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/log"
	"github.com/xlab/libvpx-go/vpx"
)

// Encoder instance
type Encoder struct {
	Node
	ctx   *vpx.CodecCtx
	iface *vpx.CodecIface
	run   bool
	async bool
}

// NewEncoder instance. Encoder takes as input Images
// and encodes it into a vpx stream.
func NewEncoder(fps uint) *Encoder {
	dec := &Encoder{
		ctx: vpx.NewCodecCtx(),
	}

	if fps > 0 {
		dec.async = true
		go dec.producer(fps)
	}

	return dec
}

func (dec *Encoder) Write(sample *avp.Sample) error {
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
			dec.iface = vpx.EncoderIfaceVP8()
			err := vpx.Error(vpx.CodecDecInitVer(dec.ctx, dec.iface, nil, 0, vpx.EncoderABIVersion))
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

func (dec *Encoder) Close() {
	dec.run = false
	dec.Node.Close()
}

func (dec *Encoder) write() error {
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

func (dec *Encoder) producer(fps uint) {
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
