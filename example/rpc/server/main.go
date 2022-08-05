package main

import (
	"context"
	"log"
	"math/rand"
	"net"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "github.com/KKKKjl/tinykit/example/rpc/helloworld"
)

var (
	addr    = ":8082"
	letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
)

type (
	server struct {
		pb.UnimplementedGreeterServer
	}

	streamServer struct {
		pb.UnimplementedStreamServiceServer
	}
)

func NewServer() *server {
	return &server{}
}

func (s *server) SayHello(ctx context.Context, in *pb.HelloRequest) (*pb.HelloReply, error) {
	log.Printf("receive msg from %s", in.Name)
	return &pb.HelloReply{Message: in.Name + " world"}, nil
}

func NewStreamServer() *streamServer {
	return &streamServer{}
}

func (s *streamServer) StreamRpc(in *pb.ServerStreamData, stream pb.StreamService_StreamRpcServer) error {
	for {
		msg := &pb.ServerStreamData{
			Msg: randReadStr(10),
		}

		if err := stream.Send(msg); err != nil {
			return err
		}

		log.Printf("send msg <- %s", msg.Msg)

		time.Sleep(time.Second)
	}
}

func randReadStr(max int) string {
	str := make([]rune, max)

	for i := range str {
		str[i] = letters[rand.Intn(len(letters))]
	}

	return string(str)
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalln("Failed to listen:", err)
	}

	s := grpc.NewServer()

	pb.RegisterGreeterServer(s, &server{})
	pb.RegisterStreamServiceServer(s, &streamServer{})

	reflection.Register(s)
	log.Printf("Serving rpc on %s", addr)
	log.Fatal(s.Serve(lis))
}
