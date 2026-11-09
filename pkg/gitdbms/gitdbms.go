// Package gitdbms provides an embeddable API for git-as-SSOT databases.
package gitdbms

import (
	"github.com/farishan-labs/git-dbms/internal/config"
	"github.com/farishan-labs/git-dbms/internal/store"
	"github.com/farishan-labs/git-dbms/internal/validate"
)

// Re-export types
type RootConfig = config.RootConfig
type CollectionDefinition = config.CollectionDefinition
type CollectionRef = config.CollectionRef
type Store = store.Store

func LoadRoot(repoRoot string) (*RootConfig, error) {
	return config.LoadRoot(repoRoot)
}

func LoadCollection(repoRoot, dataDir string, ref CollectionRef) (*CollectionDefinition, error) {
	return config.LoadCollectionDefinition(repoRoot, dataDir, ref)
}

func NewStore(repoRoot string, def *CollectionDefinition) *Store {
	return store.New(repoRoot, def)
}

// ValidateResult is an alias for validate.Result.
type ValidateResult = validate.Result

// ValidateRepo validates all records in all collections of a repo.
func ValidateRepo(repoRoot string) ([]ValidateResult, error) {
	root, err := config.LoadRoot(repoRoot)
	if err != nil {
		return nil, err
	}
	var all []ValidateResult
	for _, ref := range root.Collections {
		def, err := config.LoadCollectionDefinition(repoRoot, root.DataDir, ref)
		if err != nil {
			return nil, err
		}
		st := store.New(repoRoot, def)
		recs, err := st.LoadAll()
		if err != nil {
			return nil, err
		}
		fields := make([]*map[string]any, len(recs))
		ids := make([]string, len(recs))
		paths := make([]string, len(recs))
		for i, r := range recs {
			f := r.Fields
			fields[i] = &f
			ids[i] = r.ID
			paths[i] = r.Path
		}
		all = append(all, validate.ValidateCollection(def, fields, ids, paths)...)
	}
	return all, nil
}
