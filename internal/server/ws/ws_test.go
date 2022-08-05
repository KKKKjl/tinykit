package ws

import (
	"context"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/suite"
)

type TestSuite struct {
	suite.Suite
}

func (suite *TestSuite) SetupTest() {
	// listen port
	go http.ListenAndServe(":8080", NewWsHanlder())
}

func (suite *TestSuite) TestTimeout() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	u := url.URL{Scheme: "ws", Host: "localhost:8080", Path: "/ws/hux"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	suite.Nil(err)

	defer c.Close()

	tricker := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-tricker.C:
			err := c.WriteMessage(websocket.PingMessage, []byte{})
			suite.Nil(err)
		}
	}
}

func TestTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}
