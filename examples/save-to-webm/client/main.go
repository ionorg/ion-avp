// Package pub-from-disk contains an example of publishing a stream to
// an ion-sfu instance from a file on disk.
package main

import (
	"bufio"
	"context"
	"os"
	"strings"

	pb "github.com/pion/ion-avp/cmd/signal/grpc/proto"
	log "github.com/pion/ion-log"
	"google.golang.org/grpc"
)

const (
	address = "localhost:50052"
	sfu     = "localhost:50051"
)

func main() {

	log.Init("info")

	// Set up a connection to the avp server.
	conn, err := grpc.Dial(address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Errorf("did not connect: %v", err)
		return
	}
	defer conn.Close()
	c := pb.NewAVPClient(conn)

	sid := os.Args[1]
	ctx := context.Background()
	client, err := c.Signal(ctx)

	if err != nil {
		log.Errorf("Error intializing avp signal stream: %s", err)
		return
	}

	buf := bufio.NewReader(os.Stdin)
	log.Infof("track id: ")
	id, err := buf.ReadString('\n')
	if err != nil {
		log.Errorf("error reading ssrc: %s", err)
		return
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
		log.Errorf("error sending signal request: %s", err)
		return
	}

	select {}
}
