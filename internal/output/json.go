// Package output renders generated records as `materialize --format
// json|sql` output (DSL_SPEC.md / DESIGN.md ¤3.3).
package output

import (
	"bytes"
	"encoding/json"

	"github.com/lp-peg/fixture-bank/internal/materialize"
)

// RenderJSON renders records as an indented JSON array, nesting each
// relation as a child array under its relation name. Key order follows
// DSL field/relation declaration order rather than Go's map order.
func RenderJSON(records []*materialize.Record) ([]byte, error) {
	arr := make([]*orderedMap, len(records))
	for i, r := range records {
		arr[i] = toOrdered(r)
	}
	return json.MarshalIndent(arr, "", "  ")
}

// orderedMap is a JSON object that marshals keys in insertion order instead
// of the alphabetical order encoding/json applies to map[string]any.
type orderedMap struct {
	keys []string
	vals map[string]any
}

func (m *orderedMap) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, k := range m.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		kb, err := json.Marshal(k)
		if err != nil {
			return nil, err
		}
		buf.Write(kb)
		buf.WriteByte(':')
		vb, err := json.Marshal(m.vals[k])
		if err != nil {
			return nil, err
		}
		buf.Write(vb)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

func toOrdered(r *materialize.Record) *orderedMap {
	m := &orderedMap{
		keys: append([]string{}, r.FieldOrder...),
		vals: make(map[string]any, len(r.Values)+len(r.Children)),
	}
	for k, v := range r.Values {
		m.vals[k] = v
	}
	for _, relName := range r.RelationOrder {
		children := r.Children[relName]
		childArr := make([]*orderedMap, len(children))
		for i, c := range children {
			childArr[i] = toOrdered(c)
		}
		m.keys = append(m.keys, relName)
		m.vals[relName] = childArr
	}
	return m
}
