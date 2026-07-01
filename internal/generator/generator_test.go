package generator_test

import (
	"math/rand"
	"testing"

	"github.com/lp-peg/fixture-bank/internal/dsl"
	"github.com/lp-peg/fixture-bank/internal/ferr"
	"github.com/lp-peg/fixture-bank/internal/generator"
)

func ptrF(f float64) *float64 { return &f }
func ptrI(i int) *int         { return &i }
func ptrI64(i int64) *int64   { return &i }

func newCtx(seed int64) *generator.Context {
	return &generator.Context{Rand: rand.New(rand.NewSource(seed))}
}

func TestFixed(t *testing.T) {
	g, err := generator.New("user", "level", dsl.Field{
		Type: dsl.TypeInt, Generator: dsl.GenFixed, Value: 50, Unique: dsl.UniqueNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	v, err := g.Generate(newCtx(1))
	if err != nil {
		t.Fatal(err)
	}
	if v != 50 {
		t.Errorf("Generate() = %v, want 50", v)
	}
}

func TestRandomInt_Range(t *testing.T) {
	g, err := generator.New("user", "n", dsl.Field{
		Type: dsl.TypeInt, Generator: dsl.GenRandomInt, Min: ptrF(1), Max: ptrF(3), Unique: dsl.UniqueNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := newCtx(1)
	for i := 0; i < 200; i++ {
		v, err := g.Generate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		n := v.(int64)
		if n < 1 || n > 3 {
			t.Fatalf("Generate() = %d, want in [1,3]", n)
		}
	}
}

func TestSample_OnlyFromCandidates(t *testing.T) {
	g, err := generator.New("order", "status", dsl.Field{
		Type: dsl.TypeString, Generator: dsl.GenSample,
		From: []any{"pending", "paid", "shipped"}, Unique: dsl.UniqueNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := newCtx(2)
	valid := map[any]bool{"pending": true, "paid": true, "shipped": true}
	for i := 0; i < 50; i++ {
		v, err := g.Generate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !valid[v] {
			t.Fatalf("Generate() = %v, not in candidate set", v)
		}
	}
}

func TestSequence_Increments(t *testing.T) {
	g, err := generator.New("user", "n", dsl.Field{
		Type: dsl.TypeInt, Generator: dsl.GenSequence, Start: ptrI64(10), Step: ptrI64(5), Unique: dsl.UniqueNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := newCtx(1)
	want := []int64{10, 15, 20}
	for i, w := range want {
		v, err := g.Generate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if v != w {
			t.Errorf("call %d: Generate() = %v, want %v", i, v, w)
		}
	}
}

func TestUUIDV4_DeterministicPerSeed(t *testing.T) {
	build := func() generator.Generator {
		g, err := generator.New("user", "id", dsl.Field{Type: dsl.TypeUUID, Generator: dsl.GenUUIDV4, Unique: dsl.UniqueNone})
		if err != nil {
			t.Fatal(err)
		}
		return g
	}
	v1, err := build().Generate(newCtx(42))
	if err != nil {
		t.Fatal(err)
	}
	v2, err := build().Generate(newCtx(42))
	if err != nil {
		t.Fatal(err)
	}
	if v1 != v2 {
		t.Errorf("same seed produced different UUIDs: %v vs %v", v1, v2)
	}
}

func TestRef_ResolvesParentField(t *testing.T) {
	g, err := generator.New("order", "user_id", dsl.Field{
		Type: dsl.TypeUUID, Generator: dsl.GenRef, Ref: "parent.id", Unique: dsl.UniqueNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := newCtx(1).WithParent(map[string]any{"id": "abc-123"})
	v, err := g.Generate(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if v != "abc-123" {
		t.Errorf("Generate() = %v, want abc-123", v)
	}
}

func TestRef_NoParent(t *testing.T) {
	g, err := generator.New("user", "x", dsl.Field{Type: dsl.TypeUUID, Generator: dsl.GenRef, Ref: "parent.id", Unique: dsl.UniqueNone})
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.Generate(newCtx(1))
	assertErrorType(t, err, ferr.TypeUnknownRef)
}

func TestRef_UnknownParentField(t *testing.T) {
	g, err := generator.New("order", "user_id", dsl.Field{Type: dsl.TypeUUID, Generator: dsl.GenRef, Ref: "parent.nope", Unique: dsl.UniqueNone})
	if err != nil {
		t.Fatal(err)
	}
	ctx := newCtx(1).WithParent(map[string]any{"id": "abc"})
	_, err = g.Generate(ctx)
	assertErrorType(t, err, ferr.TypeUnknownRef)
}

type fakeDB struct {
	queryResult []any
	queryErr    error
	existsFn    func(entity, column string, value any) (bool, error)
	queryCalls  int
}

func (f *fakeDB) Query(query string) ([]any, error) {
	f.queryCalls++
	return f.queryResult, f.queryErr
}

func (f *fakeDB) Exists(entity, column string, value any) (bool, error) {
	return f.existsFn(entity, column, value)
}

func TestPoolRef_QueriesOnceAndDrawsFromPool(t *testing.T) {
	db := &fakeDB{queryResult: []any{"p1", "p2", "p3"}}
	g, err := generator.New("order_item", "product_id", dsl.Field{
		Type: dsl.TypeUUID, Generator: dsl.GenPoolRef, Query: "SELECT id FROM products", Unique: dsl.UniqueNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := &generator.Context{Rand: rand.New(rand.NewSource(1)), DB: db}
	valid := map[any]bool{"p1": true, "p2": true, "p3": true}
	for i := 0; i < 20; i++ {
		v, err := g.Generate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if !valid[v] {
			t.Fatalf("Generate() = %v, not from pool", v)
		}
	}
	if db.queryCalls != 1 {
		t.Errorf("query issued %d times, want exactly 1", db.queryCalls)
	}
}

func TestPoolRef_EmptyPool(t *testing.T) {
	db := &fakeDB{queryResult: nil}
	g, err := generator.New("order_item", "product_id", dsl.Field{
		Type: dsl.TypeUUID, Generator: dsl.GenPoolRef, Query: "SELECT id FROM products WHERE 1=0", Unique: dsl.UniqueNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := &generator.Context{Rand: rand.New(rand.NewSource(1)), DB: db}
	_, err = g.Generate(ctx)
	assertErrorType(t, err, ferr.TypeEmptyPool)
}

func TestPoolRef_NoDBConfigured(t *testing.T) {
	g, err := generator.New("order_item", "product_id", dsl.Field{
		Type: dsl.TypeUUID, Generator: dsl.GenPoolRef, Query: "SELECT id FROM products", Unique: dsl.UniqueNone,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.Generate(newCtx(1))
	assertErrorType(t, err, ferr.TypeDBError)
}

func TestUnique_Batch_AvoidsCollisionsWithinBatch(t *testing.T) {
	g, err := generator.New("user", "status", dsl.Field{
		Type: dsl.TypeString, Generator: dsl.GenSample, From: []any{"a", "b"}, Unique: dsl.UniqueBatch,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := newCtx(1)
	seen := map[any]bool{}
	for i := 0; i < 2; i++ {
		v, err := g.Generate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if seen[v] {
			t.Fatalf("value %v generated twice under unique: batch", v)
		}
		seen[v] = true
	}
}

func TestUnique_Batch_ExhaustedRetries(t *testing.T) {
	g, err := generator.New("user", "status", dsl.Field{
		Type: dsl.TypeString, Generator: dsl.GenSample, From: []any{"a", "b"}, Unique: dsl.UniqueBatch,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := newCtx(1)
	// Only 2 possible values; the 3rd request must exhaust retries.
	if _, err := g.Generate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := g.Generate(ctx); err != nil {
		t.Fatal(err)
	}
	_, err = g.Generate(ctx)
	assertErrorType(t, err, ferr.TypeUniqueRetryExhausted)
}

func TestUnique_DB_SkipsExistingValues(t *testing.T) {
	// Each iteration builds a fresh generator: unique also enforces
	// batch-uniqueness, so reusing one generator/claimed-set across many
	// calls with only one ever-valid candidate would itself exhaust
	// retries. What's under test here is purely the DB-exists skip.
	db := &fakeDB{existsFn: func(entity, column string, value any) (bool, error) {
		return value == "taken@example.com", nil
	}}
	for i := 0; i < 20; i++ {
		g, err := generator.New("user", "email", dsl.Field{
			Type: dsl.TypeString, Generator: dsl.GenSample,
			From: []any{"taken@example.com", "free@example.com"}, Unique: dsl.UniqueDB,
		})
		if err != nil {
			t.Fatal(err)
		}
		ctx := &generator.Context{Rand: rand.New(rand.NewSource(int64(i))), DB: db}
		v, err := g.Generate(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if v == "taken@example.com" {
			t.Fatalf("unique: db returned a value that exists in the DB")
		}
	}
}

func TestUnique_DB_NoDBConfigured(t *testing.T) {
	g, err := generator.New("user", "email", dsl.Field{
		Type: dsl.TypeString, Generator: dsl.GenFixed, Value: "x@example.com", Unique: dsl.UniqueDB,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = g.Generate(newCtx(1))
	assertErrorType(t, err, ferr.TypeDBError)
}

func assertErrorType(t *testing.T, err error, want string) {
	t.Helper()
	fe, ok := err.(*ferr.Error)
	if !ok {
		t.Fatalf("error is not *ferr.Error: %v", err)
	}
	if fe.ErrorType != want {
		t.Fatalf("ErrorType = %q, want %q", fe.ErrorType, want)
	}
}
