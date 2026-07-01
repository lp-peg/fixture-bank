package generator

type sampleGenerator struct{ from []any }

func (g *sampleGenerator) Generate(ctx *Context) (any, error) {
	return g.from[ctx.Rand.Intn(len(g.from))], nil
}
