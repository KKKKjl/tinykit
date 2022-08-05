package filter

import (
	"sort"

	"github.com/KKKKjl/tinykit/internal/context"
)

type FilterChains struct {
	chain Chain
}

func NewFilterChains() *FilterChains {
	return &FilterChains{
		make(Chain, 0),
	}
}

func (f *FilterChains) Compose() HandleFilter {
	return func(ctx context.HttpContext, next Next) {
		var (
			dispatch Next
			index    int
		)

		// sort chain by priority
		sort.Sort(f.chain)

		last := f.chain.Len()

		// It executes the pending handlers in the chain inside the calling handler.
		dispatch = func(ctx context.HttpContext) {
			// early abort, return directly
			if ctx.IsAbort {
				return
			}

			// handle error
			if ctx.Err != nil {
				ctx.AbortWithMsg(ctx.Err.Error())
				return
			}

			if index == last {
				next(ctx)
				return
			}

			index++

			f.chain[index-1].Handle(ctx, dispatch)
		}

		dispatch(ctx)
	}
}

func (f *FilterChains) Use(handler ...Handler) {
	f.chain = append(f.chain, handler...)
}
