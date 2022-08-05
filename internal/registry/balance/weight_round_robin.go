package balance

import (
	"github.com/KKKKjl/tinykit/internal/registry"
)

//reference: https://github.com/nginx/nginx/commit/52327e0627f49dbda1e8db695e63a4b0af4448b1

type WeightRoundRobin struct {
}

func init() {
	balancer[WEIGHT_ROUND_ROBIN] = NewWeightRoundRobin()
}

func NewWeightRoundRobin() *WeightRoundRobin {
	return &WeightRoundRobin{}
}

func (r *WeightRoundRobin) next(services []*registry.Service) (*registry.Service, error) {
	if len(services) == 0 {
		return nil, EmptyServiceErr
	}

	var best *registry.Service

	totalWeight := 0
	for _, v := range services {
		if v == nil {
			continue
		}

		v.CurrentWeight += v.Weight
		totalWeight += v.Weight

		if best == nil || v.CurrentWeight > best.CurrentWeight {
			best = v
		}
	}

	if best == nil {
		return nil, NilServiceErr
	}

	best.CurrentWeight -= totalWeight
	return best, nil
}

func (r *WeightRoundRobin) Pick(key string, services []*registry.Service) (*registry.Service, error) {
	return r.next(services)
}

func (r *WeightRoundRobin) Scheme() string {
	return "weight_round_robin"
}
