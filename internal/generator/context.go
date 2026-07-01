// Package generator implements the DSL_SPEC.md ¤2.1 field generators
// (fixed, random_int, sample, ref, pool_ref, ...) plus the ¤2.3 `unique`
// wrapper.
package generator

import "math/rand"

// DB is the minimal database access generators need: pool_ref queries
// (¤2.2) and unique:db existence checks (¤2.3). internal/pgdb implements
// this against a real PostgreSQL connection; tests use an in-memory fake.
type DB interface {
	// Query runs a read-only query and returns the first column of each row.
	Query(query string) ([]any, error)
	// Exists reports whether entity's column already contains value.
	Exists(entity, column string, value any) (bool, error)
}

// Context carries per-record generation state: the RNG, the immediate
// parent record (for `ref: parent.<field>`), and an optional DB connection.
//
// Generators are instantiated once per field and reused across every
// record in a materialize run, so counters/caches/unique-claims live as
// generator instance state rather than in Context.
type Context struct {
	Rand   *rand.Rand
	Parent map[string]any
	DB     DB
}

// WithParent returns a copy of ctx scoped to a child record's immediate
// parent, per DSL_SPEC.md ¤3.1: "parent は常に直上の親エンティティを指す".
func (c *Context) WithParent(parent map[string]any) *Context {
	child := *c
	child.Parent = parent
	return &child
}
