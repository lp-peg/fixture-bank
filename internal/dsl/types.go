// Package dsl defines the Fixture Bank DSL data model and parses/validates
// YAML documents against it. See docs/DSL_SPEC.md for the authoritative
// specification this package implements.
package dsl

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// FieldType enumerates the value types a field can declare.
type FieldType string

const (
	TypeUUID      FieldType = "uuid"
	TypeInt       FieldType = "int"
	TypeFloat     FieldType = "float"
	TypeString    FieldType = "string"
	TypeBoolean   FieldType = "boolean"
	TypeTimestamp FieldType = "timestamp"
)

// Generator enumerates the supported field generators.
type Generator string

const (
	GenFixed       Generator = "fixed"
	GenRandomInt   Generator = "random_int"
	GenRandomFloat Generator = "random_float"
	GenSample      Generator = "sample"
	GenSequence    Generator = "sequence"
	GenFaker       Generator = "faker"
	GenUUIDV4      Generator = "uuid_v4"
	GenRef         Generator = "ref"
	GenPoolRef     Generator = "pool_ref"
)

// UniqueMode enumerates the supported uniqueness guarantees for a field.
type UniqueMode string

const (
	UniqueNone  UniqueMode = "none"
	UniqueBatch UniqueMode = "batch"
	UniqueDB    UniqueMode = "db"
)

// Field is a single field definition under `fields:`.
//
// Generator-specific parameters are kept as plain fields (rather than a
// nested map) so the struct mirrors DSL_SPEC.md ¤2.1 directly; each
// generator implementation reads only the parameters it needs.
type Field struct {
	Type      FieldType  `yaml:"type"`
	Generator Generator  `yaml:"generator"`
	Unique    UniqueMode `yaml:"unique"`

	Value     any      `yaml:"value,omitempty"`
	Min       *float64 `yaml:"min,omitempty"`
	Max       *float64 `yaml:"max,omitempty"`
	Precision *int     `yaml:"precision,omitempty"`
	From      []any    `yaml:"from,omitempty"`
	Start     *int64   `yaml:"start,omitempty"`
	Step      *int64   `yaml:"step,omitempty"`
	Provider  string   `yaml:"provider,omitempty"`
	Ref       string   `yaml:"ref,omitempty"`
	Query     string   `yaml:"query,omitempty"`
}

// FieldEntry is one (name, Field) pair, used to preserve declaration order.
type FieldEntry struct {
	Name  string
	Field Field
}

// Fields preserves the declaration order of a `fields:` mapping. Field order
// is part of the deterministic output contract (column order in SQL, key
// order in JSON), so a plain map[string]Field is not sufficient.
type Fields []FieldEntry

// UnmarshalYAML preserves mapping key order by walking the raw node instead
// of decoding directly into a Go map.
func (f *Fields) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("fields: expected a mapping, got %v", node.Kind)
	}
	entries := make(Fields, 0, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		var field Field
		if err := valNode.Decode(&field); err != nil {
			return fmt.Errorf("fields.%s: %w", keyNode.Value, err)
		}
		entries = append(entries, FieldEntry{Name: keyNode.Value, Field: field})
	}
	*f = entries
	return nil
}

// Get returns the field named name, if present.
func (f Fields) Get(name string) (Field, bool) {
	for _, e := range f {
		if e.Name == name {
			return e.Field, true
		}
	}
	return Field{}, false
}

// CountSpec describes `count_per_parent:` — either a fixed count or a
// min/max range (DSL_SPEC.md ¤3.2).
type CountSpec struct {
	Fixed *int
	Min   *int
	Max   *int
}

func (c *CountSpec) UnmarshalYAML(node *yaml.Node) error {
	var raw struct {
		Fixed *int `yaml:"fixed"`
		Min   *int `yaml:"min"`
		Max   *int `yaml:"max"`
	}
	if err := node.Decode(&raw); err != nil {
		return err
	}
	c.Fixed, c.Min, c.Max = raw.Fixed, raw.Min, raw.Max
	return nil
}

// IsZero reports whether the CountSpec was left unset in YAML.
func (c CountSpec) IsZero() bool {
	return c.Fixed == nil && c.Min == nil && c.Max == nil
}

// Relation is one child entity definition under `relations:`.
type Relation struct {
	Entity         string    `yaml:"entity"`
	CountPerParent CountSpec `yaml:"count_per_parent"`
	Fields         Fields    `yaml:"fields"`
	Relations      Relations `yaml:"relations,omitempty"`
}

// RelationEntry is one (name, Relation) pair, used to preserve declaration
// order (also the order relations are generated topologically per parent).
type RelationEntry struct {
	Name     string
	Relation Relation
}

// Relations preserves declaration order of a `relations:` mapping.
type Relations []RelationEntry

func (r *Relations) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.MappingNode {
		return fmt.Errorf("relations: expected a mapping, got %v", node.Kind)
	}
	entries := make(Relations, 0, len(node.Content)/2)
	for i := 0; i < len(node.Content); i += 2 {
		keyNode := node.Content[i]
		valNode := node.Content[i+1]
		var rel Relation
		if err := valNode.Decode(&rel); err != nil {
			return fmt.Errorf("relations.%s: %w", keyNode.Value, err)
		}
		entries = append(entries, RelationEntry{Name: keyNode.Value, Relation: rel})
	}
	*r = entries
	return nil
}

// Fixture is the top-level DSL document (DSL_SPEC.md ¤1).
type Fixture struct {
	Entity    string    `yaml:"entity"`
	Count     int       `yaml:"count"`
	Seed      *int64    `yaml:"seed"`
	Fields    Fields    `yaml:"fields"`
	Relations Relations `yaml:"relations,omitempty"`
}
