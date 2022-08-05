package pubsub

import (
	"hash/fnv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFnv32(t *testing.T) {
	assert := assert.New(t)
	key := []byte("Hello World")

	hasher := fnv.New32()
	_, err := hasher.Write(key)

	assert.Nil(err)
	assert.Equal(fnv32(string(key)), hasher.Sum32())
}
