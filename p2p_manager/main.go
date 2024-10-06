package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/matelq/p2pmp/common"

	"google.golang.org/grpc"
)

type myP2PManagerServer struct {
	common.UnimplementedP2PManagerServer
}

func (s myP2PManagerServer) SendMessage(context context.Context, manager *common.Message) (*common.Echo, error) {
	return &common.Echo{
		Text: fmt.Sprintf("Echo: %s", manager.Text),
	}, nil
}

func main() {
	lis, err := net.Listen("tcp", ":3000")

	if err != nil {
		log.Fatalf("cannot create listener: %s", err)
	}

	s := grpc.NewServer()
	service := &myP2PManagerServer{}

	common.RegisterP2PManagerServer(s, service)
	err = s.Serve(lis)

	if err != nil {
		log.Fatalf("impossible to serve: %s", err)
	}
}
