package avp

import (
	"errors"
	"io"
	"sync"

	"github.com/pion/ion-avp/pkg/log"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"
)

const (
	maxSize = 100
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
	track         *webrtc.Track
	out           chan *Sample
}

// NewBuilder Initialize a new audio sample builder
func NewBuilder(track *webrtc.Track, maxLate uint16) *Builder {
	var depacketizer rtp.Depacketizer
	var checker rtp.PartitionHeadChecker
	switch track.Codec().Name {
	case webrtc.Opus:
		depacketizer = &codecs.OpusPacket{}
		checker = &codecs.OpusPartitionHeadChecker{}
	case webrtc.VP8:
		depacketizer = &codecs.VP8Packet{}
		checker = &codecs.VP8PartitionHeadChecker{}
	case webrtc.VP9:
		depacketizer = &codecs.VP9Packet{}
		checker = &codecs.VP9PartitionHeadChecker{}
	case webrtc.H264:
		depacketizer = &codecs.H264Packet{}
	}

	b := &Builder{
		builder: samplebuilder.New(maxLate, depacketizer),
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
func (b *Builder) Track() *webrtc.Track {
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

		pkt, err := b.track.ReadRTP()
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
			log.Tracef("Read sample from builder: %s", b.Track().ID())
			sample, timestamp := b.builder.PopWithTimestamp()
			log.Tracef("Got sample from builder: %s sample: %v", b.Track().ID(), sample)

			if b.stopped.get() {
				return
			}

			if sample == nil {
				break
			}

			b.out <- &Sample{
				ID:             b.track.ID(),
				Type:           int(b.track.Codec().Type),
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
		elements := b.elements
		for _, e := range elements {
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
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stopped.get() {
		return
	}
	b.stopped.set(true)
	for _, e := range b.elements {
		e.Close()
	}
	if b.onStopHandler != nil {
		b.onStopHandler()
	}
	close(b.out)
}
