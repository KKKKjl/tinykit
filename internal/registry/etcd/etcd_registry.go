package etcd

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/KKKKjl/tinykit/internal/registry"
)

type EtcdClient struct {
	client *clientv3.Client
	ctx    context.Context
	cancel context.CancelFunc
}

// create a etcd client instance
func NewEtcdClient() (*EtcdClient, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"localhost:2379"},
		DialTimeout: time.Second * 3,
	})
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	res := &EtcdClient{
		client: cli,
		ctx:    ctx,
		cancel: cancel,
	}

	return res, nil
}

func (e *EtcdClient) putWithLease(key string, value string, ttl int64) error {
	// create new lease
	resp, err := e.client.Grant(context.TODO(), ttl)
	if err != nil {
		return err
	}

	// bind renew id
	if _, err = e.client.Put(context.TODO(), key, value, clientv3.WithLease(resp.ID)); err != nil {
		return err
	}

	// renew
	e.keepAlive(e.ctx, resp.ID)

	return nil
}

func (e *EtcdClient) Register(service *registry.Service, ttl int64) error {
	if !strings.HasPrefix(service.Name, "/discovery/") {
		service.Name = "/discovery/" + service.Name
	}

	buf, err := json.Marshal(service)
	if err != nil {
		return err
	}

	return e.putWithLease(service.Name, string(buf), ttl)
}

func (e *EtcdClient) keepAlive(ctx context.Context, id clientv3.LeaseID) error {
	keepAliveCh, err := e.client.KeepAlive(ctx, id)
	if err != nil {
		return err
	}

	go func() {
		for {
			select {
			case res := <-keepAliveCh:
				mainLog.Debugf("%d keep alive successfully", res.ID)
			case <-ctx.Done():
				// revoke lease
				e.client.Revoke(context.TODO(), id)
				return
			}
		}
	}()

	return nil
}

func (e *EtcdClient) Close() error {
	defer e.cancel()
	return e.client.Close()
}
