package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const cmdTestDSL = `
entity: user
count: 3
seed: 1
fields:
  id: {type: uuid, generator: uuid_v4}
  level: {type: int, generator: fixed, value: 50}
`

func TestMaterializeCmd_JSONAndSQL(t *testing.T) {
	dir := t.TempDir()
	dslPath := filepath.Join(dir, "fixture.yaml")
	if err := os.WriteFile(dslPath, []byte(cmdTestDSL), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := newMaterializeCmd()
	outPath := filepath.Join(dir, "out.json")
	cmd.SetArgs([]string{"--dsl", dslPath, "--format", "json", "--out", outPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"level": 50`) {
		t.Errorf("json output missing expected field:\n%s", data)
	}

	cmd = newMaterializeCmd()
	outPath = filepath.Join(dir, "out.sql")
	cmd.SetArgs([]string{"--dsl", dslPath, "--format", "sql", "--out", outPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	data, err = os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `INSERT INTO "user"`) {
		t.Errorf("sql output missing expected INSERT:\n%s", data)
	}
}

func TestMaterializeCmd_RequiresExactlyOneSource(t *testing.T) {
	cmd := newMaterializeCmd()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when neither --dsl nor --fixture is given")
	}

	cmd = newMaterializeCmd()
	cmd.SetArgs([]string{"--dsl", "a.yaml", "--fixture", "b:c"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected error when both --dsl and --fixture are given")
	}
}

func TestMaterializeCmd_CountOverride(t *testing.T) {
	dir := t.TempDir()
	dslPath := filepath.Join(dir, "fixture.yaml")
	if err := os.WriteFile(dslPath, []byte(cmdTestDSL), 0o644); err != nil {
		t.Fatal(err)
	}

	outPath := filepath.Join(dir, "out.json")
	cmd := newMaterializeCmd()
	cmd.SetArgs([]string{"--dsl", dslPath, "--count", "5", "--format", "json", "--out", outPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), `"level": 50`) != 5 {
		t.Errorf("expected 5 records with --count 5, got output:\n%s", data)
	}
}

func TestFixtureSaveListMaterialize(t *testing.T) {
	dir := t.TempDir()
	dslPath := filepath.Join(dir, "fixture.yaml")
	if err := os.WriteFile(dslPath, []byte(cmdTestDSL), 0o644); err != nil {
		t.Fatal(err)
	}
	storeDir := filepath.Join(dir, "fixtures")

	save := newFixtureSaveCmd()
	save.SetArgs([]string{"--dsl", dslPath, "--tag", "user:level50", "--store-dir", storeDir})
	if err := save.Execute(); err != nil {
		t.Fatalf("save Execute() error = %v", err)
	}

	list := newFixtureListCmd()
	list.SetArgs([]string{"--store-dir", storeDir})
	if err := list.Execute(); err != nil {
		t.Fatalf("list Execute() error = %v", err)
	}

	outPath := filepath.Join(dir, "out.json")
	mat := newMaterializeCmd()
	mat.SetArgs([]string{"--fixture", "user:level50", "--store-dir", storeDir, "--format", "json", "--out", outPath})
	if err := mat.Execute(); err != nil {
		t.Fatalf("materialize --fixture Execute() error = %v", err)
	}
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"level": 50`) {
		t.Errorf("materialize --fixture output missing expected field:\n%s", data)
	}
}
