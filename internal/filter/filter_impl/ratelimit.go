package filter_impl

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"

	"github.com/KKKKjl/tinykit/internal/context"
	"github.com/KKKKjl/tinykit/internal/filter"
	"github.com/KKKKjl/tinykit/utils"
)

var (
	limit = NewRateLimit(1)
)

type RateLimit struct {
	rate    float64 // requests per second
	brust   int     // burst size
	buckets sync.Map
}

func NewRateLimit(rate float64) *RateLimit {
	if rate < 0 {
		panic(fmt.Sprintf("cap %v < 0", rate))
	}

	return &RateLimit{
		rate:  rate,
		brust: int(math.Max(1, rate)),
	}
}

func (r *RateLimit) GetMax() float64 {
	return r.rate
}

// func (r *RateLimit) set(key string) *rate.Limiter {
// 	limiter := rate.NewLimiter(rate.Limit(r.rate), r.brust)
// 	r.buckets[key] = limiter
// 	return limiter
// }

func (r *RateLimit) Take(key string) *rate.Limiter {
	limit, _ := r.buckets.LoadOrStore(key, rate.NewLimiter(rate.Every(time.Second), r.brust))
	return limit.(*rate.Limiter)
}

func (r *RateLimit) Avabile(key string) (bool, *rate.Reservation) {
	limiter := limit.Take(key).Reserve()
	return limiter.OK(), limiter
}

func RateLimitFilter() filter.HandleFilter {
	return func(ctx context.HttpContext, next filter.Next) {
		ip, err := utils.GetIPAddr(ctx.Request)
		if err != nil {
			ctx.Error(err)
			return
		}

		ok, limiter := limit.Avabile(ip)
		if !ok {
			ctx.Error(errors.New("Cannot provide the requested token."))
			return
		}
		defer limiter.Cancel()

		if limiter.Delay().Seconds() > 0 {
			resetUnixTime := time.Now().Unix() + int64(math.Ceil(limiter.Delay().Seconds()))

			ctx.SetResponseHeader("X-Ratelimit-Limit", fmt.Sprintf("%.2f", limit.GetMax()))
			ctx.SetResponseHeader("X-Ratelimit-Reset", strconv.FormatInt(resetUnixTime, 10))
			ctx.AbortWithMsg("Too many requests, please try again in serveral seconds.")
			return
		}

		next(ctx)
	}
}

func InitRateLimit() filter.Handler {
	return filter.Handler{
		Name:     "ratelimit",
		Priority: 2,
		Handle:   RateLimitFilter(),
	}
}
