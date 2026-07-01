package generator

import (
	"github.com/brianvoe/gofakeit/v6"

	"github.com/lp-peg/fixture-bank/internal/ferr"
)

// fakerProviders maps DSL_SPEC.md ¤2.1 `faker` provider names to gofakeit
// calls. Kept intentionally small (DESIGN.md's "表現力を絞る" principle);
// extend as real fixtures need more providers.
var fakerProviders = map[string]func(f *gofakeit.Faker) any{
	"email":      func(f *gofakeit.Faker) any { return f.Email() },
	"name":       func(f *gofakeit.Faker) any { return f.Name() },
	"first_name": func(f *gofakeit.Faker) any { return f.FirstName() },
	"last_name":  func(f *gofakeit.Faker) any { return f.LastName() },
	"username":   func(f *gofakeit.Faker) any { return f.Username() },
	"phone":      func(f *gofakeit.Faker) any { return f.Phone() },
	"address":    func(f *gofakeit.Faker) any { return f.Address().Address },
	"company":    func(f *gofakeit.Faker) any { return f.Company() },
	"word":       func(f *gofakeit.Faker) any { return f.Word() },
	"sentence":   func(f *gofakeit.Faker) any { return f.Sentence(8) },
	"url":        func(f *gofakeit.Faker) any { return f.URL() },
}

type fakerGenerator struct {
	provider string
	faker    *gofakeit.Faker
}

// Generate draws from ctx.Rand (via gofakeit.NewCustom) so output is
// reproducible for a fixed DSL seed.
func (g *fakerGenerator) Generate(ctx *Context) (any, error) {
	fn, ok := fakerProviders[g.provider]
	if !ok {
		return nil, ferr.New(ferr.TypeSyntaxError, "faker: unknown provider %q", g.provider)
	}
	if g.faker == nil {
		g.faker = gofakeit.NewCustom(ctx.Rand)
	}
	return fn(g.faker), nil
}
