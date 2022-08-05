package balance

import (
	"errors"

	"github.com/KKKKjl/tinykit/internal/registry"
)

type BalanceType string

const (
	WEIGHT_ROUND_ROBIN BalanceType = "weight_round_robin"
	ROUND_ROBIN                    = "round_robin"
	CONSISTENT_HASHING             = "consistent_hashing"
)

var (
	balancer = make(map[BalanceType]Picker)
)

var (
	EmptyServiceErr = errors.New("Empty service list.")
	NilServiceErr   = errors.New("Nil service.")
)

// load balance interface
type Picker interface {
	Pick(key string, services []*registry.Service) (*registry.Service, error)

	Scheme() string
}

// create balance instance.
// If balance type is not support, it will return default type(round robin).
func NewBalancer(balanceType BalanceType) Picker {
	if balance, ok := balancer[balanceType]; ok {
		return balance
	}

	return NewRoundRobin()
}
