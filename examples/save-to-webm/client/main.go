// Package pub-from-disk contains an example of publishing a stream to
// an ion-sfu instance from a file on disk.
package main

import (
	"bufio"
	"context"
	"log"
	"os"

	pb "github.com/pion/ion-avp/cmd/server/grpc/proto"
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
		log.Fatalf("Error intializing avp signal stream: %v", err)
	}

	err = client.Send(&pb.SignalRequest{
		Payload: &pb.SignalRequest_Join{
			Join: &pb.JoinRequest{
				Sfu: sfu,
				Sid: sid,
			},
		},
	})

	if err != nil {
		log.Fatalf("Error sending publish request: %v", err)
	}

	buf := bufio.NewReader(os.Stdin)
	log.Print("track id: ")
	id, err := buf.ReadString('\n')
	if err != nil {
		log.Fatalf("error reading ssrc: %s", err)
	}

	err = client.Send(&pb.SignalRequest{
		Payload: &pb.SignalRequest_Process{
			Process: &pb.Process{
				Pid: id,
				Sid: sid,
				Tid: id,
				Eid: "webmsaver",
			},
		},
	})

	select {}
}
