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
	"google.golang.org/grpc/credentials/insecure"
)

type ClientServerImpl struct {
	common.UnimplementedClientServerServer
}

func (server ClientServerImpl) CallFuncOnClient(context context.Context, text *common.Text) (*common.Text, error) {
	log.Printf("CallFuncOnClient called: %s", text.Data)

	return &common.Text{Data: fmt.Sprintf("Echo: %s", text.Data)}, nil
}

func callServer() {
	conn, err := grpc.NewClient("89.169.34.96:3000", grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		panic(err)
	}

	defer conn.Close()

	for {
		client := common.NewServerServerClient(conn)
		res, err := client.CallFuncOnServer(context.Background(), &common.Text{Data: "Message from client to server"})

		if err != nil {
			panic(err)
		}

		log.Printf("Response: %s", res.Data)
		time.Sleep(time.Second * 2)
	}
}

func main() {
	conn, err := net.DialTimeout("tcp", "89.169.34.96:3001", 10*time.Second)

	if err != nil {
		panic(err)
	}

	yamuxSession, err := yamux.Server(conn, yamux.DefaultConfig())

	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer()
	clientServerImpl := &ClientServerImpl{}
	common.RegisterClientServerServer(grpcServer, clientServerImpl)

	log.Println("launching gRPC server over TCP connection...")

	go callServer()

	err = grpcServer.Serve(yamuxSession)

	if err != nil {
		panic(err)
	}
}
