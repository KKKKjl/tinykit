package etcd

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"

	"github.com/KKKKjl/tinykit/internal/registry"
	"github.com/KKKKjl/tinykit/logger"
)

var (
	log     = logger.GetLogger()
	mainLog = log.WithField("prefix", "discovery")
)

var (
	once              sync.Once
	_defaultDiscovery *EtcdDiscovery
)

type Config struct {
	Nodes       []string
	DialTimeout time.Duration
	Prefix      string
}

type EtcdDiscovery struct {
	client         *clientv3.Client
	service        map[string]*registry.Service
	mu             sync.RWMutex
	updateInterval time.Duration
	done           chan struct{}
	prefix         string
}

func initDefault() {
	once.Do(func() {
		_defaultDiscovery = New(nil)
	})
}

func Builder() registry.Builder {
	if _defaultDiscovery == nil {
		initDefault()
	}
	return _defaultDiscovery
}

func New(c *Config) *EtcdDiscovery {
	if c == nil {
		c = &Config{
			Nodes:       []string{"localhost:2379"},
			DialTimeout: time.Second * 3,
			Prefix:      "/discovery/",
		}
	}

	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   c.Nodes,
		DialTimeout: c.DialTimeout,
	})
	if err != nil {
		panic(err)
	}

	discovery := &EtcdDiscovery{
		service:        make(map[string]*registry.Service),
		client:         cli,
		updateInterval: time.Second * 3,
		done:           make(chan struct{}),
		prefix:         c.Prefix,
	}

	go discovery.watch()

	return discovery
}

func (e *EtcdDiscovery) GetService() ([]*registry.Service, error) {
	servers := make([]*registry.Service, 0)

	resp, err := e.client.Get(context.TODO(), e.prefix, clientv3.WithPrefix())
	if err != nil {
		return servers, err
	}

	for _, v := range resp.Kvs {
		var service registry.Service
		if err := json.Unmarshal(v.Value, &service); err != nil {
			continue
		}

		e.PutServer(&service)
		servers = append(servers, &service)
	}

	return servers, nil
}

func (e *EtcdDiscovery) PutServer(service *registry.Service) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.service[service.Name]; !ok {
		e.service[service.Name] = service
	}
}

func (e *EtcdDiscovery) DelServer(service *registry.Service) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.service, service.Name)
}

func (e *EtcdDiscovery) ListServer() []*registry.Service {
	e.mu.RLock()
	defer e.mu.RUnlock()

	servers := make([]*registry.Service, 0, len(e.service))
	for _, v := range e.service {
		servers = append(servers, v)
	}

	return servers
}

func (e *EtcdDiscovery) watch() {
	wch := e.client.Watch(context.TODO(), e.prefix, clientv3.WithPrefix())

	ticker := time.NewTicker(e.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case res, ok := <-wch:
			if ok {
				if err := e.eventCallback(res.Events); err != nil {
					mainLog.Errorf("watch update callback error: %s", err)
				}
			}
		case <-ticker.C:
			if err := e.sync(); err != nil {
				mainLog.Errorf("sync servers error: %s", err)
			}
		case <-e.done:
			if err := e.client.Close(); err != nil {
				mainLog.Errorf("close etcd client error: %s", err)
			}
			return
		}
	}
}

func (e *EtcdDiscovery) eventCallback(events []*clientv3.Event) error {
	for _, event := range events {
		var service registry.Service
		if err := json.Unmarshal(event.Kv.Value, &service); err != nil {
			mainLog.Error("Fail to unmarshal:", err)
			continue
		}

		switch event.Type {
		case clientv3.EventTypePut:
			e.PutServer(&service)
			mainLog.Infof("Watch put server change: %s", service.Name)

		case clientv3.EventTypeDelete:
			e.DelServer(&service)
			mainLog.Infof("Watch del server change: %s", service.Name)
		}
	}

	return nil
}

func (e *EtcdDiscovery) sync() error {
	resp, err := e.client.Get(context.TODO(), e.prefix, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, v := range resp.Kvs {
		var service registry.Service
		if err := json.Unmarshal(v.Value, &service); err != nil {
			mainLog.Error("Fail to unmarshal:", err)
			continue
		}

		e.PutServer(&service)
	}

	return nil
}

func (r *EtcdDiscovery) Scheme() string {
	return "etcd"
}

func (e *EtcdDiscovery) Close() {
	e.done <- struct{}{}
}
