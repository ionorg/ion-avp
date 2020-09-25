package elements

import (
	"bytes"
	"sync"
	"testing"

	"github.com/at-wat/ebml-go"
	"github.com/at-wat/ebml-go/webm"
	avp "github.com/pion/ion-avp/pkg"
	"github.com/stretchr/testify/assert"
)

// BufWriter instance
type BufWriter struct {
	sync.Mutex
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
	w.Lock()
	defer w.Unlock()
	_, err := w.buf.Write(sample.Payload.([]byte))
	return err
}
func (w *BufWriter) Attach(e avp.Element) error {
	return ErrAttachNotSupported
}
func (w *BufWriter) Close() {}

type Header struct {
	Header  webm.EBMLHeader `ebml:"EBML"`
	Segment webm.Segment    `ebml:"Segment,size=unknown"`
}

// VP8 keyframe packet
var rawKeyframePkt = []byte{
	0x90, 0xe0, 0x69, 0x8f, 0xd9, 0xc2, 0x93, 0xda, 0x1c, 0x64,
	0x27, 0x82, 0x00, 0x01, 0x00, 0x01, 0xFF, 0xFF, 0xFF, 0xFF, 0x98, 0x36, 0xbe, 0x88, 0x9e,
}

var rawOpusPkt = []byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x90, 0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x90}

func TestWebMSaver_BlockWriterInit(t *testing.T) {
	saver := NewWebmSaver(WebmSaverConfig{
		ID: "id",
	})

	writer := NewBufWriter()
	err := saver.Attach(writer)
	assert.NoError(t, err)

	err = saver.Write(&avp.Sample{
		Type:    avp.TypeVP8,
		Payload: rawKeyframePkt,
	})
	assert.NoError(t, err)

	err = saver.Write(&avp.Sample{
		Type:    avp.TypeOpus,
		Payload: rawOpusPkt,
	})
	assert.NoError(t, err)

	var header Header
	writer.Lock()
	err = ebml.Unmarshal(bytes.NewReader(writer.buf.Bytes()), &header)
	assert.NoError(t, err)
	writer.Unlock()

	assert.NotNil(t, header.Header)

	saver.Close()

	assert.Len(t, header.Segment.Tracks.TrackEntry, 2)
}
