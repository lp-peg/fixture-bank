package generator

type fixedGenerator struct{ value any }

func (g *fixedGenerator) Generate(ctx *Context) (any, error) {
	return g.value, nil
}
