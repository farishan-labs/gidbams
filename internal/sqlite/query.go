// Package sqlite query helpers.
package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

// Query runs a read-only SQL query against dbPath and returns rows as JSON.
func Query(dbPath, q string) (string, error) {
	qt := strings.TrimSpace(strings.ToLower(q))
	if !strings.HasPrefix(qt, "select") && !strings.HasPrefix(qt, "pragma") && !strings.HasPrefix(qt, "with") {
		return "", fmt.Errorf("only SELECT/PRAGMA/WITH queries are allowed")
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return "", err
	}
	defer db.Close()
	rows, err := db.Query(q)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	cols, err := rows.Columns()
	if err != nil {
		return "", err
	}
	var out []map[string]any
	for rows.Next() {
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return "", err
		}
		row := map[string]any{}
		for i, c := range cols {
			// convert []byte to string
			if b, ok := vals[i].([]byte); ok {
				row[c] = string(b)
			} else {
				row[c] = vals[i]
			}
		}
		out = append(out, row)
	}
	b, err := json.Marshal(out)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

