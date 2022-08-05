package balance

import (
	"testing"

	"github.com/KKKKjl/tinykit/internal/registry"
	"github.com/stretchr/testify/assert"
)

func TestRoundRobin(t *testing.T) {
	services := []*registry.Service{
		{
			Addr: "localhost:8080",
		},
		{
			Addr: "localhost:8081",
		},
		{
			Addr: "localhost:8082",
		},
	}
	assert := assert.New(t)

	roundRobin := NewRoundRobin()

	expects := []struct {
		Expected string
	}{
		{Expected: "localhost:8081"},
		{Expected: "localhost:8082"},
		{Expected: "localhost:8080"},
		{Expected: "localhost:8081"},
		{Expected: "localhost:8082"},
		{Expected: "localhost:8080"},
	}

	for _, v := range expects {
		service, err := roundRobin.Pick(v.Expected, services)

		assert.Nil(err)
		assert.Equalf(service.Addr, v.Expected, "expected %s, got %s", v.Expected, service.Addr)
	}
}

func TestRoundRobinEmptyService(t *testing.T) {
	assert := assert.New(t)
	roundRobin := NewRoundRobin()

	_, err := roundRobin.Pick("", []*registry.Service{})
	assert.EqualError(err, EmptyServiceErr.Error())
}
