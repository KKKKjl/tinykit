package server

import (
	"fmt"
	"reflect"
	"time"

	"github.com/KKKKjl/tinykit/internal/filter"
	"github.com/KKKKjl/tinykit/internal/filter/filter_impl"
)

type Option func(*GatewayServer)

var h = make(map[string]interface{})

func init() {
	h["ratelimit"] = filter_impl.InitRateLimit
	h["cors"] = filter_impl.InitCorsLimit
}

func WithFilters(chains ...string) Option {
	return func(gs *GatewayServer) {
		gs.chains = filter.NewFilterChains()

		for _, v := range chains {
			val, ok := h[v]
			if !ok {
				panic(fmt.Sprintf("%s not registered", v))
			}

			var mul = reflect.ValueOf(val)
			if mul.Kind() != reflect.Func {
				panic(fmt.Sprintf("%s not a function", v))
			}

			var res = mul.Call(nil)

			filter, ok := res[0].Interface().(filter.Handler)
			if !ok {
				panic(fmt.Sprintf("not"))
			}

			gs.chains.Use(filter)
		}
	}
}

func WithTimeout(timeout time.Duration) Option {
	return func(gs *GatewayServer) {
		gs.Timeout = timeout
	}
}
