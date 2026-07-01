package generator

import "math"

type randomIntGenerator struct{ min, max int64 }

func (g *randomIntGenerator) Generate(ctx *Context) (any, error) {
	span := g.max - g.min + 1
	return g.min + ctx.Rand.Int63n(span), nil
}

type randomFloatGenerator struct {
	min, max  float64
	precision int
}

func (g *randomFloatGenerator) Generate(ctx *Context) (any, error) {
	v := g.min + ctx.Rand.Float64()*(g.max-g.min)
	scale := math.Pow(10, float64(g.precision))
	return math.Round(v*scale) / scale, nil
}
