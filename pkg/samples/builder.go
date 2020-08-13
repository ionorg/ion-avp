package samples

import (
	"errors"
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
	sequence uint16
	track    *webrtc.Track
	outChan  chan *Sample
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
		outChan: make(chan *Sample, maxSize),
	}

	if checker != nil {
		samplebuilder.WithPartitionHeadChecker(checker)(b.builder)
	}

	b.start()

	return b
}

// Track returns the builders underlying track
func (b *Builder) Track() *webrtc.Track {
	return b.track
}

func (b *Builder) start() {
	log.Debugf("Reading rtp for track: %s", b.Track().ID())
	go func() {
		for {
			b.mu.RLock()
			stop := b.stop
			b.mu.RUnlock()
			if stop {
				return
			}

			pkt, err := b.track.ReadRTP()
			if err != nil {
				log.Errorf("Error reading track rtp %s", err)
				continue
			}

			b.builder.Push(pkt)
		}
	}()

	go func() {
		for {
			b.mu.RLock()
			stop := b.stop
			b.mu.RUnlock()
			if stop {
				return
			}
			sample, timestamp := b.builder.PopWithTimestamp()
			if sample == nil {
				return
			}
			b.outChan <- &Sample{
				Type:           int(b.track.Codec().Type),
				SequenceNumber: b.sequence,
				Timestamp:      timestamp,
				Payload:        sample.Data,
			}
			b.sequence++
		}
	}()
}

// Read sample
func (b *Builder) Read() *Sample {
	return <-b.outChan
}

// Stop stop all buffer
func (b *Builder) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.stop {
		return
	}
	b.stop = true
}
