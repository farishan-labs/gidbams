// Package mcp implements a minimal MCP (Model Context Protocol) server over stdio.
package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/farishan-labs/git-dbms/internal/config"
	"path/filepath"

	"github.com/farishan-labs/git-dbms/internal/sqlite"
	"github.com/farishan-labs/git-dbms/internal/store"
	"github.com/farishan-labs/git-dbms/internal/validate"
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
			"serverInfo":      map[string]any{"name": "git-dbms", "version": "0.1.0"},
			"capabilities":    map[string]any{"tools": map[string]any{}},
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
		{"name": "query", "description": "Run a read-only SQL query against the derived SQLite DB", "inputSchema": map[string]any{"type": "object", "properties": map[string]any{"sql": map[string]any{"type": "string"}}, "required": []string{"sql"}}},
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
			return "", fmt.Errorf("derived db not found: run git-dbms build first")
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
	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
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
