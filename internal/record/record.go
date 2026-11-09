// Package record loads and writes individual records in multiple formats.
package record

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Record is a parsed record: frontmatter/metadata as map plus optional body.
type Record struct {
	ID     string         `json:"id"`
	Path   string         `json:"-"`
	Format string         `json:"-"`
	Fields map[string]any `json:"fields"`
	Body   string         `json:"body,omitempty"`
}

// Load reads a record file and parses it according to extension.
func Load(path string) (*Record, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	ext := strings.ToLower(filepath.Ext(path))
	r := &Record{Path: path, Fields: map[string]any{}}
	switch ext {
	case ".md", ".mdx":
		r.Format = "md"
		fields, body, err := parseFrontmatter(b)
		if err != nil {
			return nil, err
		}
		r.Fields = fields
		r.Body = body
	case ".json":
		r.Format = "json"
		if err := json.Unmarshal(b, &r.Fields); err != nil {
			return nil, fmt.Errorf("parse json %s: %w", path, err)
		}
	case ".yaml", ".yml":
		r.Format = "yaml"
		if err := yaml.Unmarshal(b, &r.Fields); err != nil {
			return nil, fmt.Errorf("parse yaml %s: %w", path, err)
		}
	default:
		return nil, fmt.Errorf("unsupported record format: %s", ext)
	}
	if id, ok := r.Fields["id"].(string); ok {
		r.ID = id
	} else {
		r.ID = strings.TrimSuffix(filepath.Base(path), ext)
	}
	return r, nil
}

func parseFrontmatter(b []byte) (map[string]any, string, error) {
	s := string(b)
	if !strings.HasPrefix(s, "---\n") && !strings.HasPrefix(s, "---\r\n") {
		return map[string]any{}, s, nil
	}
	// find closing ---
	rest := s[4:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return nil, "", fmt.Errorf("unterminated frontmatter")
	}
	fm := rest[:idx]
	body := rest[idx+4:]
	body = strings.TrimPrefix(body, "\n")
	fields := map[string]any{}
	if err := yaml.Unmarshal([]byte(fm), &fields); err != nil {
		return nil, "", fmt.Errorf("parse frontmatter: %w", err)
	}
	if fields == nil {
		fields = map[string]any{}
	}
	return fields, body, nil
}

// Save writes a record back to disk in its format.
func (r *Record) Save() error {
	if r.Path == "" {
		return fmt.Errorf("record has no path")
	}
	switch r.Format {
	case "md":
		var buf bytes.Buffer
		buf.WriteString("---\n")
		b, err := yaml.Marshal(r.Fields)
		if err != nil {
			return err
		}
		buf.Write(b)
		buf.WriteString("---\n")
		if r.Body != "" {
			buf.WriteString(r.Body)
			if !strings.HasSuffix(r.Body, "\n") {
				buf.WriteString("\n")
			}
		}
		return os.WriteFile(r.Path, buf.Bytes(), 0o644)
	case "json":
		b, err := json.MarshalIndent(r.Fields, "", "  ")
		if err != nil {
			return err
		}
		b = append(b, '\n')
		return os.WriteFile(r.Path, b, 0o644)
	case "yaml", "yml":
		b, err := yaml.Marshal(r.Fields)
		if err != nil {
			return err
		}
		return os.WriteFile(r.Path, b, 0o644)
	default:
		return fmt.Errorf("unsupported format: %s", r.Format)
	}
}

// FieldString returns a field as string.
func FieldString(fields map[string]any, key string) string {
	v, ok := fields[key]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		b, _ := json.Marshal(t)
		return string(b)
	}
}

