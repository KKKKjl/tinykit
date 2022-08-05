package balance

import (
	"sync/atomic"

	"github.com/KKKKjl/tinykit/internal/registry"
)

type RoundRobin struct {
	count *uint64
}

func init() {
	balancer[ROUND_ROBIN] = NewRoundRobin()
}

func NewRoundRobin() *RoundRobin {
	var op uint64
	op = 0

	return &RoundRobin{
		count: &op,
	}
}

func (r *RoundRobin) Pick(key string, services []*registry.Service) (*registry.Service, error) {
	length := len(services)
	if length <= 0 {
		return nil, EmptyServiceErr
	}

	return services[int(atomic.AddUint64(r.count, 1))%length], nil
}

func (r *RoundRobin) Scheme() string {
	return "round_robin"
}
