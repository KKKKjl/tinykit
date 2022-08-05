package filter

import (
	"sort"
	"testing"
)

func TestSort(t *testing.T) {
	chain := Chain{
		{"handle1", 1, ReqMode, nil},
		{"handle2", 3, ReqMode, nil},
		{"handle3", 2, ReqMode, nil},
		{"handle4", 4, ReqMode, nil},
	}

	sort.Sort(chain)

	for k, v := range chain {
		if chain.Len()-k != v.Priority {
			t.Errorf("sort error, want %d, got %d", chain.Len()-k, v.Priority)
		}
	}

}
