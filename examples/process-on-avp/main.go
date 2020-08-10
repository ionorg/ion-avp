// Package pub-from-disk contains an example of publishing a stream to
// an ion-sfu instance from a file on disk.
package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	avp "github.com/pion/ion-avp/cmd/server/grpc/proto"
	"github.com/pion/webrtc/v3"
	"google.golang.org/grpc"
)

const (
	address = "localhost:50052"
)

func main() {
	// Set up a connection to the avp server.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := avp.NewSFUClient(conn)

	sid := os.Args[1]
	ctx := context.Background()
	client, err := c.Signal(ctx)

	if err != nil {
		log.Fatalf("Error publishing stream: %v", err)
	}

	err = client.Send(&avp.SignalRequest{
		Payload: &sfu.SignalRequest_Join{
			Join: &sfu.JoinRequest{
				Sid: sid,
				Offer: &sfu.SessionDescription{
					Type: pubOffer.Type.String(),
					Sdp:  []byte(pubOffer.SDP),
				},
			},
		},
	})

	if err != nil {
		log.Fatalf("Error sending publish request: %v", err)
	}

	for {
		reply, err := client.Recv()

		if err == io.EOF {
			// WebRTC Transport closed
			fmt.Println("WebRTC Transport Closed")
		}

		if err != nil {
			log.Fatalf("Error receving publish response: %v", err)
		}

		switch payload := reply.Payload.(type) {
		case *sfu.SignalReply_Join:
			fmt.Printf("Got answer from sfu. Starting streaming for pid %s!\n", payload.Join.GetPid())
			// Set the remote SessionDescription
			if err = peerConnection.SetRemoteDescription(webrtc.SessionDescription{
				Type: webrtc.SDPTypeAnswer,
				SDP:  string(payload.Join.Answer.Sdp),
			}); err != nil {
				panic(err)
			}
		}
	}
}

// Search for Codec PayloadType
//
// Since we are answering we need to match the remote PayloadType
func getPayloadType(m webrtc.MediaEngine, codecType webrtc.RTPCodecType, codecName string) uint8 {
	for _, codec := range m.GetCodecsByKind(codecType) {
		if codec.Name == codecName {
			return codec.PayloadType
		}
	}
	panic(fmt.Sprintf("Remote peer does not support %s", codecName))
}
