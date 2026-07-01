// Package fixturestore implements local, file-based storage and
// tag-based lookup of saved DSL fixtures (DESIGN.md ¤5
// "Fixtureの保存・タグ管理"), e.g. `--fixture user:level50:has_premium_pass`.
package fixturestore

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lp-peg/fixture-bank/internal/dsl"
	"github.com/lp-peg/fixture-bank/internal/ferr"
)

// Store is a directory of saved fixtures, one YAML file per tag.
type Store struct {
	dir string
}

// New returns a Store rooted at dir. dir need not exist yet; it is
// created on the first Save.
func New(dir string) *Store {
	return &Store{dir: dir}
}

func (s *Store) path(tag string) string {
	return filepath.Join(s.dir, tag+".yaml")
}

// Save writes raw DSL YAML under tag (e.g. "user:level50:has_premium_pass").
func (s *Store) Save(tag string, dslYAML []byte) error {
	if tag == "" {
		return ferr.New(ferr.TypeSyntaxError, "fixture tag must not be empty")
	}
	path := s.path(tag)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("fixturestore: %w", err)
	}
	if err := os.WriteFile(path, dslYAML, 0o644); err != nil {
		return fmt.Errorf("fixturestore: %w", err)
	}
	return nil
}

// Load reads and parses (syntax-validates) the fixture saved under tag.
func (s *Store) Load(tag string) (*dsl.Fixture, error) {
	data, err := os.ReadFile(s.path(tag))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ferr.New(ferr.TypeFixtureNotFound, "no fixture saved under tag %q", tag)
		}
		return nil, fmt.Errorf("fixturestore: %w", err)
	}
	return dsl.Parse(data)
}

// List returns every saved fixture tag, sorted.
func (s *Store) List() ([]string, error) {
	if _, err := os.Stat(s.dir); os.IsNotExist(err) {
		return nil, nil
	}

	var tags []string
	err := filepath.WalkDir(s.dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		rel, err := filepath.Rel(s.dir, path)
		if err != nil {
			return err
		}
		tags = append(tags, strings.TrimSuffix(rel, ".yaml"))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(tags)
	return tags, nil
}
