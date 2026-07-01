package materialize

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/lp-peg/fixture-bank/internal/dsl"
	"github.com/lp-peg/fixture-bank/internal/generator"
)

// Options overrides DSL-declared values at materialize time.
// DSL_SPEC.md ¤4: "--count はルートエンティティの生成件数のみを上書きする".
type Options struct {
	// Count overrides the root entity's generation count. nil uses the
	// DSL's own `count`.
	Count *int
	// Seed overrides the DSL's `seed`. nil uses the DSL's own `seed`, or a
	// time-derived seed if the DSL doesn't set one either.
	Seed *int64
	// DB backs pool_ref and unique:db. nil is fine as long as the fixture
	// uses neither.
	DB generator.DB
}

// Run generates fixture.Count (or opts.Count, if set) root records, each
// with its full relations tree, per the topological (parent → child) order
// required by DSL_SPEC.md ¤3.3.
func Run(fx *dsl.Fixture, opts Options) ([]*Record, error) {
	plan, err := buildEntityPlan(fx.Entity, fx.Fields, fx.Relations)
	if err != nil {
		return nil, err
	}

	seed := time.Now().UnixNano()
	if fx.Seed != nil {
		seed = *fx.Seed
	}
	if opts.Seed != nil {
		seed = *opts.Seed
	}
	rng := rand.New(rand.NewSource(seed))

	count := fx.Count
	if opts.Count != nil {
		count = *opts.Count
	}

	ctx := &generator.Context{Rand: rng, DB: opts.DB}
	records := make([]*Record, 0, count)
	for i := 0; i < count; i++ {
		rec, err := plan.generateOne(ctx)
		if err != nil {
			return nil, err
		}
		records = append(records, rec)
	}
	return records, nil
}

// entityPlan is a compiled (generator-instantiated) form of a dsl.Fixture
// or dsl.Relation. Building it once and reusing it across every generated
// record is what lets stateful generators (sequence, unique, pool_ref)
// hold correct per-field state across a whole batch.
type entityPlan struct {
	entity    string
	fields    []fieldPlan
	relations []relationPlan
}

type fieldPlan struct {
	name string
	gen  generator.Generator
}

type relationPlan struct {
	name      string
	countSpec dsl.CountSpec
	child     *entityPlan
}

func buildEntityPlan(entity string, fields dsl.Fields, relations dsl.Relations) (*entityPlan, error) {
	plan := &entityPlan{entity: entity}
	for _, fe := range fields {
		gen, err := generator.New(entity, fe.Name, fe.Field)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", entity, fe.Name, err)
		}
		plan.fields = append(plan.fields, fieldPlan{name: fe.Name, gen: gen})
	}
	for _, re := range relations {
		childPlan, err := buildEntityPlan(re.Relation.Entity, re.Relation.Fields, re.Relation.Relations)
		if err != nil {
			return nil, err
		}
		plan.relations = append(plan.relations, relationPlan{
			name:      re.Name,
			countSpec: re.Relation.CountPerParent,
			child:     childPlan,
		})
	}
	return plan, nil
}

func (p *entityPlan) generateOne(ctx *generator.Context) (*Record, error) {
	values := make(map[string]any, len(p.fields))
	fieldOrder := make([]string, 0, len(p.fields))
	for _, fp := range p.fields {
		v, err := fp.gen.Generate(ctx)
		if err != nil {
			return nil, fmt.Errorf("%s.%s: %w", p.entity, fp.name, err)
		}
		values[fp.name] = v
		fieldOrder = append(fieldOrder, fp.name)
	}

	rec := &Record{Entity: p.entity, Values: values, FieldOrder: fieldOrder}
	if len(p.relations) == 0 {
		return rec, nil
	}

	rec.Children = make(map[string][]*Record, len(p.relations))
	rec.RelationOrder = make([]string, 0, len(p.relations))
	childCtx := ctx.WithParent(values)
	for _, rp := range p.relations {
		n := resolveCount(rp.countSpec, ctx.Rand)
		children := make([]*Record, 0, n)
		for i := 0; i < n; i++ {
			child, err := rp.child.generateOne(childCtx)
			if err != nil {
				return nil, err
			}
			children = append(children, child)
		}
		rec.Children[rp.name] = children
		rec.RelationOrder = append(rec.RelationOrder, rp.name)
	}
	return rec, nil
}

// resolveCount implements DSL_SPEC.md ¤3.2: {fixed: N} or {min, max}.
func resolveCount(spec dsl.CountSpec, rng *rand.Rand) int {
	if spec.Fixed != nil {
		return *spec.Fixed
	}
	min, max := *spec.Min, *spec.Max
	return min + rng.Intn(max-min+1)
}
