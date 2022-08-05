package broker

import (
	"sync"

	"github.com/gorilla/websocket"
)

type Broker interface {
	Publish(topic string, message []byte) error
	Subscribe(topic string) error
	Unsubscribe(topic string) error
	Close() error
	String() string
}

type Node struct {
	Uid   uint64
	Ws    *websocket.Conn
	Done  chan struct{}
	Renew chan struct{}
	Once  sync.Once
}
