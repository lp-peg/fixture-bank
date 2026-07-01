package dsl

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/lp-peg/fixture-bank/internal/ferr"
)

// Parse decodes and validates a Fixture Bank DSL document.
//
// It performs "syntax validation" only (DSL_SPEC.md ¤5 step 1): YAML
// well-formedness plus type/generator/parameter consistency. Schema
// integrity (step 2) and sandbox DB execution (step 3) happen elsewhere,
// against a live database connection.
func Parse(data []byte) (*Fixture, error) {
	var fx Fixture
	if err := yaml.Unmarshal(data, &fx); err != nil {
		return nil, ferr.New(ferr.TypeSyntaxError, "invalid YAML: %v", err)
	}
	normalize(&fx)
	if err := Validate(&fx); err != nil {
		return nil, err
	}
	return &fx, nil
}

// normalize fills in DSL-level defaults (DSL_SPEC.md ¤1: count defaults to
// 1, unique defaults to "none") recursively through relations.
func normalize(fx *Fixture) {
	if fx.Count == 0 {
		fx.Count = 1
	}
	normalizeFields(fx.Fields)
	normalizeRelations(fx.Relations)
}

func normalizeFields(fields Fields) {
	for i := range fields {
		if fields[i].Field.Unique == "" {
			fields[i].Field.Unique = UniqueNone
		}
	}
}

func normalizeRelations(relations Relations) {
	for i := range relations {
		normalizeFields(relations[i].Relation.Fields)
		normalizeRelations(relations[i].Relation.Relations)
	}
}

var validTypes = map[FieldType]bool{
	TypeUUID: true, TypeInt: true, TypeFloat: true,
	TypeString: true, TypeBoolean: true, TypeTimestamp: true,
}

var validGenerators = map[Generator]bool{
	GenFixed: true, GenRandomInt: true, GenRandomFloat: true, GenSample: true,
	GenSequence: true, GenFaker: true, GenUUIDV4: true, GenRef: true, GenPoolRef: true,
}

var validUnique = map[UniqueMode]bool{
	UniqueNone: true, UniqueBatch: true, UniqueDB: true,
}

// Validate performs DSL_SPEC.md ¤5 step 1 syntax validation: every
// type/generator/unique value is one of the defined enums, and each
// generator has the parameters it requires.
func Validate(fx *Fixture) error {
	if fx.Entity == "" {
		return ferr.New(ferr.TypeSyntaxError, "entity is required")
	}
	if fx.Count < 0 {
		return ferr.New(ferr.TypeSyntaxError, "count must be >= 0, got %d", fx.Count)
	}
	if len(fx.Fields) == 0 {
		return ferr.New(ferr.TypeSyntaxError, "%s: fields is required", fx.Entity)
	}
	if err := validateFields(fx.Entity, fx.Fields); err != nil {
		return err
	}
	return validateRelations(fx.Entity, fx.Relations)
}

func validateFields(entityPath string, fields Fields) error {
	seen := map[string]bool{}
	for _, entry := range fields {
		path := fmt.Sprintf("%s.fields.%s", entityPath, entry.Name)
		if seen[entry.Name] {
			return ferr.New(ferr.TypeSyntaxError, "%s: duplicate field name", path)
		}
		seen[entry.Name] = true
		if err := validateField(path, entry.Field); err != nil {
			return err
		}
	}
	return nil
}

func validateField(path string, f Field) error {
	if !validTypes[f.Type] {
		return ferr.New(ferr.TypeSyntaxError, "%s: unknown type %q", path, f.Type)
	}
	if !validGenerators[f.Generator] {
		return ferr.New(ferr.TypeSyntaxError, "%s: unknown generator %q", path, f.Generator)
	}
	if !validUnique[f.Unique] {
		return ferr.New(ferr.TypeSyntaxError, "%s: unknown unique mode %q", path, f.Unique)
	}

	switch f.Generator {
	case GenFixed:
		if f.Value == nil {
			return ferr.New(ferr.TypeSyntaxError, "%s: generator fixed requires value", path)
		}
	case GenRandomInt, GenRandomFloat:
		if f.Min == nil || f.Max == nil {
			return ferr.New(ferr.TypeSyntaxError, "%s: generator %s requires min and max", path, f.Generator)
		}
		if *f.Min > *f.Max {
			return ferr.New(ferr.TypeSyntaxError, "%s: min must be <= max", path)
		}
	case GenSample:
		if len(f.From) == 0 {
			return ferr.New(ferr.TypeSyntaxError, "%s: generator sample requires from", path)
		}
	case GenSequence:
		// start/step are optional (default 1/1), nothing required.
	case GenFaker:
		if f.Provider == "" {
			return ferr.New(ferr.TypeSyntaxError, "%s: generator faker requires provider", path)
		}
	case GenUUIDV4:
		// no parameters
	case GenRef:
		if f.Ref == "" {
			return ferr.New(ferr.TypeSyntaxError, "%s: generator ref requires ref", path)
		}
	case GenPoolRef:
		if f.Query == "" {
			return ferr.New(ferr.TypeSyntaxError, "%s: generator pool_ref requires query", path)
		}
	}
	return nil
}

func validateRelations(entityPath string, relations Relations) error {
	seen := map[string]bool{}
	for _, entry := range relations {
		path := fmt.Sprintf("%s.relations.%s", entityPath, entry.Name)
		if seen[entry.Name] {
			return ferr.New(ferr.TypeSyntaxError, "%s: duplicate relation name", path)
		}
		seen[entry.Name] = true

		rel := entry.Relation
		if rel.Entity == "" {
			return ferr.New(ferr.TypeSyntaxError, "%s: entity is required", path)
		}
		if len(rel.Fields) == 0 {
			return ferr.New(ferr.TypeSyntaxError, "%s: fields is required", path)
		}
		if rel.CountPerParent.IsZero() {
			return ferr.New(ferr.TypeSyntaxError, "%s: count_per_parent is required", path)
		}
		if rel.CountPerParent.Fixed == nil && (rel.CountPerParent.Min == nil || rel.CountPerParent.Max == nil) {
			return ferr.New(ferr.TypeSyntaxError, "%s: count_per_parent must set either fixed, or both min and max", path)
		}
		if rel.CountPerParent.Min != nil && rel.CountPerParent.Max != nil && *rel.CountPerParent.Min > *rel.CountPerParent.Max {
			return ferr.New(ferr.TypeSyntaxError, "%s: count_per_parent.min must be <= max", path)
		}
		if err := validateFields(path, rel.Fields); err != nil {
			return err
		}
		if err := validateRelations(path, rel.Relations); err != nil {
			return err
		}
	}
	return nil
}
