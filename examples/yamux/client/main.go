package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/hashicorp/yamux"
	"github.com/matelq/p2pmp/examples/yamux/common"

	"google.golang.org/grpc"
)

type myP2PManagerServer struct {
	common.UnimplementedP2PManagerServer
}

func (s myP2PManagerServer) SendMessage(context context.Context, message *common.Message) (*common.Echo, error) {
	log.Printf("SendMessage called: %s", message.Text)

	return &common.Echo{
		Text: fmt.Sprintf("Echo: %s", message.Text),
	}, nil
}

func main() {
	conn, err := net.DialTimeout("tcp", "127.0.0.1:3000", 10*time.Second)

	if err != nil {
		panic(err)
	}

	session, err := yamux.Server(conn, yamux.DefaultConfig())

	if err != nil {
		panic(err)
	}

	server := grpc.NewServer()
	service := &myP2PManagerServer{}
	common.RegisterP2PManagerServer(server, service)

	log.Println("launching gRPC server over TCP connection...")

	err = server.Serve(session)

	if err != nil {
		panic(err)
	}
}
