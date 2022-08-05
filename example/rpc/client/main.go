package main

import (
	"flag"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

var (
	gateWayAddr        = flag.String("addr", "localhost:8081", "http service address")
	rpcAddr     string = ":8082"
	name        string = "tinykit"
)

func ws() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *gateWayAddr, Path: "/echo"}
	log.Printf("connecting to %s", u.String())

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})

	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", message)
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			select {
			case <-done:
			case <-time.After(time.Second):
			}
			return
		}
	}
}

func main() {
	ws()
	// conn, err := grpc.Dial(rpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	// if err != nil {
	// 	log.Fatalf("did not connect: %v", err)
	// }
	// defer conn.Close()

	// // c := pb.NewGreeterClient(conn)

	// stream := pb.NewStreamServiceClient(conn)

	// Contact the server and print out its response.
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// defer cancel()

	// r, err := c.SayHello(ctx, &pb.HelloRequest{Name: name})
	// if err != nil {
	// 	log.Fatalf("could not greet: %v", err)
	// }

	// resp, err := stream.StreamRpc(context.Background(), &pb.ServerStreamData{Msg: ""})
	// if err != nil {
	// 	log.Fatalf("err: %v", err)
	// }

	// for {
	// 	data, err := resp.Recv()
	// 	if err != nil {
	// 		if err == io.EOF {
	// 			log.Println("receive msg done")
	// 			return
	// 		}

	// 		log.Fatalf("receive data err %v", err)
	// 		return
	// 	}

	// 	log.Printf("receive msg from client %s", data.Msg)
	// }

	//log.Printf("Greeting: %s", r.GetMessage())
}
