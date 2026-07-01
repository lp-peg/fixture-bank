package generator

// sequenceGenerator is instantiated once per field and reused across every
// record in the batch, so it can hold its own running counter directly
// instead of needing shared batch state.
type sequenceGenerator struct {
	start, step int64
	next        int64
	started     bool
}

func (g *sequenceGenerator) Generate(ctx *Context) (any, error) {
	if !g.started {
		g.next = g.start
		g.started = true
	} else {
		g.next += g.step
	}
	return g.next, nil
}
