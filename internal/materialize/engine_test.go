package materialize_test

import (
	"encoding/json"
	"testing"

	"github.com/lp-peg/fixture-bank/internal/dsl"
	"github.com/lp-peg/fixture-bank/internal/materialize"
	"github.com/lp-peg/fixture-bank/internal/output"
)

const nestedDSL = `
entity: user
count: 4
seed: 42
fields:
  id: {type: uuid, generator: uuid_v4}
  level: {type: int, generator: fixed, value: 50}

relations:
  orders:
    entity: order
    count_per_parent: {min: 1, max: 3}
    fields:
      id: {type: uuid, generator: uuid_v4}
      user_id: {type: uuid, generator: ref, ref: parent.id}
    relations:
      items:
        entity: order_item
        count_per_parent: {fixed: 2}
        fields:
          id: {type: uuid, generator: uuid_v4}
          order_id: {type: uuid, generator: ref, ref: parent.id}
`

func mustParse(t *testing.T, y string) *dsl.Fixture {
	t.Helper()
	fx, err := dsl.Parse([]byte(y))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	return fx
}

func TestRun_RootCount(t *testing.T) {
	fx := mustParse(t, nestedDSL)
	records, err := materialize.Run(fx, materialize.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 4 {
		t.Fatalf("len(records) = %d, want 4 (DSL count)", len(records))
	}
}

func TestRun_CountOverride(t *testing.T) {
	fx := mustParse(t, nestedDSL)
	n := 7
	records, err := materialize.Run(fx, materialize.Options{Count: &n})
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 7 {
		t.Fatalf("len(records) = %d, want 7 (--count override)", len(records))
	}
}

func TestRun_RelationsRespectCountPerParentAndRef(t *testing.T) {
	fx := mustParse(t, nestedDSL)
	records, err := materialize.Run(fx, materialize.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, user := range records {
		orders := user.Children["orders"]
		if len(orders) < 1 || len(orders) > 3 {
			t.Fatalf("orders count = %d, want in [1,3]", len(orders))
		}
		for _, order := range orders {
			if order.Values["user_id"] != user.Values["id"] {
				t.Errorf("order.user_id = %v, want %v (parent.id)", order.Values["user_id"], user.Values["id"])
			}
			items := order.Children["items"]
			if len(items) != 2 {
				t.Fatalf("items count = %d, want fixed 2", len(items))
			}
			for _, item := range items {
				if item.Values["order_id"] != order.Values["id"] {
					t.Errorf("item.order_id = %v, want %v (parent.id)", item.Values["order_id"], order.Values["id"])
				}
			}
		}
	}
}

func TestRun_DeterministicGivenSameSeed(t *testing.T) {
	fx1 := mustParse(t, nestedDSL)
	fx2 := mustParse(t, nestedDSL)

	r1, err := materialize.Run(fx1, materialize.Options{})
	if err != nil {
		t.Fatal(err)
	}
	r2, err := materialize.Run(fx2, materialize.Options{})
	if err != nil {
		t.Fatal(err)
	}

	j1, err := output.RenderJSON(r1)
	if err != nil {
		t.Fatal(err)
	}
	j2, err := output.RenderJSON(r2)
	if err != nil {
		t.Fatal(err)
	}
	if string(j1) != string(j2) {
		t.Fatalf("same seed produced different output:\n%s\n---\n%s", j1, j2)
	}

	// Sanity: it's not just empty/degenerate output.
	var decoded []map[string]any
	if err := json.Unmarshal(j1, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) != 4 {
		t.Fatalf("decoded len = %d, want 4", len(decoded))
	}
}

func TestRun_SeedOverride(t *testing.T) {
	fx := mustParse(t, nestedDSL)
	var seedA int64 = 100
	var seedB int64 = 200

	rA, err := materialize.Run(fx, materialize.Options{Seed: &seedA})
	if err != nil {
		t.Fatal(err)
	}
	rB, err := materialize.Run(fx, materialize.Options{Seed: &seedB})
	if err != nil {
		t.Fatal(err)
	}
	if rA[0].Values["id"] == rB[0].Values["id"] {
		t.Fatalf("different seeds produced identical output")
	}
}
