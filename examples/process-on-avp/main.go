// Package pub-from-disk contains an example of publishing a stream to
// an ion-sfu instance from a file on disk.
package main

import (
	"context"
	"log"
	"os"

	avp "github.com/pion/ion-avp/cmd/server/grpc/proto"
	"google.golang.org/grpc"
)

const (
	address = "localhost:50052"
	sfu     = "localhost:50051"
)

func main() {
	// Set up a connection to the avp server.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := avp.NewAVPClient(conn)

	sid := os.Args[1]
	ctx := context.Background()
	client, err := c.Signal(ctx)

	if err != nil {
		log.Fatalf("Error intializing avp signal stream: %v", err)
	}

	err = client.Send(&avp.SignalRequest{
		Payload: &avp.SignalRequest_Join{
			Join: &avp.JoinRequest{
				Sfu: sfu,
				Sid: sid,
			},
		},
	})

	if err != nil {
		log.Fatalf("Error sending publish request: %v", err)
	}

	select {}
}
