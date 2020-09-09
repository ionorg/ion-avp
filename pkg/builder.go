package avp

import (
	"errors"
	"fmt"
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
	mu       sync.RWMutex
	stop     bool
	builder  *samplebuilder.SampleBuilder
	elements sync.Map
	sequence uint16
	track    *webrtc.Track
	out      chan *Sample
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
		builder:  samplebuilder.New(maxLate, depacketizer),
		elements: sync.Map{},
		track:    track,
		out:      make(chan *Sample, maxSize),
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
	b.elements.Store(e.ID(), e)
}

// Track returns the builders underlying track
func (b *Builder) Track() *webrtc.Track {
	return b.track
}

func (b *Builder) build() {
	log.Debugf("Reading rtp for track: %s", b.Track().ID())
	for {
		b.mu.RLock()
		stop := b.stop
		b.mu.RUnlock()
		if stop {
			return
		}

		pkt, err := b.track.ReadRTP()
		if err != nil {
			if err == io.EOF {
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

			b.mu.RLock()
			stop = b.stop
			b.mu.RUnlock()
			if stop {
				return
			}

			if sample == nil {
				break
			}
			b.out <- &Sample{
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

		b.mu.Lock()
		elements := b.elements
		stop := b.stop
		b.mu.Unlock()
		if stop {
			return
		}

		elements.Range(func(_, element interface{}) bool {
			err := element.(Element).Write(sample)
			if err != nil {
				log.Errorf("error writing sample: %s", err)
			}
			return true
		})
	}
}

// Stop stop all buffer
func (b *Builder) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stop {
		return
	}
	b.stop = true

	b.elements.Range(func(eid, element interface{}) bool {
		element.(Element).Close()
		b.elements.Delete(eid)
		return true
	})
	close(b.out)
}

func (b *Builder) stats() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	info := fmt.Sprintf("      track: %s\n", b.track.ID())
	b.elements.Range(func(id, _ interface{}) bool {
		info += fmt.Sprintf("        element: %s\n", id)
		return true
	})
	return info
}
