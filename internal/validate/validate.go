// Package validate validates records against collection schemas.
package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/farishan-labs/git-dbms/internal/config"
)

// Result holds validation outcome for one record.
type Result struct {
	Path   string   `json:"path"`
	ID     string   `json:"id"`
	Valid  bool     `json:"valid"`
	Errors []string `json:"errors,omitempty"`
}

// ValidateRecord validates a record's fields against a collection definition.
func ValidateRecord(def *config.CollectionDefinition, r *map[string]any, id string, path string) Result {
	res := Result{Path: path, ID: id, Valid: true}
	fields := *r
	// required + type checks
	seen := map[string]bool{}
	for _, col := range def.Columns {
		seen[col.Name] = true
		v, ok := fields[col.Name]
		if !ok || v == nil {
			if col.Required {
				res.Errors = append(res.Errors, fmt.Sprintf("missing required field %q", col.Name))
				res.Valid = false
			}
			if col.Default != nil && (!ok || v == nil) {
				fields[col.Name] = col.Default
				v = col.Default
				ok = true
			}
			if !ok || v == nil {
				continue
			}
		}
		if err := checkType(col, v); err != nil {
			res.Errors = append(res.Errors, err.Error())
			res.Valid = false
			continue
		}
		// constraints
		if col.Pattern != "" {
			s, _ := v.(string)
			re, err := regexp.Compile(col.Pattern)
			if err == nil && !re.MatchString(s) {
				res.Errors = append(res.Errors, fmt.Sprintf("field %q does not match pattern %s", col.Name, col.Pattern))
				res.Valid = false
			}
		}
		if len(col.Enum) > 0 {
			s := fmt.Sprintf("%v", v)
			okEnum := false
			for _, e := range col.Enum {
				if s == e {
					okEnum = true
					break
				}
			}
			if !okEnum {
				res.Errors = append(res.Errors, fmt.Sprintf("field %q value %v not in enum", col.Name, v))
				res.Valid = false
			}
		}
		if col.Min != nil || col.Max != nil {
			f, ok := toFloat(v)
			if ok {
				if col.Min != nil && f < *col.Min {
					res.Errors = append(res.Errors, fmt.Sprintf("field %q below min %v", col.Name, *col.Min))
					res.Valid = false
				}
				if col.Max != nil && f > *col.Max {
					res.Errors = append(res.Errors, fmt.Sprintf("field %q above max %v", col.Name, *col.Max))
					res.Valid = false
				}
			}
		}
	}
	// unknown fields allowed (forward compat). unique checked at collection level.
	_ = seen
	return res
}

func checkType(col config.Column, v any) error {
	switch col.Type {
	case "", "any":
		return nil
	case "string":
		if _, ok := v.(string); !ok {
			return fmt.Errorf("field %q expected string, got %T", col.Name, v)
		}
	case "int", "integer":
		switch v.(type) {
		case int, int64, float64:
			// accept numeric
		default:
			return fmt.Errorf("field %q expected int, got %T", col.Name, v)
		}
	case "float", "number":
		if _, ok := toFloat(v); !ok {
			return fmt.Errorf("field %q expected number, got %T", col.Name, v)
		}
	case "bool", "boolean":
		if _, ok := v.(bool); !ok {
			return fmt.Errorf("field %q expected bool, got %T", col.Name, v)
		}
	case "array", "list":
		switch v.(type) {
		case []any, []string, []int, []float64:
		default:
			return fmt.Errorf("field %q expected array, got %T", col.Name, v)
		}
	case "object", "map":
		switch v.(type) {
		case map[string]any, map[any]any:
		default:
			return fmt.Errorf("field %q expected object, got %T", col.Name, v)
		}
	case "date":
		s, ok := v.(string)
		if !ok {
			return fmt.Errorf("field %q expected date string, got %T", col.Name, v)
		}
		if len(s) < 10 {
			return fmt.Errorf("field %q invalid date: %s", col.Name, s)
		}
	}
	return nil
}

func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case float32:
		return float64(t), true
	case float64:
		return t, true
	}
	return 0, false
}

// ValidateCollection validates all records in a collection dir against def,
// including uniqueness constraints.
func ValidateCollection(def *config.CollectionDefinition, records []*map[string]any, ids []string, paths []string) []Result {
	results := make([]Result, len(records))
	uniqueVals := map[string]map[any]bool{}
	for _, col := range def.Columns {
		if col.Unique {
			uniqueVals[col.Name] = map[any]bool{}
		}
	}
	for i := range records {
		r := ValidateRecord(def, records[i], ids[i], paths[i])
		// uniqueness
		for _, col := range def.Columns {
			if !col.Unique {
				continue
			}
			v := (*records[i])[col.Name]
			if v == nil {
				continue
			}
			if uniqueVals[col.Name][v] {
				r.Errors = append(r.Errors, fmt.Sprintf("duplicate value %v for unique field %q", v, col.Name))
				r.Valid = false
			}
			uniqueVals[col.Name][v] = true
		}
		results[i] = r
	}
	return results
}

// FormatErrors formats a result list into a single error string if any invalid.
func FormatErrors(results []Result) string {
	var b strings.Builder
	bad := 0
	for _, r := range results {
		if !r.Valid {
			bad++
			for _, e := range r.Errors {
				fmt.Fprintf(&b, "%s: %s\n", r.ID, e)
			}
		}
	}
	if bad == 0 {
		return ""
	}
	return fmt.Sprintf("%d invalid records:\n%s", bad, b.String())
}
