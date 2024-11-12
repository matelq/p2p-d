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

func handleConn(conn *grpc.ClientConn, n int) {
	defer conn.Close()

	for {
		client := common.NewClientServerClient(conn)
		res, err := client.CallFuncOnClient(context.Background(), &common.Text{Data: fmt.Sprintf("Message from server to client %d", n)})

		if err != nil {
			panic(err)
		}

		log.Printf("Response: %s", res.Data)
		time.Sleep(time.Second * 2)
	}
}

type ServerServerImpl struct {
	common.UnimplementedServerServerServer
}

func (server ServerServerImpl) CallFuncOnServer(context context.Context, text *common.Text) (*common.Text, error) {
	log.Printf("CallFuncOnServer called: %s", text.Data)

	return &common.Text{Data: fmt.Sprintf("Echo: %s", text.Data)}, nil
}

func startSever() {
	listener, err := net.Listen("tcp", ":3000")

	if err != nil {
		panic(err)
	}

	grpcServer := grpc.NewServer()
	serverServerImpl := &ServerServerImpl{}
	common.RegisterServerServerServer(grpcServer, serverServerImpl)

	log.Println("launching TPC GRPC server...")
	err = grpcServer.Serve(listener)

	if err != nil {
		panic(err)
	}
}

func main() {
	go startSever()

	log.Println("launching TPC server...")

	listener, err := net.Listen("tcp", ":3001")

	if err != nil {
		panic(err)
	}

	defer listener.Close()

	n := 0

	for {
		log.Println("waiting TCP connections...")

		conn, err := listener.Accept()

		if err != nil {
			panic(err)
		}

		yamuxSession, err := yamux.Client(conn, yamux.DefaultConfig())

		if err != nil {
			panic(err)
		}

		log.Println("launching gRPC server over TCP connection")

		clientConn, err := grpc.NewClient(":3000", grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return yamuxSession.Open() }))

		if err != nil {
			panic(err)
		}

		go handleConn(clientConn, n)
		n++
	}
}
