package generator

import (
	"strings"

	"github.com/lp-peg/fixture-bank/internal/ferr"
)

type refGenerator struct{ ref string }

// Generate resolves `ref: parent.<field>` against ctx.Parent, per
// DSL_SPEC.md ¤3.1. Multi-level ancestor refs (`ancestor.N`) are
// intentionally out of scope for v1 (DSL_SPEC.md ¤7).
func (g *refGenerator) Generate(ctx *Context) (any, error) {
	parts := strings.SplitN(g.ref, ".", 2)
	if len(parts) != 2 || parts[0] != "parent" {
		return nil, ferr.New(ferr.TypeUnknownRef, "ref %q: only \"parent.<field>\" is supported", g.ref)
	}
	if ctx.Parent == nil {
		return nil, ferr.New(ferr.TypeUnknownRef, "ref %q: no parent in scope (used on a root entity?)", g.ref)
	}
	val, ok := ctx.Parent[parts[1]]
	if !ok {
		return nil, ferr.New(ferr.TypeUnknownRef, "ref %q: parent has no field %q", g.ref, parts[1])
	}
	return val, nil
}
