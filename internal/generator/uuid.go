package generator

import "github.com/google/uuid"

type uuidGenerator struct{}

// Generate draws entropy from ctx.Rand (rather than crypto/rand) so a
// fixed DSL seed reproduces the same UUIDs, per DESIGN.md's determinism
// goal.
func (g *uuidGenerator) Generate(ctx *Context) (any, error) {
	id, err := uuid.NewRandomFromReader(ctx.Rand)
	if err != nil {
		return nil, err
	}
	return id.String(), nil
}
