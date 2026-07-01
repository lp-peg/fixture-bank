package output_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/lp-peg/fixture-bank/internal/materialize"
	"github.com/lp-peg/fixture-bank/internal/output"
)

func sampleRecords() []*materialize.Record {
	item := &materialize.Record{
		Entity:     "order_item",
		FieldOrder: []string{"id", "quantity"},
		Values:     map[string]any{"id": "item-1", "quantity": int64(2)},
	}
	order := &materialize.Record{
		Entity:        "order",
		FieldOrder:    []string{"id", "status"},
		Values:        map[string]any{"id": "order-1", "status": "paid"},
		RelationOrder: []string{"items"},
		Children:      map[string][]*materialize.Record{"items": {item}},
	}
	user := &materialize.Record{
		Entity:        "user",
		FieldOrder:    []string{"id", "email", "is_admin", "note"},
		Values:        map[string]any{"id": "user-1", "email": "a@example.com", "is_admin": true, "note": nil},
		RelationOrder: []string{"orders"},
		Children:      map[string][]*materialize.Record{"orders": {order}},
	}
	return []*materialize.Record{user}
}

func TestRenderJSON_PreservesFieldOrderAndNestsRelations(t *testing.T) {
	out, err := output.RenderJSON(sampleRecords())
	if err != nil {
		t.Fatal(err)
	}

	idIdx := strings.Index(string(out), `"id"`)
	emailIdx := strings.Index(string(out), `"email"`)
	ordersIdx := strings.Index(string(out), `"orders"`)
	if !(idIdx < emailIdx && emailIdx < ordersIdx) {
		t.Fatalf("key order not preserved: id=%d email=%d orders=%d\n%s", idIdx, emailIdx, ordersIdx, out)
	}

	var decoded []map[string]any
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}
	orders := decoded[0]["orders"].([]any)
	if len(orders) != 1 {
		t.Fatalf("len(orders) = %d, want 1", len(orders))
	}
	items := orders[0].(map[string]any)["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
}

func TestRenderSQL_ParentBeforeChildAndEscaping(t *testing.T) {
	records := sampleRecords()
	records[0].Values["email"] = "o'brien@example.com"

	sql, err := output.RenderSQL(records)
	if err != nil {
		t.Fatal(err)
	}

	userIdx := strings.Index(sql, `INSERT INTO "user"`)
	orderIdx := strings.Index(sql, `INSERT INTO "order"`)
	itemIdx := strings.Index(sql, `INSERT INTO "order_item"`)
	if !(userIdx >= 0 && userIdx < orderIdx && orderIdx < itemIdx) {
		t.Fatalf("expected user before order before order_item, got:\n%s", sql)
	}
	if !strings.Contains(sql, `'o''brien@example.com'`) {
		t.Fatalf("single quote not escaped:\n%s", sql)
	}
	if !strings.Contains(sql, "TRUE") {
		t.Fatalf("boolean not rendered as TRUE:\n%s", sql)
	}
	if !strings.Contains(sql, "NULL") {
		t.Fatalf("nil not rendered as NULL:\n%s", sql)
	}
}

func TestRenderSQL_UnsupportedValueType(t *testing.T) {
	records := []*materialize.Record{{
		Entity:     "user",
		FieldOrder: []string{"weird"},
		Values:     map[string]any{"weird": []string{"not", "supported"}},
	}}
	if _, err := output.RenderSQL(records); err == nil {
		t.Fatal("expected error for unsupported value type, got nil")
	}
}
