// Package httpserver provides a small HTTP API over gidbams.
package httpserver

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/farishan-labs/gidbams/internal/config"
	"github.com/farishan-labs/gidbams/internal/sqlite"
	"github.com/farishan-labs/gidbams/internal/store"
	"github.com/farishan-labs/gidbams/internal/validate"
)

type Server struct {
	RepoRoot string
	Token    string
}

func New(repoRoot, token string) *Server {
	return &Server{RepoRoot: repoRoot, Token: token}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/collections", s.auth(s.handleCollections))
	mux.HandleFunc("/records", s.auth(s.handleRecords))
	mux.HandleFunc("/record", s.auth(s.handleRecord))
	mux.HandleFunc("/query", s.auth(s.handleQuery))
	mux.HandleFunc("/stats", s.auth(s.handleStats))
	mux.HandleFunc("/search", s.auth(s.handleSearch))
	mux.HandleFunc("/validate", s.auth(s.handleValidate))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		mux.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func (s *Server) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.Token != "" {
			h := r.Header.Get("Authorization")
			if h != "Bearer "+s.Token {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, map[string]string{"status": "ok"})
}

func (s *Server) handleCollections(w http.ResponseWriter, r *http.Request) {
	root, err := config.LoadRoot(s.RepoRoot)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	names := []string{}
	for _, c := range root.Collections {
		names = append(names, c.Name)
	}
	writeJSON(w, names)
}

func (s *Server) handleRecords(w http.ResponseWriter, r *http.Request) {
	coll := r.URL.Query().Get("collection")
	root, err := config.LoadRoot(s.RepoRoot)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	def, err := findDef(root, s.RepoRoot, coll)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	st := store.New(s.RepoRoot, def)
	recs, err := st.LoadAll()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	limit := 0
	offset := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		fmt.Sscanf(v, "%d", &limit)
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		fmt.Sscanf(v, "%d", &offset)
	}
	if offset > len(recs) {
		offset = len(recs)
	}
	if limit <= 0 || offset+limit > len(recs) {
		limit = len(recs) - offset
	}
	out := []map[string]any{}
	for i := offset; i < offset+limit; i++ {
		out = append(out, recs[i].Fields)
	}
	writeJSON(w, out)
}

func (s *Server) handleRecord(w http.ResponseWriter, r *http.Request) {
	coll := r.URL.Query().Get("collection")
	id := r.URL.Query().Get("id")
	root, err := config.LoadRoot(s.RepoRoot)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	def, err := findDef(root, s.RepoRoot, coll)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	st := store.New(s.RepoRoot, def)
	rec, err := st.Get(id)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	writeJSON(w, rec.Fields)
}

func (s *Server) handleQuery(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("sql")
	dbPath := filepath.Join(s.RepoRoot, ".gitdb", "data.db")
	if v := config.EnvDB(); v != "" {
		dbPath = v
	}
	out, err := sqlite.Query(dbPath, q)
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(out))
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))
	root, err := config.LoadRoot(s.RepoRoot)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	out := []map[string]any{}
	for _, ref := range root.Collections {
		def, err := config.LoadCollectionDefinition(s.RepoRoot, root.DataDir, ref)
		if err != nil {
			continue
		}
		st := store.New(s.RepoRoot, def)
		recs, err := st.LoadAll()
		if err != nil {
			continue
		}
		for _, rec := range recs {
			jb, _ := json.Marshal(rec.Fields)
			if strings.Contains(strings.ToLower(rec.ID), q) || strings.Contains(strings.ToLower(string(jb)), q) || strings.Contains(strings.ToLower(rec.Body), q) {
				entry := map[string]any{"collection": ref.Name}
				for k, v := range rec.Fields {
					entry[k] = v
				}
				out = append(out, entry)
			}
		}
	}
	writeJSON(w, out)
}

func (s *Server) handleValidate(w http.ResponseWriter, r *http.Request) {
	coll := r.URL.Query().Get("collection")
	root, err := config.LoadRoot(s.RepoRoot)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if coll == "" {
		// validate all
		all := []map[string]any{}
		for _, ref := range root.Collections {
			def, err := config.LoadCollectionDefinition(s.RepoRoot, root.DataDir, ref)
			if err != nil {
				continue
			}
			st := store.New(s.RepoRoot, def)
			recs, _ := st.LoadAll()
			fields := make([]*map[string]any, len(recs))
			ids := make([]string, len(recs))
			paths := make([]string, len(recs))
			for i, rec := range recs {
				f := rec.Fields
				fields[i] = &f
				ids[i] = rec.ID
				paths[i] = rec.Path
			}
			res := validate.ValidateCollection(def, fields, ids, paths)
			for _, r := range res {
				all = append(all, map[string]any{"id": r.ID, "valid": r.Valid, "errors": r.Errors})
			}
		}
		writeJSON(w, all)
		return
	}
	def, err := findDef(root, s.RepoRoot, coll)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	st := store.New(s.RepoRoot, def)
	recs, _ := st.LoadAll()
	fields := make([]*map[string]any, len(recs))
	ids := make([]string, len(recs))
	paths := make([]string, len(recs))
	for i, rec := range recs {
		f := rec.Fields
		fields[i] = &f
		ids[i] = rec.ID
		paths[i] = rec.Path
	}
	res := validate.ValidateCollection(def, fields, ids, paths)
	out := []map[string]any{}
	for _, r := range res {
		out = append(out, map[string]any{"id": r.ID, "valid": r.Valid, "errors": r.Errors})
	}
	writeJSON(w, out)
}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	root, err := config.LoadRoot(s.RepoRoot)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	stats := map[string]int{}
	for _, ref := range root.Collections {
		def, err := config.LoadCollectionDefinition(s.RepoRoot, root.DataDir, ref)
		if err != nil {
			continue
		}
		st := store.New(s.RepoRoot, def)
		recs, _ := st.LoadAll()
		stats[ref.Name] = len(recs)
	}
	writeJSON(w, stats)
}

func findDef(root *config.RootConfig, repoRoot, name string) (*config.CollectionDefinition, error) {
	for _, ref := range root.Collections {
		if ref.Name == name {
			return config.LoadCollectionDefinition(repoRoot, root.DataDir, ref)
		}
	}
	return nil, fmt.Errorf("collection not found: %s", name)
}
