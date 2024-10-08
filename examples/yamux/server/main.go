package main

import (
	"context"
	"log"
	"net"

	"github.com/hashicorp/yamux"
	"github.com/matelq/p2pmp/common"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func handleConn(conn *grpc.ClientConn) {
	defer conn.Close()

	client := common.NewP2PManagerClient(conn)
	res, err := client.SendMessage(context.Background(), &common.Message{Text: "Message from server to client"})

	if err != nil {
		panic(err)
	}

	log.Printf("From client to server: %s", res.Text)
}

func main() {
	log.Println("launching TPC server...")

	lis, err := net.Listen("tcp", ":3000")

	if err != nil {
		panic(err)
	}

	defer lis.Close()

	for {
		log.Println("waiting TCP connections...")

		conn, err := lis.Accept()

		if err != nil {
			panic(err)
		}

		session, err := yamux.Client(conn, yamux.DefaultConfig())

		if err != nil {
			panic(err)
		}

		log.Println("launching gRPC server over TCP connection")

		clientConn, err := grpc.NewClient(":8000", grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return session.Open() }))

		if err != nil {
			panic(err)
		}

		go handleConn(clientConn)
	}
}
