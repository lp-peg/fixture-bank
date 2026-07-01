package fixturestore_test

import (
	"testing"

	"github.com/lp-peg/fixture-bank/internal/ferr"
	"github.com/lp-peg/fixture-bank/internal/fixturestore"
)

const sampleDSL = `
entity: user
count: 1
fields:
  id: {type: uuid, generator: uuid_v4}
  level: {type: int, generator: fixed, value: 50}
`

func TestStore_SaveLoadList(t *testing.T) {
	store := fixturestore.New(t.TempDir())
	tag := "user:level50:has_premium_pass"

	if err := store.Save(tag, []byte(sampleDSL)); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	fx, err := store.Load(tag)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if fx.Entity != "user" {
		t.Errorf("Entity = %q, want user", fx.Entity)
	}

	tags, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tags) != 1 || tags[0] != tag {
		t.Errorf("List() = %v, want [%q]", tags, tag)
	}
}

func TestStore_LoadMissing(t *testing.T) {
	store := fixturestore.New(t.TempDir())
	_, err := store.Load("does:not:exist")
	fe, ok := err.(*ferr.Error)
	if !ok || fe.ErrorType != ferr.TypeFixtureNotFound {
		t.Fatalf("error = %v, want ferr.TypeFixtureNotFound", err)
	}
}

func TestStore_ListOnMissingDir(t *testing.T) {
	store := fixturestore.New(t.TempDir() + "/does-not-exist-yet")
	tags, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(tags) != 0 {
		t.Errorf("List() = %v, want empty", tags)
	}
}

func TestStore_SaveRejectsEmptyTag(t *testing.T) {
	store := fixturestore.New(t.TempDir())
	err := store.Save("", []byte(sampleDSL))
	fe, ok := err.(*ferr.Error)
	if !ok || fe.ErrorType != ferr.TypeSyntaxError {
		t.Fatalf("error = %v, want ferr.TypeSyntaxError", err)
	}
}
