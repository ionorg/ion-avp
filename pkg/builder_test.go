package avp

import (
	"context"
	"io"
	"testing"
	"time"

	"github.com/pion/transport/test"
	"github.com/pion/webrtc/v3"
	"github.com/pion/webrtc/v3/pkg/media"
	"github.com/stretchr/testify/assert"
)

// newPair creates two new peer connections (an offerer and an answerer) using
// the api.
func newPair(cfg webrtc.Configuration, api *webrtc.API) (pcOffer *webrtc.PeerConnection, pcAnswer *webrtc.PeerConnection, err error) {
	pca, err := api.NewPeerConnection(cfg)
	if err != nil {
		return nil, nil, err
	}

	pcb, err := api.NewPeerConnection(cfg)
	if err != nil {
		return nil, nil, err
	}

	return pca, pcb, nil
}

func signalPair(pcOffer *webrtc.PeerConnection, pcAnswer *webrtc.PeerConnection) error {
	offer, err := pcOffer.CreateOffer(nil)
	if err != nil {
		return err
	}
	gatherComplete := webrtc.GatheringCompletePromise(pcOffer)
	if err = pcOffer.SetLocalDescription(offer); err != nil {
		return err
	}
	<-gatherComplete
	if err = pcAnswer.SetRemoteDescription(*pcOffer.LocalDescription()); err != nil {
		return err
	}

	answer, err := pcAnswer.CreateAnswer(nil)
	if err != nil {
		return err
	}
	if err = pcAnswer.SetLocalDescription(answer); err != nil {
		return err
	}
	return pcOffer.SetRemoteDescription(*pcAnswer.LocalDescription())
}

func sendRTPUntilDone(done <-chan struct{}, t *testing.T, tracks []*webrtc.TrackLocalStaticSample) {
	for {
		select {
		case <-time.After(20 * time.Millisecond):
			for _, track := range tracks {
				err := track.WriteSample(media.Sample{Data: []byte{0x01, 0x02, 0x03, 0x04}})
				if err == io.ErrClosedPipe {
					return
				}
				assert.NoError(t, err)
			}
		case <-done:
			return
		}
	}
}

func TestNewBuilder_WithOpusName(t *testing.T) {
	lim := test.TimeOut(time.Second * 30)
	defer lim.Stop()

	report := test.CheckRoutines(t)
	defer report()

	me := webrtc.MediaEngine{}
	_ = me.RegisterDefaultCodecs()
	api := webrtc.NewAPI(webrtc.WithMediaEngine(&me))
	sfu, remote, err := newPair(webrtc.Configuration{}, api)
	defer remote.Close()
	defer sfu.Close()
	assert.NoError(t, err)

	track, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: MimeTypeOpus}, "audio", "pion")
	assert.NoError(t, err)

	_, err = remote.AddTrack(track)
	assert.NoError(t, err)

	onBuilderFired, onBuilderFiredFunc := context.WithCancel(context.Background())
	sfu.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		builder := MustBuilder(NewBuilder(track, 200))

		// To ensure that forward and go on build stops immediately
		builder.stop()
		assert.NotNil(t, builder)

		builder.AttachElement(&elementMock{})
		assert.Equal(t, track, builder.Track())

		onBuilderFiredFunc()
	})

	// defer builder.stop()

	err = signalPair(remote, sfu)
	assert.NoError(t, err)
	sendRTPUntilDone(onBuilderFired.Done(), t, []*webrtc.TrackLocalStaticSample{track})

}

func TestNewBuilder_WithVP8Packet(t *testing.T) {
	report := test.CheckRoutines(t)
	defer report()

	lim := test.TimeOut(time.Second * 30)
	defer lim.Stop()

	var builder *Builder
	me := webrtc.MediaEngine{}
	_ = me.RegisterDefaultCodecs()
	api := webrtc.NewAPI(webrtc.WithMediaEngine(&me))
	sfu, remote, err := newPair(webrtc.Configuration{}, api)
	assert.NoError(t, err)

	track, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: MimeTypeVP8}, "video", "pion")
	assert.NoError(t, err)

	_, err = remote.AddTrack(track)
	assert.NoError(t, err)

	onBuilderFired, onBuilderFiredFunc := context.WithCancel(context.Background())
	sfu.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		builder = MustBuilder(NewBuilder(track, 200))
		defer builder.stop()
		assert.NotNil(t, builder)

		builder.AttachElement(&elementMock{})
		assert.Equal(t, track, builder.Track())

		onBuilderFiredFunc()
	})

	err = signalPair(remote, sfu)
	assert.NoError(t, err)
	sendRTPUntilDone(onBuilderFired.Done(), t, []*webrtc.TrackLocalStaticSample{track})

	assert.NoError(t, remote.Close())
	assert.NoError(t, sfu.Close())

	// This is to ensure that the builder stop is not called when it has already been stopped
	builder.stop()
}

func TestNewBuilder_WithVP9Packet(t *testing.T) {
	report := test.CheckRoutines(t)
	defer report()

	lim := test.TimeOut(time.Second * 30)
	defer lim.Stop()

	var builder *Builder
	me := webrtc.MediaEngine{}
	_ = me.RegisterDefaultCodecs()
	api := webrtc.NewAPI(webrtc.WithMediaEngine(&me))
	sfu, remote, err := newPair(webrtc.Configuration{}, api)
	assert.NoError(t, err)

	track, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: MimeTypeVP9}, "video", "pion")
	assert.NoError(t, err)

	_, err = remote.AddTrack(track)
	assert.NoError(t, err)

	onBuilderFired, onBuilderFiredFunc := context.WithCancel(context.Background())
	sfu.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		builder = MustBuilder(NewBuilder(track, 200))
		defer builder.stop()
		assert.NotNil(t, builder)

		builder.AttachElement(&elementMock{})
		assert.Equal(t, track, builder.Track())

		// To cause the building to stop while trying to read tracks
		builder.stop()

		time.Sleep(time.Second * 10)

		onBuilderFiredFunc()
	})

	err = signalPair(remote, sfu)
	assert.NoError(t, err)
	sendRTPUntilDone(onBuilderFired.Done(), t, []*webrtc.TrackLocalStaticSample{track})

	assert.NoError(t, remote.Close())
	assert.NoError(t, sfu.Close())

	// This is to ensure that the builder stop is not called when it has already been stopped
	builder.stop()
}

func TestNewBuilder_WithH264Packet(t *testing.T) {
	report := test.CheckRoutines(t)
	defer report()

	lim := test.TimeOut(time.Second * 30)
	defer lim.Stop()

	me := webrtc.MediaEngine{}
	_ = me.RegisterDefaultCodecs()
	api := webrtc.NewAPI(webrtc.WithMediaEngine(&me))
	sfu, remote, err := newPair(webrtc.Configuration{}, api)
	assert.NoError(t, err)

	defer sfu.Close()
	defer remote.Close()

	track, err := webrtc.NewTrackLocalStaticSample(webrtc.RTPCodecCapability{MimeType: MimeTypeH264}, "video", "pion")
	assert.NoError(t, err)

	_, err = remote.AddTrack(track)
	assert.NoError(t, err)

	onBuilderFired, onBuilderFiredFunc := context.WithCancel(context.Background())
	sfu.OnTrack(func(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
		builder := MustBuilder(NewBuilder(track, 200))
		assert.NotNil(t, builder)

		builder.AttachElement(&elementMock{})
		assert.Equal(t, track, builder.Track())

		// Add sleep to ensure tracks are being processed by go forward
		time.Sleep(time.Second * 10)
		onBuilderFiredFunc()

		// To ensure that builder.Stop returns at the second time while trying to call another Stop
		builder.stop()
		builder.stop()
	})

	err = signalPair(remote, sfu)
	assert.NoError(t, err)
	sendRTPUntilDone(onBuilderFired.Done(), t, []*webrtc.TrackLocalStaticSample{track})
}
