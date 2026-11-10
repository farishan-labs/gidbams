// Package config loads gidbams repository configuration and collection definitions.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// RootConfig describes a gidbams repository.
type RootConfig struct {
	Version     int             `yaml:"version"`
	DataDir     string          `yaml:"data_dir"`
	Collections []CollectionRef `yaml:"collections"`
	Settings    Settings        `yaml:"settings"`
}

type Settings struct {
	DefaultBranch string `yaml:"default_branch"`
	ValidateOnCI  bool   `yaml:"validate_on_ci"`
}

type CollectionRef struct {
	Name   string `yaml:"name"`
	Path   string `yaml:"path"`   // relative to data_dir
	Schema string `yaml:"schema"` // relative path to definition file (optional)
}

// CollectionDefinition describes a single collection's schema and storage.
type CollectionDefinition struct {
	Name        string       `yaml:"name"`
	RecordFile  RecordFile   `yaml:"record_file"`
	Columns     []Column     `yaml:"columns"`
	ForeignKeys []ForeignKey `yaml:"foreign_keys"`
	DefaultView string       `yaml:"default_view"`
	Titles      []string     `yaml:"titles"`
	DataDir     string       `yaml:"data_dir"`
}

type RecordFile struct {
	Format string `yaml:"format"` // md|json|yaml|yml|csv
	Name   string `yaml:"name"`   // template, e.g. "{id}.md"
}

type Column struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"` // string,int,float,bool,array,object,date
	Required bool     `yaml:"required"`
	Unique   bool     `yaml:"unique"`
	Min      *float64 `yaml:"min"`
	Max      *float64 `yaml:"max"`
	Pattern  string   `yaml:"pattern"`
	Enum     []string `yaml:"enum"`
	Default  any      `yaml:"default"`
}

type ForeignKey struct {
	Column     string `yaml:"column"`
	References string `yaml:"references"` // collection.column
}

const (
	DefaultConfigFile = "gitdb.yaml"
	DefaultDataDir    = "data"
)

// LoadRoot reads the root config from repo root. If no config file exists,
// a default config covering data/ subdirectories is synthesized.
func LoadRoot(repoRoot string) (*RootConfig, error) {
	cfgPath := filepath.Join(repoRoot, DefaultConfigFile)
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return synthesizeRoot(repoRoot)
		}
		return nil, err
	}
	var c RootConfig
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse %s: %w", cfgPath, err)
	}
	if c.DataDir == "" {
		c.DataDir = DefaultDataDir
	}
	if c.Version == 0 {
		c.Version = 1
	}
	for i := range c.Collections {
		if err := validateRef(c.Collections[i]); err != nil {
			return nil, err
		}
	}
	return &c, nil
}

func synthesizeRoot(repoRoot string) (*RootConfig, error) {
	dataDir := filepath.Join(repoRoot, DefaultDataDir)
	c := &RootConfig{Version: 1, DataDir: DefaultDataDir, Settings: Settings{DefaultBranch: "main", ValidateOnCI: true}}
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		c.Collections = append(c.Collections, CollectionRef{Name: e.Name(), Path: e.Name()})
	}
	return c, nil
}

// LoadCollectionDefinition loads a collection's definition. It looks for
// data/<path>/.collection/definition.yaml, then data/<path>/schema.yaml,
// then <repoRoot>/schemas/<name>.yaml.
func LoadCollectionDefinition(repoRoot string, dataDir string, ref CollectionRef) (*CollectionDefinition, error) {
	candidates := []string{
		filepath.Join(repoRoot, dataDir, ref.Path, ".collection", "definition.yaml"),
		filepath.Join(repoRoot, dataDir, ref.Path, ".collection", "definition.yml"),
		filepath.Join(repoRoot, dataDir, ref.Path, "schema.yaml"),
		filepath.Join(repoRoot, dataDir, ref.Path, "schema.yml"),
	}
	if ref.Schema != "" {
		candidates = append([]string{filepath.Join(repoRoot, ref.Schema)}, candidates...)
	}
	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var d CollectionDefinition
		if err := yaml.Unmarshal(b, &d); err != nil {
			return nil, fmt.Errorf("parse %s: %w", p, err)
		}
		if d.Name == "" {
			d.Name = ref.Name
		}
		if d.DataDir == "" {
			d.DataDir = filepath.Join(dataDir, ref.Path)
		}
		if d.RecordFile.Format == "" {
			d.RecordFile.Format = "md"
		}
		if d.RecordFile.Name == "" {
			d.RecordFile.Name = "{id}.md"
		}
		return &d, nil
	}
	// No schema found: synthesize from directory with md defaults.
	d := &CollectionDefinition{
		Name:       ref.Name,
		DataDir:    filepath.Join(dataDir, ref.Path),
		RecordFile: RecordFile{Format: "md", Name: "{id}.md"},
	}
	return d, nil
}

// SaveRoot writes root config.
func SaveRoot(repoRoot string, c *RootConfig) error {
	b, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(repoRoot, DefaultConfigFile), b, 0o644)
}

func validateRef(ref CollectionRef) error {
	if ref.Name == "" {
		return fmt.Errorf("collection name required")
	}
	if filepath.IsAbs(ref.Path) {
		return fmt.Errorf("collection path must be relative: %s", ref.Path)
	}
	if strings.Contains(ref.Path, "..") {
		return fmt.Errorf("collection path must not contain '..': %s", ref.Path)
	}
	if ref.Schema != "" {
		if filepath.IsAbs(ref.Schema) {
			return fmt.Errorf("schema path must be relative: %s", ref.Schema)
		}
		if strings.Contains(ref.Schema, "..") {
			return fmt.Errorf("schema path must not contain '..': %s", ref.Schema)
		}
	}
	return nil
}
