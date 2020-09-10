package avp

import (
	"context"
	"github.com/pion/rtcp"
	"github.com/pion/transport/test"
	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"sync"
	"testing"
	"time"
)

func CreateTestRTCTransport() *WebRTCTransport {
	return NewWebRTCTransport("id", WebRTCTransportConfig{})
}

func TestNewWebRTCTransport(t *testing.T) {
	report := test.CheckRoutines(t)
	defer report()

	me := webrtc.MediaEngine{}
	me.RegisterDefaultCodecs()
	api := webrtc.NewAPI(webrtc.WithMediaEngine(me))
	remoteA, _, err := newPair(webrtc.Configuration{}, api)
	assert.NoError(t, err)
	defer remoteA.Close()

	track, err := remoteA.NewTrack(webrtc.DefaultPayloadTypeOpus, rand.Uint32(), "audio", "pion")
	assert.NoError(t, err)

	_, err = remoteA.AddTrack(track)
	assert.NoError(t, err)

	onTransportCloseFired, onTransportCloseFunc := context.WithCancel(context.Background())

	registry := NewRegistry()
	registry.AddElement("test-eid", testFunc)
	Init(registry)

	rtcTransport := CreateTestRTCTransport()

	rtcTransport.OnClose(func() {
		onTransportCloseFunc()
	})

	rtcTransport.Process("123", "tid", "eid", []byte{})

	offer, err := remoteA.CreateOffer(nil)
	gatherComplete := webrtc.GatheringCompletePromise(remoteA)
	assert.NoError(t, err)
	assert.NoError(t, remoteA.SetLocalDescription(offer))
	<-gatherComplete

	assert.NoError(t, rtcTransport.SetRemoteDescription(offer))

	answer, err := rtcTransport.CreateAnswer()
	assert.NoError(t, err)
	assert.NoError(t, rtcTransport.SetLocalDescription(answer))
	assert.NoError(t, remoteA.SetRemoteDescription(answer))

	time.Sleep(time.Second * 10)

	expectedString := []string{"track", "pending"}

	stats := rtcTransport.stats()
	for _, expected := range expectedString {
		assert.Contains(t, stats, expected)
	}

	assert.NoError(t, rtcTransport.Close())

	assert.NotNil(t, rtcTransport)

	sendRTPUntilDone(onTransportCloseFired.Done(), t, []*webrtc.Track{track})
}

func TestNewWebRTCTransportWithBuilder(t *testing.T) {
	report := test.CheckRoutines(t)
	defer report()

	me := webrtc.MediaEngine{}
	me.RegisterDefaultCodecs()
	api := webrtc.NewAPI(webrtc.WithMediaEngine(me))
	_, remoteB, err := newPair(webrtc.Configuration{}, api)
	assert.NoError(t, err)
	defer remoteB.Close()
	assert.NoError(t, err)

	track, err := remoteB.NewTrack(webrtc.DefaultPayloadTypeOpus, rand.Uint32(), "audio", "pion")
	assert.NoError(t, err)

	_, err = remoteB.AddTrack(track)
	assert.NoError(t, err)

	onTransportCloseFired, onTransportCloseFunc := context.WithCancel(context.Background())

	registry := NewRegistry()
	registry.AddElement("test-eid", testFunc)
	Init(registry)

	rtcTransport := CreateTestRTCTransport()
	assert.NotNil(t, rtcTransport)

	rtcTransport.OnClose(func() {
		onTransportCloseFunc()
	})

	rtcTransport.Process("123", "tid", "test-eid", []byte{})
	rtcTransport.Process("123", "tid", "test-eid", []byte{})
	offer, err := rtcTransport.CreateOffer()
	assert.NoError(t, err)
	assert.NotNil(t, offer)

	assert.NoError(t, rtcTransport.SetLocalDescription(offer))
	assert.NoError(t, remoteB.SetRemoteDescription(*rtcTransport.pc.LocalDescription()))

	answer, err := remoteB.CreateAnswer(nil)
	assert.NoError(t, err)
	err = remoteB.SetLocalDescription(answer)
	assert.NoError(t, err)
	assert.NoError(t, rtcTransport.SetRemoteDescription(*remoteB.LocalDescription()))

	rtcTransport.OnICECandidate(func(c *webrtc.ICECandidate) {})

	assert.NoError(t, rtcTransport.AddICECandidate(webrtc.ICECandidateInit{Candidate: "1986380506 99 udp 2 10.0.75.1 53634 typ host generation 0 network-id 2"}))
	assert.NoError(t, rtcTransport.Close())
	<-onTransportCloseFired.Done()

	assert.NoError(t, rtcTransport.pc.Close())

}

func TestNewWebRTCTransportWithOnNegotiation(t *testing.T) {
	report := test.CheckRoutines(t)
	defer report()

	me := webrtc.MediaEngine{}
	me.RegisterDefaultCodecs()
	api := webrtc.NewAPI(webrtc.WithMediaEngine(me))
	_, remoteB, err := newPair(webrtc.Configuration{}, api)
	assert.NoError(t, err)
	defer remoteB.Close()
	assert.NoError(t, err)

	onTransportCloseFired, onTransportCloseFunc := context.WithCancel(context.Background())

	registry := NewRegistry()
	registry.AddElement("test-eid", testFunc)
	Init(registry)

	rtcTransport := CreateTestRTCTransport()
	assert.NotNil(t, rtcTransport)

	rtcTransport.OnClose(func() {
		onTransportCloseFunc()
	})

	var wg sync.WaitGroup
	wg.Add(1)
	remoteB.OnNegotiationNeeded(func() {
		offer, offerErr := remoteB.CreateOffer(nil)
		gatherComplete := webrtc.GatheringCompletePromise(remoteB)
		assert.NoError(t, offerErr)
		assert.NoError(t, remoteB.SetLocalDescription(offer))

		<-gatherComplete
		assert.NoError(t, rtcTransport.SetRemoteDescription(offer))

		answer, transportErr := rtcTransport.CreateAnswer()
		assert.NoError(t, transportErr)

		assert.NoError(t, rtcTransport.SetLocalDescription(answer))
		assert.NoError(t, remoteB.SetRemoteDescription(answer))

		wg.Done()
	})

	track, err := remoteB.NewTrack(webrtc.DefaultPayloadTypeOpus, rand.Uint32(), "audio", "pion")
	assert.NoError(t, err)

	sender, err := remoteB.AddTrack(track)
	assert.NoError(t, err)

	wg.Wait()

	err = remoteB.RemoveTrack(sender)
	assert.NoError(t, err)

	assert.NoError(t, remoteB.WriteRTCP([]rtcp.Packet{&rtcp.ReceiverEstimatedMaximumBitrate{Bitrate: 10000000, SenderSSRC: track.SSRC()}}))
	rtcTransport.OnICECandidate(func(c *webrtc.ICECandidate) {})

	assert.NoError(t, rtcTransport.AddICECandidate(webrtc.ICECandidateInit{Candidate: "1986380506 99 udp 2 10.0.75.1 53634 typ host generation 0 network-id 2"}))

	time.Sleep(time.Second * 10)

	assert.NoError(t, rtcTransport.Close())
	<-onTransportCloseFired.Done()

	assert.NoError(t, rtcTransport.pc.Close())
}
