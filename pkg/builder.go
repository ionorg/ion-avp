package avp

import (
	"errors"
	"io"
	"strings"
	"sync"

	log "github.com/pion/ion-log"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"
)

const (
	maxSize      = 100
	MimeTypeH264 = "video/h264"
	MimeTypeOpus = "audio/opus"
	MimeTypeVP8  = "video/vp8"
	MimeTypeVP9  = "video/vp9"
	MimeTypeG722 = "audio/G722"
	MimeTypePCMU = "audio/PCMU"
	MimeTypePCMA = "audio/PCMA"
)

var (
	// ErrCodecNotSupported is returned when a rtp packed it pushed with an unsupported codec
	ErrCodecNotSupported = errors.New("codec not supported")
)

// Builder Module for building video/audio samples from rtp streams
type Builder struct {
	mu            sync.RWMutex
	stopped       atomicBool
	onStopHandler func()
	builder       *samplebuilder.SampleBuilder
	elements      []Element
	sequence      uint16
	track         *webrtc.TrackRemote
	out           chan *Sample
}

// NewBuilder Initialize a new audio sample builder
func NewBuilder(track *webrtc.TrackRemote, maxLate uint16) *Builder {
	var depacketizer rtp.Depacketizer
	var checker rtp.PartitionHeadChecker
	switch strings.ToLower(track.Codec().MimeType) {
	case strings.ToLower(MimeTypeOpus):
		depacketizer = &codecs.OpusPacket{}
		checker = &codecs.OpusPartitionHeadChecker{}
	case strings.ToLower(MimeTypeVP8):
		depacketizer = &codecs.VP8Packet{}
		checker = &codecs.VP8PartitionHeadChecker{}
	case strings.ToLower(MimeTypeVP9):
		depacketizer = &codecs.VP9Packet{}
		checker = &codecs.VP9PartitionHeadChecker{}
	case strings.ToLower(MimeTypeH264):
		depacketizer = &codecs.H264Packet{}
	}

	b := &Builder{
		builder: samplebuilder.New(maxLate, depacketizer, track.Codec().ClockRate),
		track:   track,
		out:     make(chan *Sample, maxSize),
	}

	if checker != nil {
		samplebuilder.WithPartitionHeadChecker(checker)(b.builder)
	}

	go b.build()
	go b.forward()

	return b
}

// AttachElement attaches a element to a builder
func (b *Builder) AttachElement(e Element) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.elements = append(b.elements, e)
}

// Track returns the builders underlying track
func (b *Builder) Track() *webrtc.TrackRemote {
	return b.track
}

// OnStop is called when a builder is stopped
func (b *Builder) OnStop(f func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.onStopHandler = f
}

func (b *Builder) build() {
	log.Debugf("Reading rtp for track: %s", b.Track().ID())
	for {
		if b.stopped.get() {
			return
		}

		pkt, _, err := b.track.ReadRTP()
		if err != nil {
			if err == io.EOF {
				b.stop()
				return
			}
			log.Errorf("Error reading track rtp %s", err)
			continue
		}

		b.builder.Push(pkt)

		for {
			sample, timestamp := b.builder.PopWithTimestamp()

			if b.stopped.get() {
				return
			}

			if sample == nil {
				break
			}

			log.Tracef("Sample from builder: %s sample: %v", b.Track().ID(), sample)

			b.out <- &Sample{
				ID:             b.track.ID(),
				Type:           int(b.track.Kind()),
				SequenceNumber: b.sequence,
				Timestamp:      timestamp,
				Payload:        sample.Data,
			}
			b.sequence++
		}
	}
}

// Read sample
func (b *Builder) forward() {
	for {
		sample := <-b.out

		if b.stopped.get() {
			return
		}

		b.mu.RLock()
		for _, e := range b.elements {
			err := e.Write(sample)
			if err != nil {
				log.Errorf("error writing sample: %s", err)
			}
		}
		b.mu.RUnlock()
	}
}

// Stop stop all buffer
func (b *Builder) stop() {
	if b.stopped.get() {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.stopped.set(true)
	for _, e := range b.elements {
		e.Close()
	}
	if b.onStopHandler != nil {
		b.onStopHandler()
	}
	close(b.out)
}
