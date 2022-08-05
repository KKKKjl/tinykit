package proxy

import (
	"github.com/KKKKjl/tinykit/internal/rewrite"
)

type ProxyOption func(*Proxy)

func WithRules(rules map[string]string) ProxyOption {
	return func(proxy *Proxy) {
		for k, v := range rules {
			rule, err := rewrite.NewRule(k, v)
			if err != nil {
				mainLog.Errorf("create rule error %v", err)
			}

			// proxy.ReWrite.AddRule(rewrite.Rule{
			// 	Pattern: "/api/v1/",
			// 	ReWrite: func(req *http.Request) (*url.URL, error) {
			// 		return url.Parse("http://localhost:8080/api/v1/")
			// 	},
			// })

			proxy.ReWrite.AddRule(rule)
		}
	}
}
