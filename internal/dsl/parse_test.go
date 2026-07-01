package dsl_test

import (
	"strings"
	"testing"

	"github.com/lp-peg/fixture-bank/internal/dsl"
	"github.com/lp-peg/fixture-bank/internal/ferr"
)

const fullExample = `
entity: user
count: 1
seed: 42
fields:
  id: {type: uuid, generator: uuid_v4}
  level: {type: int, generator: fixed, value: 50}
  premium_pass: {type: boolean, generator: fixed, value: true}
  email: {type: string, generator: faker, provider: email, unique: db}

relations:
  orders:
    entity: order
    count_per_parent: {min: 1, max: 3}
    fields:
      id: {type: uuid, generator: uuid_v4}
      user_id: {type: uuid, generator: ref, ref: parent.id}
      status: {type: string, generator: sample, from: [pending, paid, shipped]}
    relations:
      items:
        entity: order_item
        count_per_parent: {fixed: 2}
        fields:
          id: {type: uuid, generator: uuid_v4}
          order_id: {type: uuid, generator: ref, ref: parent.id}
          product_id:
            type: uuid
            generator: pool_ref
            query: "SELECT id FROM products WHERE status = 'active' LIMIT 200"
          quantity: {type: int, generator: random_int, min: 1, max: 5}
`

func TestParse_FullExample(t *testing.T) {
	fx, err := dsl.Parse([]byte(fullExample))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if fx.Entity != "user" {
		t.Errorf("Entity = %q, want user", fx.Entity)
	}
	if len(fx.Fields) != 4 {
		t.Errorf("len(Fields) = %d, want 4", len(fx.Fields))
	}
	// Declaration order must survive parsing.
	wantOrder := []string{"id", "level", "premium_pass", "email"}
	for i, want := range wantOrder {
		if fx.Fields[i].Name != want {
			t.Errorf("Fields[%d].Name = %q, want %q", i, fx.Fields[i].Name, want)
		}
	}

	orders, ok := findRelation(fx.Relations, "orders")
	if !ok {
		t.Fatalf("relations.orders not found")
	}
	if orders.Entity != "order" {
		t.Errorf("orders.Entity = %q, want order", orders.Entity)
	}
	if orders.CountPerParent.Min == nil || *orders.CountPerParent.Min != 1 {
		t.Errorf("orders.CountPerParent.Min = %v, want 1", orders.CountPerParent.Min)
	}

	items, ok := findRelation(orders.Relations, "items")
	if !ok {
		t.Fatalf("relations.orders.relations.items not found")
	}
	if items.CountPerParent.Fixed == nil || *items.CountPerParent.Fixed != 2 {
		t.Errorf("items.CountPerParent.Fixed = %v, want 2", items.CountPerParent.Fixed)
	}
}

func findRelation(relations dsl.Relations, name string) (dsl.Relation, bool) {
	for _, r := range relations {
		if r.Name == name {
			return r.Relation, true
		}
	}
	return dsl.Relation{}, false
}

func TestParse_DefaultsApplied(t *testing.T) {
	fx, err := dsl.Parse([]byte(`
entity: user
fields:
  id: {type: uuid, generator: uuid_v4}
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if fx.Count != 1 {
		t.Errorf("Count = %d, want default 1", fx.Count)
	}
	f, _ := fx.Fields.Get("id")
	if f.Unique != dsl.UniqueNone {
		t.Errorf("Unique = %q, want default none", f.Unique)
	}
}

func TestParse_SyntaxErrors(t *testing.T) {
	cases := []struct {
		name string
		yaml string
	}{
		{"missing entity", `
fields:
  id: {type: uuid, generator: uuid_v4}
`},
		{"missing fields", `
entity: user
`},
		{"unknown type", `
entity: user
fields:
  id: {type: bogus, generator: uuid_v4}
`},
		{"unknown generator", `
entity: user
fields:
  id: {type: uuid, generator: bogus}
`},
		{"fixed without value", `
entity: user
fields:
  id: {type: int, generator: fixed}
`},
		{"random_int without max", `
entity: user
fields:
  id: {type: int, generator: random_int, min: 1}
`},
		{"random_int min > max", `
entity: user
fields:
  id: {type: int, generator: random_int, min: 5, max: 1}
`},
		{"sample without from", `
entity: user
fields:
  id: {type: string, generator: sample}
`},
		{"faker without provider", `
entity: user
fields:
  id: {type: string, generator: faker}
`},
		{"ref without ref", `
entity: user
fields:
  id: {type: string, generator: ref}
`},
		{"pool_ref without query", `
entity: user
fields:
  id: {type: uuid, generator: pool_ref}
`},
		{"duplicate field", `
entity: user
fields:
  id: {type: uuid, generator: uuid_v4}
  id: {type: uuid, generator: uuid_v4}
`},
		{"relation missing count_per_parent", `
entity: user
fields:
  id: {type: uuid, generator: uuid_v4}
relations:
  orders:
    entity: order
    fields:
      id: {type: uuid, generator: uuid_v4}
`},
		{"invalid unique mode", `
entity: user
fields:
  id: {type: uuid, generator: uuid_v4, unique: sometimes}
`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := dsl.Parse([]byte(tc.yaml))
			if err == nil {
				t.Fatalf("Parse() error = nil, want syntax_error")
			}
			var fe *ferr.Error
			if !errorsAs(err, &fe) {
				t.Fatalf("error is not *ferr.Error: %v", err)
			}
			if fe.ErrorType != ferr.TypeSyntaxError {
				t.Errorf("ErrorType = %q, want %q", fe.ErrorType, ferr.TypeSyntaxError)
			}
		})
	}
}

func errorsAs(err error, target **ferr.Error) bool {
	fe, ok := err.(*ferr.Error)
	if ok {
		*target = fe
	}
	return ok
}

func TestParse_InvalidYAML(t *testing.T) {
	_, err := dsl.Parse([]byte("entity: [this is not\n  valid yaml"))
	if err == nil || !strings.Contains(err.Error(), "syntax_error") {
		t.Fatalf("Parse() error = %v, want syntax_error", err)
	}
}
