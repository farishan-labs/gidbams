// Package sqlite builds a derived SQLite database from gidbams collections.
package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/farishan-labs/gidbams/internal/config"
	"github.com/farishan-labs/gidbams/internal/store"
)

// Builder generates a SQLite db file at dbPath from repo collections.
type Builder struct {
	RepoRoot string
	DbPath   string
}

func New(repoRoot, dbPath string) *Builder {
	return &Builder{RepoRoot: repoRoot, DbPath: dbPath}
}

func sqliteType(t string) string {
	switch strings.ToLower(t) {
	case "int", "integer":
		return "INTEGER"
	case "float", "number":
		return "REAL"
	case "bool", "boolean":
		return "INTEGER"
	default:
		return "TEXT"
	}
}

// Build creates (or overwrites) the db file.
func (b *Builder) Build(collections []struct {
	Def   *config.CollectionDefinition
	Store *store.Store
}) error {
	if err := os.MkdirAll(filepath.Dir(b.DbPath), 0o755); err != nil {
		return err
	}
	// remove existing
	_ = os.Remove(b.DbPath)
	db, err := sql.Open("sqlite", b.DbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	// metadata
	if _, err := db.Exec(`CREATE TABLE _meta (key TEXT PRIMARY KEY, value TEXT)`); err != nil {
		return err
	}
	_, _ = db.Exec(`INSERT INTO _meta (key,value) VALUES ('builder','gidbams')`)

	for _, c := range collections {
		def := c.Def
		if def == nil {
			continue
		}
		table := sanitize(def.Name)
		cols := []string{"_id TEXT PRIMARY KEY"}
		for _, col := range def.Columns {
			cols = append(cols, fmt.Sprintf("%s %s", sanitize(col.Name), sqliteType(col.Type)))
		}
		cols = append(cols, "_body TEXT", "_path TEXT", "_json TEXT")
		create := fmt.Sprintf("CREATE TABLE %s (%s)", table, strings.Join(cols, ", "))
		if _, err := db.Exec(create); err != nil {
			return fmt.Errorf("create table %s: %w", table, err)
		}

		recs, err := c.Store.LoadAll()
		if err != nil {
			return err
		}
		for _, r := range recs {
			// insert
			colNames := []string{"_id"}
			placeholders := []string{"?"}
			args := []any{r.ID}
			for _, col := range def.Columns {
				colNames = append(colNames, sanitize(col.Name))
				placeholders = append(placeholders, "?")
				v := r.Fields[col.Name]
				// convert bool to int
				if bb, ok := v.(bool); ok {
					if bb {
						args = append(args, 1)
					} else {
						args = append(args, 0)
					}
					continue
				}
				if arr, ok := v.([]any); ok {
					bb, _ := json.Marshal(arr)
					args = append(args, string(bb))
					continue
				}
				if m, ok := v.(map[string]any); ok {
					bb, _ := json.Marshal(m)
					args = append(args, string(bb))
					continue
				}
				args = append(args, v)
			}
			colNames = append(colNames, "_body", "_path", "_json")
			placeholders = append(placeholders, "?, ?, ?")
			jb, _ := json.Marshal(r.Fields)
			args = append(args, r.Body, r.Path, string(jb))

			q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, strings.Join(colNames, ","), strings.Join(placeholders, ","))
			if _, err := db.Exec(q, args...); err != nil {
				return fmt.Errorf("insert %s/%s: %w", table, r.ID, err)
			}
		}
	}
	return nil
}

func sanitize(s string) string {
	// simple: replace non-alnum with underscore
	out := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return '_'
	}, s)
	if out == "" {
		out = "_col"
	}
	// ensure not starting with digit
	if out[0] >= '0' && out[0] <= '9' {
		out = "_" + out
	}
	return out
}

// Open opens existing db read-only-ish.
func Open(dbPath string) (*sql.DB, error) {
	return sql.Open("sqlite", dbPath)
}
