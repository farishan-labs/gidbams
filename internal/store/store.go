// Package store manages a collection's records on disk.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/farishan-labs/git-dbms/internal/config"
	"github.com/farishan-labs/git-dbms/internal/record"
)

// Store is a collection store rooted at a directory.
type Store struct {
	Def      *config.CollectionDefinition
	RepoRoot string
	Dir      string // absolute path to collection data dir
}

func New(repoRoot string, def *config.CollectionDefinition) *Store {
	dir := def.DataDir
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(repoRoot, dir)
	}
	return &Store{Def: def, RepoRoot: repoRoot, Dir: dir}
}

// List returns sorted record file paths in the collection dir (recursive).
func (s *Store) List() ([]string, error) {
	var files []string
	err := filepath.Walk(s.Dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// skip hidden/.collection
			base := filepath.Base(p)
			if strings.HasPrefix(base, ".") && p != s.Dir {
				return filepath.SkipDir
			}
			return nil
		}
		base := filepath.Base(p)
		if base == "schema.yaml" || base == "schema.yml" || base == "definition.yaml" || base == "definition.yml" {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if ext == ".md" || ext == ".mdx" || ext == ".json" || ext == ".yaml" || ext == ".yml" {
			files = append(files, p)
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// LoadAll loads all records.
func (s *Store) LoadAll() ([]*record.Record, error) {
	files, err := s.List()
	if err != nil {
		return nil, err
	}
	out := make([]*record.Record, 0, len(files))
	for _, f := range files {
		r, err := record.Load(f)
		if err != nil {
			return nil, fmt.Errorf("load %s: %w", f, err)
		}
		out = append(out, r)
	}
	return out, nil
}

// Get loads a record by id (searches file name / frontmatter id).
func (s *Store) Get(id string) (*record.Record, error) {
	files, err := s.List()
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		r, err := record.Load(f)
		if err != nil {
			continue
		}
		if r.ID == id {
			return r, nil
		}
	}
	return nil, fmt.Errorf("record not found: %s", id)
}

// Create writes a new record. id must be provided in fields or as arg.
func (s *Store) Create(id string, fields map[string]any, body string) (*record.Record, error) {
	if id == "" {
		if v, ok := fields["id"].(string); ok {
			id = v
		}
	}
	if id == "" {
		return nil, fmt.Errorf("id required")
	}
	if err := validateID(id); err != nil {
		return nil, err
	}
	fields["id"] = id
	format := s.Def.RecordFile.Format
	if format == "" {
		format = "md"
	}
	ext := "." + format
	if format == "yaml" {
		ext = ".yaml"
	}
	if format == "md" {
		ext = ".md"
	}
	name := strings.ReplaceAll(s.Def.RecordFile.Name, "{id}", id)
	if name == "" || name == "{id}" {
		name = id + ext
	}
	if filepath.Ext(name) == "" {
		name += ext
	}
	// ensure nested dirs by category? flat for now
	path := filepath.Join(s.Dir, name)
	if !isWithin(s.Dir, path) {
		return nil, fmt.Errorf("invalid record path for id %q", id)
	}
	if _, err := os.Stat(path); err == nil {
		return nil, fmt.Errorf("record already exists: %s", id)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	r := &record.Record{ID: id, Path: path, Format: format, Fields: fields, Body: body}
	if err := r.Save(); err != nil {
		return nil, err
	}
	return r, nil
}

// Update overwrites an existing record's fields/body.
func (s *Store) Update(id string, fields map[string]any, body *string) (*record.Record, error) {
	r, err := s.Get(id)
	if err != nil {
		return nil, err
	}
	if r.Fields == nil {
		r.Fields = map[string]any{}
	}
	for k, v := range fields {
		r.Fields[k] = v
	}
	r.Fields["id"] = id
	if body != nil {
		r.Body = *body
	}
	if err := r.Save(); err != nil {
		return nil, err
	}
	return r, nil
}

// Delete removes a record file.
func (s *Store) Delete(id string) error {
	r, err := s.Get(id)
	if err != nil {
		return err
	}
	return os.Remove(r.Path)
}

func validateID(id string) error {
	if id == "" {
		return fmt.Errorf("id required")
	}
	if strings.ContainsAny(id, "/\\") {
		return fmt.Errorf("id must not contain path separators: %q", id)
	}
	if id == "." || id == ".." {
		return fmt.Errorf("invalid id: %q", id)
	}
	if strings.Contains(id, "..") {
		return fmt.Errorf("invalid id: %q", id)
	}
	return nil
}

func isWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return false
	}
	return true
}
