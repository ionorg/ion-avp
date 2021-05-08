package elements

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/at-wat/ebml-go/mkvcore"
	"github.com/at-wat/ebml-go/webm"

	avp "github.com/pion/ion-avp/pkg"
	log "github.com/pion/ion-log"
)

const (
	defaultWidth           = 640
	defaultHeight          = 480
	maxBufferedSamples     = 60 * 15 // 60 FPS for 15 seconds
	maxAudioVideoSyncDelay = time.Duration(15) * time.Second
)

// webmSaverStats keep track of statistics for the sake of logging
type webmSaverStats struct {
	audio                      int
	videoKey, videoInter       int
	droppedAudio, droppedVideo int
	unknown                    int
}

// WebmSaver Module for saving rtp streams to webm
type WebmSaver struct {
	sync.Mutex
	firstWrite                     bool
	dateUTC                        time.Time
	closed                         bool
	writeInProgress                int32
	audioWriter, videoWriter       webm.BlockWriteCloser
	vttAudioWriter, vttVideoWriter webm.BlockWriteCloser
	audioTimestamp, videoTimestamp uint32
	sampleWriter                   *SampleWriter
	preBuffering                   []*avp.Sample

	statsContext      string
	preBufferingStats webmSaverStats
	liveStats         webmSaverStats
}

// NewWebmSaver Initialize a new webm saver
func NewWebmSaver() *WebmSaver {
	return &WebmSaver{
		firstWrite:   true,
		sampleWriter: NewSampleWriter(),
		preBuffering: make([]*avp.Sample, 0, maxBufferedSamples),
	}
}

// SetStatsContext sets the logging information to display when writing
// stats to the log.
func (s *WebmSaver) SetStatsContext(context string) {
	s.statsContext = context
}

// Write sample to webmsaver
func (s *WebmSaver) Write(sample *avp.Sample) error {

	s.Lock()

	if s.firstWrite {
		s.dateUTC = time.Now()
		s.firstWrite = false
	}

	if s.closed {
		// already closed, ignore samples that are pending
		s.Unlock()
		return nil
	}

	atomic.StoreInt32(&(s.writeInProgress), 1)
	s.Unlock()

	if s.handlePrebuffer(sample) {
		atomic.StoreInt32(&(s.writeInProgress), 0)
		return nil
	}

	s.handleStats(sample, &s.liveStats)

	if sample.Type == avp.TypeVP8 {
		if sample.PrevDroppedPackets > 0 {
			s.pushVideoDropped(sample)
		}
		s.pushVP8(sample)
	} else if sample.Type == avp.TypeOpus {
		if sample.PrevDroppedPackets > 0 {
			s.pushAudioDropped(sample)
		}
		s.pushOpus(sample)
	}
	atomic.StoreInt32(&(s.writeInProgress), 0)
	return nil
}

func (s *WebmSaver) handlePrebuffer(sample *avp.Sample) bool {
	if s.preBuffering == nil {
		return false
	}

	s.handleStats(sample, &s.preBufferingStats)

	if sample != nil {
		s.preBuffering = append(s.preBuffering, sample)
	}

	initVideoNow := func(useWidth, useHeight int) {
		// Initialize WebM saver using received frame size.
		s.initWriter(useWidth, useHeight)

		preBuffering := s.preBuffering
		s.preBuffering = nil

		for _, bufferedSample := range preBuffering {
			s.Write(bufferedSample)
		}
	}

	if sample == nil || len(s.preBuffering) == cap(s.preBuffering) {
		initVideoNow(defaultWidth, defaultHeight)
		return true
	}

	if sample.Type != avp.TypeVP8 {
		return true
	}

	payload := sample.Payload.([]byte)
	if len(payload) < 10 {
		return true
	}

	// Read VP8 header.
	if payload[0]&0x1 != 0 {
		return true
	}

	// Keyframe has frame information.
	raw := uint(payload[6]) | uint(payload[7])<<8 | uint(payload[8])<<16 | uint(payload[9])<<24
	width := int(raw & 0x3FFF)
	height := int((raw >> 16) & 0x3FFF)

	initVideoNow(width, height)
	return true
}

func (s *WebmSaver) handleStats(sample *avp.Sample, useStats *webmSaverStats) {
	if len(s.statsContext) < 1 {
		return
	}

	report := func(value *int, total, mask int, message string) {
		wasZero := (*value) == 0
		(*value) += total
		if wasZero || (*value)&mask == 0x00 {
			log.Debugf("WebM saver stat: %d for %s, context=%s\n", (*value), message, s.statsContext)
		}
	}

	if nil == sample {
		report(&useStats.unknown, 1, 0xFF, "unknown nil samples")
		return
	}

	switch sample.Type {
	case avp.TypeOpus:
		if sample.PrevDroppedPackets > 0 {
			report(&useStats.droppedAudio, int(sample.PrevDroppedPackets), 0xFF, "audio dropped")
		}
		report(&useStats.audio, 1, 0xFF, "audio")
	case avp.TypeVP8:
		if sample.PrevDroppedPackets > 0 {
			report(&useStats.droppedVideo, int(sample.PrevDroppedPackets), 0xFF, "video dropped")
		}

		payload := sample.Payload.([]byte)

		videoKeyframe := (payload[0]&0x1 == 0)

		if videoKeyframe {
			report(&useStats.videoKey, 1, 0x3, "video key")
		} else {
			report(&useStats.videoInter, 1, 0x3F, "video")
		}
	default:
		report(&useStats.unknown, 1, 0xFF, "unknown samples")
	}
}

// Attach attach a child element
func (s *WebmSaver) Attach(e avp.Element) {
	s.sampleWriter.Attach(e)
}

// Close Close the WebmSaver
func (s *WebmSaver) Close() {

	// wait for any pending writes to complete
	for {
		var alreadyClosed bool
		var closed bool

		s.Lock()
		if s.closed {
			alreadyClosed = true
		}
		if atomic.LoadInt32(&(s.writeInProgress)) == 0 {
			s.closed = true
			closed = true
		}
		s.Unlock()

		if alreadyClosed {
			return
		}
		if closed {
			break
		}
		time.Sleep(time.Millisecond * 20)
	}

	s.handlePrebuffer(nil)

	s.closed = true

	if s.vttAudioWriter != nil {
		if err := s.vttAudioWriter.Close(); err != nil {
			log.Errorf("vtt audio close err: %s", err)
		}
	}
	if s.vttVideoWriter != nil {
		if err := s.vttVideoWriter.Close(); err != nil {
			log.Errorf("vtt video close err: %s", err)
		}
	}
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

func (s *WebmSaver) pushAudioDropped(sample *avp.Sample) {
	if s.vttAudioWriter != nil {
		var metaPayload [2]byte
		// big endian encoded value as two bytes
		metaPayload[0] = uint8(sample.PrevDroppedPackets >> 8)
		metaPayload[1] = uint8(sample.PrevDroppedPackets & 0xFF)

		referenceTimestamp := s.audioTimestamp
		if referenceTimestamp == 0 {
			referenceTimestamp = sample.Timestamp
		}

		t := (sample.Timestamp - referenceTimestamp) / 48
		if _, err := s.vttAudioWriter.Write(true, int64(t), metaPayload[:]); err != nil {
			log.Errorf("vtt audio writer err: %s", err)
		}
	}
}

func (s *WebmSaver) pushVideoDropped(sample *avp.Sample) {
	if s.vttAudioWriter != nil {
		var metaPayload [2]byte
		// big endian encoded value as two bytes
		metaPayload[0] = uint8(sample.PrevDroppedPackets >> 8)
		metaPayload[1] = uint8(sample.PrevDroppedPackets & 0xFF)

		referenceTimestamp := s.videoTimestamp
		if referenceTimestamp == 0 {
			referenceTimestamp = sample.Timestamp
		}

		t := (sample.Timestamp - referenceTimestamp) / 90
		if _, err := s.vttVideoWriter.Write(true, int64(t), metaPayload[:]); err != nil {
			log.Errorf("vtt video writer err: %s", err)
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
	useInterceptor := mkvcore.MustBlockInterceptor(mkvcore.NewMultiTrackBlockSorter(mkvcore.WithMaxTimescaleDelay(maxAudioVideoSyncDelay.Milliseconds()), mkvcore.WithSortRule(mkvcore.BlockSorterDropOutdated)))

	options := []mkvcore.BlockWriterOption{
		mkvcore.WithSegmentInfo(&webm.Info{
			TimecodeScale: webm.DefaultSegmentInfo.TimecodeScale,
			MuxingApp:     webm.DefaultSegmentInfo.MuxingApp,
			WritingApp:    webm.DefaultSegmentInfo.WritingApp,
			DateUTC:       s.dateUTC,
		}),
		mkvcore.WithSeekHead(true),
		mkvcore.WithBlockInterceptor(useInterceptor),
	}
	ws, err := webm.NewSimpleBlockWriter(s.sampleWriter,
		[]webm.TrackEntry{
			{
				Name:            "VttAudioDroppedPacketMeta",
				TrackNumber:     1,
				TrackUID:        98765,
				CodecID:         "D_WEBVTT/METADATA",
				TrackType:       0x21,
				DefaultDuration: 20000000,
			}, {
				Name:            "Audio",
				TrackNumber:     2,
				TrackUID:        12345,
				CodecID:         "A_OPUS",
				TrackType:       2,
				DefaultDuration: 20000000,
				Audio: &webm.Audio{
					SamplingFrequency: 48000.0,
					Channels:          2,
				},
			}, {
				Name:            "VttVideoDroppedPacketMeta",
				TrackNumber:     3,
				TrackUID:        54321,
				CodecID:         "D_WEBVTT/METADATA",
				TrackType:       0x21,
				DefaultDuration: 20000000,
			}, {
				Name:            "Video",
				TrackNumber:     4,
				TrackUID:        67890,
				CodecID:         "V_VP8",
				TrackType:       1,
				DefaultDuration: 20000000,
				Video: &webm.Video{
					PixelWidth:  uint64(width),
					PixelHeight: uint64(height),
				},
			},
		}, options...)
	if err != nil {
		log.Errorf("init writer err: %s", err)
	}
	log.Infof("WebM saver has started with video width=%d, height=%d\n", width, height)
	s.audioWriter = ws[1]
	s.videoWriter = ws[3]
	s.vttAudioWriter = ws[0]
	s.vttVideoWriter = ws[2]
}

// SampleWriter for writing samples
type SampleWriter struct {
	Node
}

// NewSampleWriter creates a new sample writer
func NewSampleWriter() *SampleWriter {
	return &SampleWriter{}
}

// Write sample
func (w *SampleWriter) Write(p []byte) (n int, err error) {
	err = w.Node.Write(&avp.Sample{
		Type:    TypeBinary,
		Payload: p,
	})

	if err != nil {
		return 0, err
	}

	return len(p), nil
}

func (w *SampleWriter) Close() error {
	w.Node.Close()
	return nil
}
