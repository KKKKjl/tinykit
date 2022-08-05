package main

import (
	"context"
	"encoding/json"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/KKKKjl/tinykit/internal/server/ws"
	"github.com/gorilla/websocket"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()

	buf, err := json.Marshal(&ws.PushMsg{
		NotificationId: "1",
		Source:         "ios",
		Topic:          "test",
		Op:             "subscribe",
	})
	if err != nil {
		log.Fatal(err)
	}

	// send subscribe message
	c.WriteMessage(websocket.TextMessage, buf)

	waiter := make(chan os.Signal, 1)
	signal.Notify(waiter, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		for {
			select {
			case sig := <-waiter:
				{
					log.Printf("Received signal %v, exiting.", sig)
					cancel()
					return
				}
			}
		}
	}()

	go func() {
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("read:", err)
				return
			}
			log.Printf("recv: %s", string(message))

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	tricker := time.NewTicker(5 * time.Second)
	defer tricker.Stop()

	for {
		select {
		case <-ctx.Done():
			if err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")); err != nil {
				log.Println("write close:", err)
				return
			}

			log.Println("close successfully")
			return
		case <-tricker.C:
			if err := c.WriteMessage(websocket.PingMessage, []byte("ping")); err != nil {
				log.Println(err)
			}
			log.Println("ping")
		}
	}
}
