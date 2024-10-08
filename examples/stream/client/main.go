package main

import (
	"context"
	"log"
	"time"

	"github.com/matelq/p2pmp/examples/stream/common"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient(":3000", grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}

	client := common.NewP2PManagerClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.Stream(ctx)

	if err != nil {
		log.Fatalf("could not open stream: %v", err)
	}

	stream.Send(&common.Message{Text: "text"})

	for {
		message, err := stream.Recv()

		if err != nil {
			panic(err)
		}

		log.Println(message)

		stream.Send(&common.Message{Text: message.Text + "more text"})
		time.Sleep(time.Second)
	}
}
