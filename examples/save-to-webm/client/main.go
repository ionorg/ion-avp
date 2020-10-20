// Package pub-from-disk contains an example of publishing a stream to
// an ion-sfu instance from a file on disk.
package main

import (
	"bufio"
	"context"
	"log"
	"os"
	"strings"

	pb "github.com/pion/ion-avp/cmd/signal/grpc/proto"
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
	c := pb.NewAVPClient(conn)

	sid := os.Args[1]
	ctx := context.Background()
	client, err := c.Signal(ctx)

	if err != nil {
		log.Fatalf("Error intializing avp signal stream: %s", err)
	}

	buf := bufio.NewReader(os.Stdin)
	log.Print("track id: ")
	id, err := buf.ReadString('\n')
	if err != nil {
		log.Fatalf("error reading ssrc: %s", err)
	}
	id = strings.TrimSpace(id)

	err = client.Send(&pb.SignalRequest{
		Payload: &pb.SignalRequest_Process{
			Process: &pb.Process{
				Sfu: sfu,
				Pid: id,
				Sid: sid,
				Tid: id,
				Eid: "webmsaver",
			},
		},
	})

	if err != nil {
		log.Fatalf("error sending signal request: %s", err)
	}

	select {}
}
