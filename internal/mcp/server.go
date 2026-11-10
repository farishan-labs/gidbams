// Package mcp implements a minimal MCP (Model Context Protocol) server over stdio.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime/debug"
	"strings"

	"github.com/farishan-labs/gidbams/internal/config"
	"path/filepath"

	"github.com/farishan-labs/gidbams/internal/sqlite"
	"github.com/farishan-labs/gidbams/internal/store"
	"github.com/farishan-labs/gidbams/internal/validate"
)

type Server struct {
	RepoRoot string
	Stdin    io.Reader
	Stdout   io.Writer
}

type rpcReq struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}

type rpcResp struct {
	JSONRPC string `json:"jsonrpc"`
	ID      any    `json:"id"`
	Result  any    `json:"result,omitempty"`
	Error   any    `json:"error,omitempty"`
}

func New(repoRoot string) *Server {
	return &Server{RepoRoot: repoRoot, Stdin: os.Stdin, Stdout: os.Stdout}
}

func (s *Server) Run() error {
	r := bufio.NewReader(s.Stdin)
	for {
		line, err := r.ReadString('\n')
		if err != nil && line == "" {
			if err == io.EOF {
				return nil
			}
			return err
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var req rpcReq
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			continue
		}
		resp := s.handle(req)
		b, _ := json.Marshal(resp)
		fmt.Fprintf(s.Stdout, "%s\n", b)
	}
}

func (s *Server) handle(req rpcReq) rpcResp {
	switch req.Method {
	case "initialize":
		return rpcResp{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"protocolVersion": "2024-11-05",
			"serverInfo":      map[string]any{"name": "gidbams", "version": "0.1.0"},
			"capabilities":    map[string]any{"tools": map[string]any{}, "resources": map[string]any{}},
		}}
	case "tools/list":
		return rpcResp{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"tools": toolDefs()}}
	case "tools/call":
		var p struct {
			Name      string         `json:"name"`
			Arguments map[string]any `json:"arguments"`
		}
		_ = json.Unmarshal(req.Params, &p)
		res, err := s.callTool(p.Name, p.Arguments)
		if err != nil {
			return rpcResp{JSONRPC: "2.0", ID: req.ID, Error: map[string]any{"code": -32000, "message": err.Error()}}
		}
		return rpcResp{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"content": []map[string]any{{"type": "text", "text": res}},
		}}
	case "resources/list":
		root, _ := config.LoadRoot(s.RepoRoot)
		resources := []map[string]any{}
		if root != nil {
			for _, ref := range root.Collections {
				resources = append(resources, map[string]any{
					"uri":      "gitdb://" + ref.Name,
					"name":     ref.Name,
					"mimeType": "application/json",
				})
			}
		}
		return rpcResp{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"resources": resources}}
	case "resources/read":
		var p struct {
			URI string `json:"uri"`
		}
		_ = json.Unmarshal(req.Params, &p)
		// expected gitdb://collection/id
		uri := strings.TrimPrefix(p.URI, "gitdb://")
		parts := strings.SplitN(uri, "/", 2)
		if len(parts) < 2 {
			return rpcResp{JSONRPC: "2.0", ID: req.ID, Error: map[string]any{"code": -32000, "message": "invalid uri"}}
		}
		coll, id := parts[0], parts[1]
		def, err := findDef(s.RepoRoot, rootOrLoad(s.RepoRoot), coll)
		if err != nil {
			return rpcResp{JSONRPC: "2.0", ID: req.ID, Error: map[string]any{"code": -32000, "message": err.Error()}}
		}
		st := store.New(s.RepoRoot, def)
		rec, err := st.Get(id)
		if err != nil {
			return rpcResp{JSONRPC: "2.0", ID: req.ID, Error: map[string]any{"code": -32000, "message": err.Error()}}
		}
		b, _ := json.Marshal(rec.Fields)
		return rpcResp{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{
			"contents": []map[string]any{{"uri": p.URI, "mimeType": "application/json", "text": string(b)}},
		}}
	default:
		return rpcResp{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{}}
	}
}

func toolDefs() []map[string]any {
	return []map[string]any{
		{"name": "list_collections", "description": "List collections in the repo", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{}}},
		{"name": "list_records", "description": "List record ids in a collection", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"collection": map[string]any{"type": "string"}}, "required": []string{"collection"}}},
		{"name": "get_record", "description": "Get a record by collection and id", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"collection": map[string]any{"type": "string"}, "id": map[string]any{"type": "string"}}, "required": []string{"collection", "id"}}},
		{"name": "validate", "description": "Validate all records in a collection", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"collection": map[string]any{"type": "string"}}}},
		{"name": "search_records", "description": "Search records in a collection by substring", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"collection": map[string]any{"type": "string"}, "query": map[string]any{"type": "string"}}, "required": []string{"collection", "query"}}},
		{"name": "search", "description": "Search all collections by substring", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"query": map[string]any{"type": "string"}}, "required": []string{"query"}}},
		{"name": "query", "description": "Run a read-only SQL query against the derived SQLite DB", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"sql": map[string]any{"type": "string"}}, "required": []string{"sql"}}},
		{"name": "build", "description": "Rebuild the derived SQLite database", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{}}},
		{"name": "stats", "description": "Return record counts per collection", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{}}},
		{"name": "doctor", "description": "Check repository health", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{}}},
		{"name": "schema", "description": "Show a collection schema", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"collection": map[string]any{"type": "string"}}, "required": []string{"collection"}}},
		{"name": "version", "description": "Return build version", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{}}},
	}
}

func (s *Server) callTool(name string, args map[string]any) (string, error) {
	root, err := config.LoadRoot(s.RepoRoot)
	if err != nil {
		return "", err
	}
	switch name {
	case "list_collections":
		out := []string{}
		for _, c := range root.Collections {
			out = append(out, c.Name)
		}
		b, _ := json.Marshal(out)
		return string(b), nil
	case "list_records":
		coll := strArg(args, "collection")
		def, err := findDef(s.RepoRoot, root, coll)
		if err != nil {
			return "", err
		}
		st := store.New(s.RepoRoot, def)
		recs, err := st.LoadAll()
		if err != nil {
			return "", err
		}
		ids := []string{}
		for _, r := range recs {
			ids = append(ids, r.ID)
		}
		b, _ := json.Marshal(ids)
		return string(b), nil
	case "get_record":
		coll := strArg(args, "collection")
		id := strArg(args, "id")
		def, err := findDef(s.RepoRoot, root, coll)
		if err != nil {
			return "", err
		}
		st := store.New(s.RepoRoot, def)
		r, err := st.Get(id)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(r.Fields)
		return string(b), nil
	case "validate":
		coll := strArg(args, "collection")
		def, err := findDef(s.RepoRoot, root, coll)
		if err != nil {
			return "", err
		}
		st := store.New(s.RepoRoot, def)
		recs, err := st.LoadAll()
		if err != nil {
			return "", err
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
		results := validate.ValidateCollection(def, fields, ids, paths)
		b, _ := json.Marshal(results)
		return string(b), nil
	case "query":
		sqlStr := strArg(args, "sql")
		dbPath := filepath.Join(s.RepoRoot, ".gitdb", "data.db")
		if v := os.Getenv("GIT_DBMS_DB"); v != "" {
			dbPath = v
		}
		if _, err := os.Stat(dbPath); err != nil {
			// build
			// use cli? avoid import cycle; build via sqlite.Builder not collections. Skip: require prebuilt.
			return "", fmt.Errorf("derived db not found: run gidbams build first")
		}
		out, err := sqlite.Query(dbPath, sqlStr)
		if err != nil {
			return "", err
		}
		return out, nil
	case "search_records":
		coll := strArg(args, "collection")
		q := strArg(args, "query")
		def, err := findDef(s.RepoRoot, root, coll)
		if err != nil {
			return "", err
		}
		st := store.New(s.RepoRoot, def)
		recs, err := st.LoadAll()
		if err != nil {
			return "", err
		}
		out := []map[string]any{}
		lq := strings.ToLower(q)
		for _, r := range recs {
			match := strings.Contains(strings.ToLower(r.ID), lq)
			if !match {
				jb, _ := json.Marshal(r.Fields)
				if strings.Contains(strings.ToLower(string(jb)), lq) {
					match = true
				}
			}
			if !match && strings.Contains(strings.ToLower(r.Body), lq) {
				match = true
			}
			if match {
				out = append(out, r.Fields)
			}
		}
		b, _ := json.Marshal(out)
		return string(b), nil
	case "search":
		q := strings.ToLower(strArg(args, "query"))
		root, err := config.LoadRoot(s.RepoRoot)
		if err != nil {
			return "", err
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
			for _, r := range recs {
				jb, _ := json.Marshal(r.Fields)
				if strings.Contains(strings.ToLower(r.ID), q) || strings.Contains(strings.ToLower(string(jb)), q) || strings.Contains(strings.ToLower(r.Body), q) {
					entry := map[string]any{"collection": ref.Name}
					for k, v := range r.Fields {
						entry[k] = v
					}
					out = append(out, entry)
				}
			}
		}
		b, _ := json.Marshal(out)
		return string(b), nil
	case "build":
		root, err := config.LoadRoot(s.RepoRoot)
		if err != nil {
			return "", err
		}
		dbPath := filepath.Join(s.RepoRoot, ".gitdb", "data.db")
		if v := os.Getenv("GIT_DBMS_DB"); v != "" {
			dbPath = v
		}
		b := sqlite.New(s.RepoRoot, dbPath)
		colls := make([]struct {
			Def   *config.CollectionDefinition
			Store *store.Store
		}, 0, len(root.Collections))
		for _, ref := range root.Collections {
			def, err := config.LoadCollectionDefinition(s.RepoRoot, root.DataDir, ref)
			if err != nil {
				return "", err
			}
			colls = append(colls, struct {
				Def   *config.CollectionDefinition
				Store *store.Store
			}{Def: def, Store: store.New(s.RepoRoot, def)})
		}
		if err := b.Build(colls); err != nil {
			return "", err
		}
		return `{"status":"ok"}`, nil
	case "stats":
		root, err := config.LoadRoot(s.RepoRoot)
		if err != nil {
			return "", err
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
		b, _ := json.Marshal(stats)
		return string(b), nil
	case "doctor":
		root, err := config.LoadRoot(s.RepoRoot)
		if err != nil {
			return "", err
		}
		results := []map[string]any{}
		for _, ref := range root.Collections {
			def, err := config.LoadCollectionDefinition(s.RepoRoot, root.DataDir, ref)
			status := "ok"
			if err != nil {
				status = "error: " + err.Error()
			}
			results = append(results, map[string]any{"collection": ref.Name, "status": status, "columns": len(def.Columns)})
		}
		b, _ := json.Marshal(results)
		return string(b), nil
	case "schema":
		coll := strArg(args, "collection")
		root, err := config.LoadRoot(s.RepoRoot)
		if err != nil {
			return "", err
		}
		def, err := findDef(s.RepoRoot, root, coll)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(def)
		return string(b), nil
	case "version":
		info, ok := debug.ReadBuildInfo()
		if ok {
			b, _ := json.Marshal(map[string]string{"version": info.Main.Version, "go": info.GoVersion})
			return string(b), nil
		}
		return `{"version":"dev"}`, nil
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

func rootOrLoad(repoRoot string) *config.RootConfig {
	root, _ := config.LoadRoot(repoRoot)
	return root
}

func findDef(repoRoot string, root *config.RootConfig, name string) (*config.CollectionDefinition, error) {
	for _, ref := range root.Collections {
		if ref.Name == name {
			return config.LoadCollectionDefinition(repoRoot, root.DataDir, ref)
		}
	}
	return nil, fmt.Errorf("collection not found: %s", name)
}

func strArg(args map[string]any, key string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}
