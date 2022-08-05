package filter

import (
	"github.com/KKKKjl/tinykit/internal/context"
)

var (
	ReqMode  HandleMode = "RequestChain"
	RespMode HandleMode = "ResponseChain"
)

type (
	HandleMode string

	Next func(context.HttpContext)

	HandleFilter func(context.HttpContext, Next)

	Handler struct {
		Name     string     // 过滤器名字
		Priority int        // 优先级
		Type     HandleMode // 过滤器类型
		Handle   HandleFilter
	}

	Chain []Handler // chain is a list of Handlers
)

func (c Chain) Len() int {
	return len(c)
}

func (c Chain) Less(i, j int) bool {
	return c[i].Priority > c[j].Priority
}

func (c Chain) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
