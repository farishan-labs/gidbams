package validate

import (
	"testing"

	"github.com/farishan-labs/git-dbms/internal/config"
)

func TestValidateRecordRequired(t *testing.T) {
	def := &config.CollectionDefinition{
		Columns: []config.Column{
			{Name: "id", Type: "string", Required: true},
			{Name: "status", Type: "string", Enum: []string{"active", "inactive"}},
		},
	}
	fields := map[string]any{"id": "x", "status": "nope"}
	res := ValidateRecord(def, &fields, "x", "p.md")
	if res.Valid {
		t.Fatal("expected invalid")
	}
	if len(res.Errors) == 0 {
		t.Fatal("expected errors")
	}
	fields2 := map[string]any{"id": "x", "status": "active"}
	res2 := ValidateRecord(def, &fields2, "x", "p.md")
	if !res2.Valid {
		t.Fatalf("expected valid, got %v", res2.Errors)
	}
}

func TestValidateRecordMissingRequired(t *testing.T) {
	def := &config.CollectionDefinition{
		Columns: []config.Column{
			{Name: "id", Type: "string", Required: true},
		},
	}
	fields := map[string]any{}
	res := ValidateRecord(def, &fields, "x", "p.md")
	if res.Valid {
		t.Fatal("expected invalid")
	}
}

