// Package gitdbms provides an embeddable API for git-as-SSOT databases.
package gitdbms

import (
	"github.com/farishan-labs/git-dbms/internal/config"
	"github.com/farishan-labs/git-dbms/internal/store"
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

