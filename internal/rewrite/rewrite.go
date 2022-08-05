package rewrite

import (
	"net/http"
	"net/url"
	"regexp"
	"sync"
)

type (
	Rule struct {
		Pattern string
		To      string
		Enable  bool
		*regexp.Regexp
	}

	ReWrite struct {
		rules []*Rule
		mu    sync.RWMutex
	}
)

func NewReWrite() *ReWrite {
	return &ReWrite{
		rules: make([]*Rule, 0),
	}
}

func (r *ReWrite) AddRule(rule ...*Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.rules = append(r.rules, rule...)
}

func (r *ReWrite) RemoveRule(rule *Rule) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, v := range r.rules {
		if v.Pattern == rule.Pattern && v.To == rule.To && v.Regexp.String() == rule.Regexp.String() {
			r.rules = append(r.rules[:i], r.rules[i+1:]...)
			break
		}
	}
}

func (r *ReWrite) UpdateRule(rule *Rule) {
	panic("implement me")
}

// Match return the matched rule.
func (r *ReWrite) Match(path string) *Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, v := range r.rules {
		if v.Enable && v.Regexp.MatchString(path) {
			return v
		}
	}

	return nil
}

func NewRule(pattern string, to string) (*Rule, error) {
	reg, err := regexp.Compile(pattern)
	if err != nil {
		return nil, err
	}

	return &Rule{
		Pattern: pattern,
		To:      to,
		Enable:  true,
		Regexp:  reg,
	}, nil
}

// Rewrite rewrites the request path.
func (r *Rule) ReWrite(req http.Request) (*url.URL, error) {
	return url.Parse(r.To)
}
