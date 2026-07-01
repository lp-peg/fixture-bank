package generator

import (
	"github.com/lp-peg/fixture-bank/internal/dsl"
	"github.com/lp-peg/fixture-bank/internal/ferr"
)

// Generator produces one field value per call to Generate.
type Generator interface {
	Generate(ctx *Context) (any, error)
}

// New builds the generator for a field, wrapping it in unique-retry logic
// per DSL_SPEC.md ¤2.3 when the field declares unique != none.
//
// entity and fieldName scope unique:db checks (which table/column to
// query) and have no effect when unique is "none".
func New(entity, fieldName string, field dsl.Field) (Generator, error) {
	base, err := newBase(field)
	if err != nil {
		return nil, err
	}
	if field.Unique == dsl.UniqueNone {
		return base, nil
	}
	return &uniqueWrapper{inner: base, entity: entity, column: fieldName, mode: field.Unique}, nil
}

func newBase(field dsl.Field) (Generator, error) {
	switch field.Generator {
	case dsl.GenFixed:
		return &fixedGenerator{value: field.Value}, nil
	case dsl.GenRandomInt:
		return &randomIntGenerator{min: int64(*field.Min), max: int64(*field.Max)}, nil
	case dsl.GenRandomFloat:
		precision := 2
		if field.Precision != nil {
			precision = *field.Precision
		}
		return &randomFloatGenerator{min: *field.Min, max: *field.Max, precision: precision}, nil
	case dsl.GenSample:
		return &sampleGenerator{from: field.From}, nil
	case dsl.GenSequence:
		start, step := int64(1), int64(1)
		if field.Start != nil {
			start = *field.Start
		}
		if field.Step != nil {
			step = *field.Step
		}
		return &sequenceGenerator{start: start, step: step}, nil
	case dsl.GenFaker:
		return &fakerGenerator{provider: field.Provider}, nil
	case dsl.GenUUIDV4:
		return &uuidGenerator{}, nil
	case dsl.GenRef:
		return &refGenerator{ref: field.Ref}, nil
	case dsl.GenPoolRef:
		return &poolRefGenerator{query: field.Query}, nil
	default:
		return nil, ferr.New(ferr.TypeSyntaxError, "unknown generator %q", field.Generator)
	}
}
