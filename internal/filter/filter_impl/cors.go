package filter_impl

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/KKKKjl/tinykit/internal/context"
	"github.com/KKKKjl/tinykit/internal/filter"
)

type Cors struct {
	AllowOrigins       []string
	AllowMethods       []string
	AllowHeaders       []string
	AllowExposeHeaders []string
	AllowAllOrigin     bool
	AllowAllMethods    bool
	AllowAllHeaders    bool
}

func CORS() filter.HandleFilter {
	config := &Cors{
		AllowOrigins:       []string{},
		AllowMethods:       []string{"GET", "POST"},
		AllowHeaders:       []string{"Origin", "Content-Length", "Content-Type"},
		AllowExposeHeaders: []string{"X-TinyKit-Trace-Id", "Authorization", "X-Ratelimit-Limit", "X-Ratelimit-Reset"},
		AllowAllOrigin:     true,
	}
	return newCors(config)
}

func (cors *Cors) handlePreflight(ctx context.HttpContext) {
	allowHeadersStr := strings.Join(cors.AllowHeaders, ",")
	cors.SetHeader(ctx, "Access-Control-Allow-Headers", allowHeadersStr)

	allowMethodsStr := strings.Join(cors.AllowMethods, ",")
	cors.SetHeader(ctx, "Access-Control-Allow-Methods", allowMethodsStr)

	allowExposeHeadersStr := strings.Join(cors.AllowExposeHeaders, ",")
	cors.SetHeader(ctx, "Access-Control-Expose-Headers", allowExposeHeadersStr)

	if cors.AllowAllOrigin {
		cors.SetHeader(ctx, "Access-Control-Allow-Origin", "*")
	} else if len(cors.AllowOrigins) > 0 {
		str := strings.Join(cors.AllowOrigins, ",")
		cors.SetHeader(ctx, "Access-Control-Allow-Origin", str)
	}
}

func (cors *Cors) SetHeader(ctx context.HttpContext, key string, value string) {
	ctx.Request.Header.Set(key, value)
}

// check if the origin is allowed
func (cors *Cors) validateOrigin(origin string) bool {
	if cors.AllowAllOrigin {
		return true
	}

	origin = strings.ToLower(origin)
	for _, v := range cors.AllowOrigins {
		if strings.ToLower(v) == origin {
			return true
		}
	}

	return false
}

func newCors(cors *Cors) filter.HandleFilter {
	return func(ctx context.HttpContext, next filter.Next) {
		origin := ctx.Request.Header.Get("origin")

		if origin != "" && cors.validateOrigin(origin) {
			ctx.AbortWithMsg(fmt.Sprintf("The request origin header %s not allowed", origin))
			return
		}

		cors.handlePreflight(ctx)
		if ctx.Method == "OPTIONS" {
			ctx.AbortWithStatus(http.StatusOK)
			return
		}

		next(ctx)
	}
}

func InitCorsLimit() filter.Handler {
	return filter.Handler{
		Name:     "cors",
		Priority: 1,
		Handle:   CORS(),
	}
}
