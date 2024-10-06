package main

import (
	"context"
	"log"
	"time"

	"github.com/matelq/p2pmp/common"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	conn, err := grpc.NewClient(":3000", grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("did not connect: %s", err)
	}

	c := common.NewP2PManagerClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.SendMessage(ctx, &common.Message{Text: "Hello World!"})
	if err != nil {
		log.Fatalf("could not send: %v", err)
	}
	log.Printf("echo: %s", r.Text)

}
