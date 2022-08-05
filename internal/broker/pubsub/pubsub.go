package pubsub

import (
	"fmt"
	"strconv"
	"sync"

	"github.com/KKKKjl/tinykit/internal/broker"
	"github.com/KKKKjl/tinykit/logger"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	LIMIT       = 100
	SHARE_COUNT = 32
)

type Bucket struct {
	idx         int
	mu          sync.RWMutex
	subscribers map[uint64]*broker.Node // subscriber id -> ws
}

type Pubsub struct {
	mu     sync.RWMutex
	topics map[string][]*Bucket // Reduced lock granularity in 32 segments (SHARE_COUNT)
	logger *logrus.Logger
}

func New() *Pubsub {
	return &Pubsub{
		topics: make(map[string][]*Bucket),
		logger: logger.GetLogger(),
		mu:     sync.RWMutex{},
	}
}

func (ps *Pubsub) add(topic string, node *broker.Node) error {
	ps.mu.Lock()
	if _, ok := ps.topics[topic]; !ok {
		ps.topics[topic] = make([]*Bucket, 0, SHARE_COUNT)

		for i := 0; i < SHARE_COUNT; i++ {
			ps.topics[topic] = append(ps.topics[topic], &Bucket{
				idx:         i,
				subscribers: make(map[uint64]*broker.Node),
			})
		}
	}

	bucket := ps.getShareBucket(strconv.FormatUint(node.Uid, 10), ps.topics[topic])
	ps.mu.Unlock()

	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	if _, ok := bucket.subscribers[node.Uid]; ok {
		return fmt.Errorf("Subscriber %d already subscribe topic %s.", node.Uid, topic)
	}

	bucket.subscribers[node.Uid] = node
	return nil
}

func (ps *Pubsub) remove(topic string, node *broker.Node) error {
	ps.mu.RLock()
	if _, ok := ps.topics[topic]; !ok {
		ps.mu.RUnlock()
		return fmt.Errorf("Topic %s not found.", topic)
	}

	bucket := ps.getShareBucket(strconv.FormatUint(node.Uid, 10), ps.topics[topic])
	ps.mu.RUnlock()

	bucket.mu.Lock()
	defer bucket.mu.Unlock()

	delete(bucket.subscribers, node.Uid)
	return nil
}

func (ps *Pubsub) Publish(topic string, message []byte) error {
	ps.mu.RLock()
	buckets, ok := ps.topics[topic]
	if !ok {
		ps.mu.RUnlock()
		return fmt.Errorf("Topic %s not found.", topic)
	}
	ps.mu.RUnlock()

	waiter := &sync.WaitGroup{}
	for _, val := range buckets {
		waiter.Add(1)
		go func(bucket *Bucket) {
			defer waiter.Done()

			bucket.mu.Lock()
			defer bucket.mu.Unlock()

			for _, subscribe := range bucket.subscribers {
				if err := subscribe.Ws.WriteMessage(websocket.TextMessage, message); err != nil {
					ps.logger.Printf("Error publishing message to %d:%v", subscribe.Uid, err)
				}
			}
		}(val)
	}

	waiter.Wait()

	return nil
}

func (ps *Pubsub) Subscribe(topic string, node *broker.Node) error {
	return ps.add(topic, node)
}

func (ps *Pubsub) UnSubscribe(topic string, node *broker.Node) error {
	return ps.remove(topic, node)
}

func (ps *Pubsub) TopicList() []string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	topics := make([]string, 0, len(ps.topics))
	for topic := range ps.topics {
		topics = append(topics, topic)
	}

	return topics
}

func (ps *Pubsub) TopicSubscribersList(topic string) []uint64 {
	ps.mu.RLock()

	var subscribers []uint64
	buckets, ok := ps.topics[topic]
	if !ok {
		ps.mu.RUnlock()
		return subscribers
	}
	ps.mu.RUnlock()

	for _, bucket := range buckets {
		bucket.mu.RLock()
		for subscriber := range bucket.subscribers {
			subscribers = append(subscribers, subscriber)
		}
		bucket.mu.RUnlock()
	}

	return subscribers
}

func (ps *Pubsub) getShareBucket(key string, buckets []*Bucket) *Bucket {
	return buckets[uint(fnv32(key))%uint(SHARE_COUNT)]
}

// func (h *Pubsub) ClearWs() error {
// 	h.mu.RLock()
// 	defer h.mu.RUnlock()

// 	// limit nums of goroutine
// 	limiter := make(chan struct{}, LIMIT)

// 	waiter := &sync.WaitGroup{}
// 	for _, v := range h.conns {
// 		limiter <- struct{}{}
// 		waiter.Add(1)

// 		go func(ws *websocket.Conn) {
// 			defer func() {
// 				waiter.Done()
// 				<-limiter
// 			}()

// 			ws.Close()
// 		}(v)
// 	}

// 	waiter.Wait()

// 	return nil
// }

func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	keyLength := len(key)
	for i := 0; i < keyLength; i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}
