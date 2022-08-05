package rewrite

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReWrite(t *testing.T) {
	reWrite := NewReWrite()

	assert := assert.New(t)

	rulesMap := map[string]string{
		"/old":   "/new",
		"/old/*": "/new/b",
	}

	for k, v := range rulesMap {
		rule, err := NewRule(k, v)
		if err != nil {
			t.Errorf("create rule error %v", err)
		}

		reWrite.AddRule(rule)
	}

	cases := []struct {
		Origin   string
		Expected string
	}{
		{Origin: "/old", Expected: "/new"},
		{Origin: "/a/b", Expected: "/c"},
	}

	for _, v := range cases {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:8081%s", v.Origin), nil)
		if err != nil {
			t.Fatalf("create http request error %v", err)
		}

		if rule := reWrite.Match(req.URL.Path); rule != nil {
			target, err := rule.ReWrite(*req)
			if err != nil {
				t.Errorf("rewrite error %v", err)
			}

			assert.Equalf(v.Expected, target.Path, "expected %s, got %s", v.Expected, target.Path)
		}
	}
}
