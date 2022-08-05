package ws

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/KKKKjl/tinykit/internal/broker"
	"github.com/KKKKjl/tinykit/internal/broker/pubsub"
	"github.com/KKKKjl/tinykit/logger"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

type WsHanlder struct {
	upgrader       websocket.Upgrader
	logger         *logrus.Logger
	pubsub         *pubsub.Pubsub
	updateInterval time.Duration
	// onMessgae      func(server *websocket.Conn, msg []byte)
	// onClose        func(server *websocket.Conn)
}

type PushMsg struct {
	NotificationId string `json:"notification_id"`
	Source         string `json:"source"`
	Topic          string `json:"topic"`
	Data           string `json:"data"`
	Op             string `json:"operation"`
}

var _defaultNode *broker.Node

func init() {
	_defaultNode = &broker.Node{}
}

func NewWsHanlder() *WsHanlder {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		//CheckOrigin:     CheckSameOrigin(),
	}

	return &WsHanlder{
		upgrader:       upgrader,
		logger:         logger.GetLogger(),
		updateInterval: time.Second * 10,
		pubsub:         pubsub.New(),
	}
}

func (ws *WsHanlder) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// update http connection
	wsConn, err := ws.upgrader.Upgrade(w, r, nil)
	if err != nil {
		ws.logger.Errorf("Upgrade error: %v", err)
		http.Error(w, fmt.Sprintf("error when upgrading connection: %v", err), http.StatusInternalServerError)
		return
	}

	// get unique id for this connection
	uid := atomic.AddUint64(&_defaultNode.Uid, 1)

	node := &broker.Node{
		Uid:   uid,
		Ws:    wsConn,
		Done:  make(chan struct{}),
		Renew: make(chan struct{}),
	}

	// register handler
	ws.setCloseHandler(node)
	ws.setPingHandler(node)

	ws.logger.Debugf("Rev new connection %d.", uid)

	go ws.read(node)
	go ws.checkIsAlive(node)
}

func (ws *WsHanlder) handleMsg(messageType int, message []byte, node *broker.Node) {
	switch messageType {
	case websocket.TextMessage:
		{
			ws.logger.Debugf("received message: %s", message)

			var msg PushMsg
			if err := json.Unmarshal(message, &msg); err != nil {
				ws.logger.Errorf("Ws %d parse msg err: %v", node.Uid, err)
				return
			}

			if msg.Topic == "" {
				return
			}

			switch msg.Op {
			case "subscribe":
				ws.logger.Debugf("Ws %d subscribe topic: %s", node.Uid, msg.Topic)

				if err := ws.pubsub.Subscribe(msg.Topic, node); err != nil {
					ws.logger.Errorf("Ws %d subscribe topic %s err: %v", node.Uid, msg.Topic, err)
				}
			case "unsubscribe":
				ws.logger.Debugf("Ws %d unsubscribe topic: %s", node.Uid, msg.Topic)

				if err := ws.pubsub.UnSubscribe(msg.Topic, node); err != nil {
					ws.logger.Errorf("Ws %d unsubscribe topic %s err: %v", node.Uid, msg.Topic, err)
				}
			case "push":
				if msg.Data == "" {
					return
				}

				ws.logger.Debugf("Ws %d publish topic %s data %s", node.Uid, msg.Topic, msg.Data)
				if err := ws.pubsub.Publish(msg.Topic, []byte(msg.Data)); err != nil {
					ws.logger.Errorf("Ws %d publish topic %s err: %v", node.Uid, msg.Topic, err)
				}
			}
		}
	case websocket.BinaryMessage:
		ws.logger.Debugf("received binary message: %s", message)
	case websocket.CloseMessage:
	case websocket.PingMessage:
		{

		}
	default:
		ws.logger.Warnf("received unknown message type(%d): %s", messageType, message)
	}

	// message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
	// c.hub.broadcast <- message
}

func (ws *WsHanlder) read(node *broker.Node) {
	if node == nil {
		return
	}

	for {
		messageType, message, err := node.Ws.ReadMessage()
		if err != nil {
			// check if connection already closed
			if _, ok := <-node.Done; !ok {
				break
			}

			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				ws.logger.Errorf("Read error(%d): %v", node.Uid, err)
			}
			break
		}

		ws.handleMsg(messageType, message, node)

		select {
		case <-node.Done:
			ws.logger.Debugf("Connection closed(%d)", node.Uid)
			return
		default:
		}
	}
}

func (ws *WsHanlder) Close(node *broker.Node) (err error) {
	if node == nil {
		return errors.New("Ws not connected.")
	}
	// ensure only close once
	node.Once.Do(func() {
		if err = node.Ws.Close(); err != nil {
			ws.logger.Errorf("close ws %d error: %v", node.Uid, err)
		}

		close(node.Done)
	})

	return
}

func (ws *WsHanlder) keepAlive(node *broker.Node) {
	if node != nil {
		node.Renew <- struct{}{}
	}
}

func (ws *WsHanlder) checkIsAlive(node *broker.Node) {
	if node == nil {
		return
	}

	tricker := time.NewTimer(ws.updateInterval)
	for {
		select {
		case <-node.Done:
			ws.logger.Debugf("connection %d closed", node.Uid)
			return
		case <-node.Renew:
			ws.logger.Debugf("connection %d keep alive", node.Uid)
			// reset timer
			tricker.Reset(ws.updateInterval)
		case <-tricker.C:
			ws.logger.Debugf("ws %d is offline", node.Uid)
			ws.Close(node)
			return
		}
	}
}

// close handler callback
func (ws *WsHanlder) setCloseHandler(node *broker.Node) {
	if node == nil {
		return
	}

	node.Ws.SetCloseHandler(func(code int, text string) error {
		ws.logger.Debugf("%d received close message with code: %d, text: %s", node.Uid, code, text)
		return ws.Close(node)
	})
}

// ping handler callback
func (ws *WsHanlder) setPingHandler(node *broker.Node) {
	if node == nil {
		return
	}

	node.Ws.SetPingHandler(func(msg string) error {
		ws.logger.Debugf("%d received ping message: %s", node.Uid, msg)
		ws.keepAlive(node)
		return nil
	})
}

func CheckSameOrigin() func(r *http.Request) bool {
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return false
		}

		uri, err := url.Parse(origin)
		if err != nil {
			return false
		}

		return strings.EqualFold(r.Host, uri.Host)
	}
}
