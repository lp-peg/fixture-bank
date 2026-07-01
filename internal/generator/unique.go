package generator

import (
	"github.com/lp-peg/fixture-bank/internal/dsl"
	"github.com/lp-peg/fixture-bank/internal/ferr"
)

// maxUniqueRetries mirrors DSL_SPEC.md ¤2.3: "既定10回" for unique: db retries.
const maxUniqueRetries = 10

// uniqueWrapper implements DSL_SPEC.md ¤2.3's `unique: batch|db` behavior.
// It is instantiated once per field and reused across the whole batch, so
// `claimed` naturally tracks every value produced so far in this run.
type uniqueWrapper struct {
	inner   Generator
	entity  string
	column  string
	mode    dsl.UniqueMode
	claimed map[any]struct{}
}

func (u *uniqueWrapper) Generate(ctx *Context) (any, error) {
	if u.claimed == nil {
		u.claimed = map[any]struct{}{}
	}
	for i := 0; i < maxUniqueRetries; i++ {
		val, err := u.inner.Generate(ctx)
		if err != nil {
			return nil, err
		}
		if _, dup := u.claimed[val]; dup {
			continue
		}
		if u.mode == dsl.UniqueDB {
			if ctx.DB == nil {
				return nil, ferr.New(ferr.TypeDBError, "unique: db requires a database connection (%s.%s)", u.entity, u.column)
			}
			exists, err := ctx.DB.Exists(u.entity, u.column, val)
			if err != nil {
				return nil, ferr.New(ferr.TypeDBError, "unique: db exists check failed: %v", err)
			}
			if exists {
				continue
			}
		}
		u.claimed[val] = struct{}{}
		return val, nil
	}
	return nil, ferr.New(ferr.TypeUniqueRetryExhausted, "%s.%s: exhausted %d retries", u.entity, u.column, maxUniqueRetries)
}
