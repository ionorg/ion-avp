package elements

import (
	"bytes"
	"testing"

	"github.com/at-wat/ebml-go"
	"github.com/at-wat/ebml-go/webm"
	avp "github.com/pion/ion-avp/pkg"
	"github.com/stretchr/testify/assert"
)

// BufWriter instance
type BufWriter struct {
	buf bytes.Buffer
}

// BufWriter instance
func NewBufWriter() *BufWriter {
	return &BufWriter{}
}
func (w *BufWriter) ID() string {
	return ""
}
func (w *BufWriter) Write(sample *avp.Sample) error {
	_, err := w.buf.Write(sample.Payload.([]byte))
	return err
}
func (w *BufWriter) Read() <-chan *avp.Sample {
	return nil
}
func (w *BufWriter) Attach(e avp.Element) error {
	return ErrAttachNotSupported
}
func (w *BufWriter) Close() {}

type Header struct {
	Header  webm.EBMLHeader `ebml:"EBML"`
	Segment webm.Segment    `ebml:"Segment,size=unknown"`
}

func TestWebMSaver_BlockWriterInit(t *testing.T) {
	saver := NewWebmSaver(WebmSaverConfig{
		ID: "id",
	})

	writer := NewBufWriter()
	err := saver.Attach(writer)
	assert.NoError(t, err)

	// Construct keyframe packet
	rawKeyframePkt := []byte{
		0x90, 0xe0, 0x69, 0x8f, 0xd9, 0xc2, 0x93, 0xda, 0x1c, 0x64,
		0x27, 0x82, 0x00, 0x01, 0x00, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0x98, 0x36, 0xbe, 0x88, 0x9e,
	}

	err = saver.Write(&avp.Sample{
		Type:    avp.TypeVP8,
		Payload: rawKeyframePkt,
	})
	assert.NoError(t, err)

	var header Header
	err = ebml.Unmarshal(bytes.NewReader(writer.buf.Bytes()), &header)
	assert.NoError(t, err)

	assert.NotNil(t, header.Header)

	saver.Close()
}
