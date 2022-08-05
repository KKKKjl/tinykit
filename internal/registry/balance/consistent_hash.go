package balance

import (
	"hash/crc32"
	"sort"
	"strconv"

	"github.com/KKKKjl/tinykit/internal/registry"
)

// Hash maps bytes to uint32
type Hash func(data []byte) uint32

type Map struct {
	hash     Hash
	replicas int
	keys     []int // Sorted
	hashMap  map[int]string
}

func init() {
	balancer[CONSISTENT_HASHING] = NewConsistentHashing(10, nil)
}

func NewConsistentHashing(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

func (m *Map) Add(service ...*registry.Service) {
	for _, v := range service {
		if v == nil || len(v.Addr) == 0 {
			continue
		}

		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + v.Addr)))
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = v.Addr
		}
	}

	sort.Ints(m.keys)
}

func (m *Map) Remove(server *registry.Service) {
	for i := 0; i < m.replicas; i++ {
		hash := int(m.hash([]byte(strconv.Itoa(i) + server.Addr)))

		if _, ok := m.hashMap[hash]; ok {
			delete(m.hashMap, hash)
		}
	}
}

func (m *Map) Pick(key string, services []*registry.Service) (*registry.Service, error) {
	if len(m.keys) == 0 {
		return nil, EmptyServiceErr
	}

	hash := int(m.hash([]byte(key)))

	// Binary search for appropriate replica.
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})

	service := registry.Service{
		Addr: m.hashMap[m.keys[idx%len(m.keys)]],
	}

	return &service, nil
}

func (m *Map) Scheme() string {
	return "consistent_hashing"
}
