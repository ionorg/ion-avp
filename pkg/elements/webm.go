package elements

import (
	"sync"

	"github.com/at-wat/ebml-go/mkvcore"
	"github.com/at-wat/ebml-go/webm"

	avp "github.com/pion/ion-avp/pkg"
	"github.com/pion/ion-avp/pkg/log"
)

const (
	// IDWebmSaver .
	IDWebmSaver = "WebmSaver"
)

// WebmSaverConfig .
type WebmSaverConfig struct {
	ID string
}

// WebmSaver Module for saving rtp streams to webm
type WebmSaver struct {
	sync.Mutex
	id                             string
	closed                         bool
	audioWriter, videoWriter       webm.BlockWriteCloser
	audioTimestamp, videoTimestamp uint32
	sampleWriter                   *SampleWriter
}

// NewWebmSaver Initialize a new webm saver
func NewWebmSaver(config WebmSaverConfig) *WebmSaver {
	return &WebmSaver{
		id:           config.ID,
		sampleWriter: NewSampleWriter(),
	}
}

// ID of element
func (s *WebmSaver) ID() string {
	return IDWebmSaver
}

// Write sample to webmsaver
func (s *WebmSaver) Write(sample *avp.Sample) error {
	if sample.Type == avp.TypeVP8 {
		s.pushVP8(sample)
	} else if sample.Type == avp.TypeOpus {
		s.pushOpus(sample)
	}
	return nil
}

// Attach attach a child element
func (s *WebmSaver) Attach(e avp.Element) error {
	return s.sampleWriter.Attach(e)
}

// Close Close the WebmSaver
func (s *WebmSaver) Close() {
	s.Lock()
	defer s.Unlock()
	log.Infof("WebmSaver.Close() => %s", s.id)

	if s.closed {
		return
	}

	s.closed = true

	if s.audioWriter != nil {
		if err := s.audioWriter.Close(); err != nil {
			log.Errorf("audio close err: %s", err)
		}
	}
	if s.videoWriter != nil {
		if err := s.videoWriter.Close(); err != nil {
			log.Errorf("video close err: %s", err)
		}
	}
}

func (s *WebmSaver) pushOpus(sample *avp.Sample) {
	if s.audioWriter != nil {
		if s.audioTimestamp == 0 {
			s.audioTimestamp = sample.Timestamp
		}
		t := (sample.Timestamp - s.audioTimestamp) / 48
		if _, err := s.audioWriter.Write(true, int64(t), sample.Payload.([]byte)); err != nil {
			log.Errorf("audio writer err: %s", err)
		}
	}
}

func (s *WebmSaver) pushVP8(sample *avp.Sample) {
	payload := sample.Payload.([]byte)
	// Read VP8 header.
	videoKeyframe := (payload[0]&0x1 == 0)

	if videoKeyframe {
		// Keyframe has frame information.
		raw := uint(payload[6]) | uint(payload[7])<<8 | uint(payload[8])<<16 | uint(payload[9])<<24
		width := int(raw & 0x3FFF)
		height := int((raw >> 16) & 0x3FFF)

		if s.videoWriter == nil || s.audioWriter == nil {
			// Initialize WebM saver using received frame size.
			s.initWriter(width, height)
		}
	}

	if s.videoWriter != nil {
		if s.videoTimestamp == 0 {
			s.videoTimestamp = sample.Timestamp
		}
		t := (sample.Timestamp - s.videoTimestamp) / 90
		if _, err := s.videoWriter.Write(videoKeyframe, int64(t), payload); err != nil {
			log.Errorf("video write err: %s", err)
		}
	}
}

func (s *WebmSaver) initWriter(width, height int) {
	ws, err := webm.NewSimpleBlockWriter(s.sampleWriter,
		[]webm.TrackEntry{
			{
				Name:            "Audio",
				TrackNumber:     1,
				TrackUID:        12345,
				CodecID:         "A_OPUS",
				TrackType:       2,
				DefaultDuration: 20000000,
				Audio: &webm.Audio{
					SamplingFrequency: 48000.0,
					Channels:          2,
				},
			}, {
				Name:            "Video",
				TrackNumber:     2,
				TrackUID:        67890,
				CodecID:         "V_VP8",
				TrackType:       1,
				DefaultDuration: 20000000,
				Video: &webm.Video{
					PixelWidth:  uint64(width),
					PixelHeight: uint64(height),
				},
			},
		}, mkvcore.WithSeekHead(true))
	if err != nil {
		log.Errorf("init writer err: %s", err)
	}
	log.Infof("WebM saver has started with video width=%d, height=%d\n", width, height)
	s.audioWriter = ws[0]
	s.videoWriter = ws[1]
}

// SampleWriter for writing samples
type SampleWriter struct {
	children map[string]avp.Element
}

// NewSampleWriter creates a new sample writer
func NewSampleWriter() *SampleWriter {
	return &SampleWriter{
		children: make(map[string]avp.Element),
	}
}

// Attach a child element
func (w *SampleWriter) Attach(e avp.Element) error {
	if w.children[e.ID()] == nil {
		log.Infof("Transcribe.Attach element => %s", e.ID())
		w.children[e.ID()] = e
		return nil
	}
	return ErrElementAlreadyAttached
}

// Write sample
func (w *SampleWriter) Write(p []byte) (n int, err error) {
	for _, e := range w.children {
		sample := &avp.Sample{
			Type:    TypeBinary,
			Payload: p,
		}
		err := e.Write(sample)
		if err != nil {
			log.Errorf("SampleWriter.Write error => %s", err)
		}
	}
	return len(p), nil
}

// Close writer
func (w *SampleWriter) Close() error {
	for _, e := range w.children {
		e.Close()
	}
	return nil
}
