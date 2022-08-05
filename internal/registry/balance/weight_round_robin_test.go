package balance

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KKKKjl/tinykit/internal/registry"
)

func TestWeightRoundRobin(t *testing.T) {
	services := []*registry.Service{
		{
			Name:     "host1",
			Version:  "v1",
			Addr:     "localhost:8080",
			Metadata: make(map[string]string),
			Weight:   3,
		},
		{
			Name:     "host2",
			Version:  "v1",
			Addr:     "localhost:8081",
			Metadata: make(map[string]string),
			Weight:   2,
		},
		{
			Name:     "host3",
			Version:  "v1",
			Addr:     "localhost:8082",
			Metadata: make(map[string]string),
			Weight:   1,
		},
	}
	assert := assert.New(t)

	roundRobin := NewWeightRoundRobin()

	expects := []struct {
		Expected string
	}{
		{Expected: "localhost:8080"},
		{Expected: "localhost:8081"},
		{Expected: "localhost:8080"},
		{Expected: "localhost:8082"},
		{Expected: "localhost:8081"},
		{Expected: "localhost:8080"},
	}

	for _, v := range expects {
		service, err := roundRobin.Pick(v.Expected, services)

		assert.Nil(err)
		assert.Equalf(service.Addr, v.Expected, "expected %s, got %s", v.Expected, service.Addr)
	}
}

func TestWeightRoundRobinEmptyService(t *testing.T) {
	assert := assert.New(t)
	roundRobin := NewWeightRoundRobin()

	_, err := roundRobin.Pick("", []*registry.Service{})
	assert.EqualError(err, EmptyServiceErr.Error())
}

func TestWeightNilService(t *testing.T) {
	assert := assert.New(t)
	roundRobin := NewWeightRoundRobin()

	_, err := roundRobin.Pick("", []*registry.Service{
		nil,
	})
	assert.EqualError(err, NilServiceErr.Error())
}
