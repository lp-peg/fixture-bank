package generator

import "github.com/lp-peg/fixture-bank/internal/ferr"

// poolRefGenerator implements DSL_SPEC.md ¤2.2: the query runs once (on
// first Generate call) and results are cached as a pool to draw from,
// regardless of how many records are generated.
type poolRefGenerator struct {
	query   string
	pool    []any
	fetched bool
}

func (g *poolRefGenerator) Generate(ctx *Context) (any, error) {
	if !g.fetched {
		if ctx.DB == nil {
			return nil, ferr.New(ferr.TypeDBError, "pool_ref requires a database connection (query: %s)", g.query)
		}
		pool, err := ctx.DB.Query(g.query)
		if err != nil {
			return nil, ferr.New(ferr.TypeDBError, "pool_ref query failed: %v", err)
		}
		if len(pool) == 0 {
			return nil, ferr.New(ferr.TypeEmptyPool, "pool_ref query returned no rows: %s", g.query)
		}
		g.pool = pool
		g.fetched = true
	}
	return g.pool[ctx.Rand.Intn(len(g.pool))], nil
}
