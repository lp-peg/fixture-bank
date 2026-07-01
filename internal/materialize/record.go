// Package materialize turns a parsed DSL fixture into generated records,
// per DESIGN.md's "materialize: DSLをもとにデータを量産" step.
package materialize

// Record is one generated row, plus its generated child relations keyed
// by relation name (DSL_SPEC.md ¤3).
//
// FieldOrder and RelationOrder preserve DSL declaration order so output
// renderers (JSON key order, SQL column order) are deterministic and
// readable rather than depending on Go's unordered map iteration.
type Record struct {
	Entity        string
	FieldOrder    []string
	Values        map[string]any
	RelationOrder []string
	Children      map[string][]*Record
}
