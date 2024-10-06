package main

import (
	"context"
	"log"
	"time"

	"github.com/matelq/p2pmp/example/common"

	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/encoding/proto"
)

type TestClientConnInterface struct {
}

func (m TestClientConnInterface) Invoke(ctx context.Context, method string, args any, reply any, opts ...grpc.CallOption) error {
	codec := encoding.GetCodecV2(proto.Name)
	bytes, _ := codec.Marshal(&common.Echo{Text: "Hello World!"})

	log.Printf("method: %v", method)
	log.Printf("args: %v", args)
	log.Printf("marshal: %v", bytes)

	return codec.Unmarshal(bytes, reply)
}

func (m TestClientConnInterface) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, nil
}

func main() {
	testConn := &TestClientConnInterface{}
	c := common.NewP2PManagerClient(testConn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.SendMessage(ctx, &common.Message{Text: "Hello World!"})
	if err != nil {
		log.Fatalf("could not send: %v", err)
	}
	log.Printf("echo: %s", r.Text)

}
