package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/matelq/p2pmp/examples/stream/common"
	"google.golang.org/grpc"
)

type myP2PManagerServer struct {
	common.UnimplementedP2PManagerServer
}

func (s *myP2PManagerServer) SendMessage(context context.Context, manager *common.Message) (*common.Echo, error) {
	return &common.Echo{
		Text: fmt.Sprintf("Echo: %s", manager.Text),
	}, nil
}

func (s *myP2PManagerServer) Stream(stream grpc.BidiStreamingServer[common.Message, common.Message]) error {
	for {
		message, err := stream.Recv()

		if err != nil {
			return err
		}

		log.Println(message)

		stream.Send(&common.Message{Text: message.Text + " more text"})
	}
}

func main() {
	lis, err := net.Listen("tcp", ":3000")

	if err != nil {
		log.Fatalf("cannot create listener: %s", err)
	}

	grpcServer := grpc.NewServer()
	service := &myP2PManagerServer{}

	common.RegisterP2PManagerServer(grpcServer, service)
	err = grpcServer.Serve(lis)

	if err != nil {
		log.Fatalf("impossible to serve: %s", err)
	}
}
