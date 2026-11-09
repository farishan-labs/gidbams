// Package cli implements the git-dbms command line.
package cli

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/farishan-labs/git-dbms/internal/config"
	"github.com/farishan-labs/git-dbms/internal/gitutil"
	"github.com/farishan-labs/git-dbms/internal/httpserver"
	"github.com/farishan-labs/git-dbms/internal/mcp"
	"github.com/farishan-labs/git-dbms/internal/record"
	"github.com/farishan-labs/git-dbms/internal/sqlite"
	"github.com/farishan-labs/git-dbms/internal/store"
	"github.com/farishan-labs/git-dbms/internal/triggers"
	"github.com/farishan-labs/git-dbms/internal/validate"
)

var repoRoot string
var initExample bool

func Execute() error {
	return rootCmd.Execute()
}

var rootCmd = &cobra.Command{
	Use:   "git-dbms",
	Short: "Git as a source-of-truth database engine",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if repoRoot == "" {
			wd, _ := os.Getwd()
			repoRoot = wd
		}
		abs, _ := filepath.Abs(repoRoot)
		repoRoot = abs
	},
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&repoRoot, "repo", "r", "", "repository root (default: cwd)")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVar(&initExample, "example", false, "scaffold an example collection")
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().StringVar(&validateFrom, "from", "", "validate files changed since this commit")
	validateCmd.Flags().StringVar(&validateTo, "to", "", "validate files changed up to this commit")
	validateCmd.Flags().BoolVar(&validateJSON, "json", false, "output results as JSON")
	validateCmd.Flags().BoolVar(&validateStrict, "strict", false, "fail on unknown fields")
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listJSON, "json", false, "output as JSON")
	listCmd.Flags().IntVar(&listLimit, "limit", 0, "max records (0=all)")
	listCmd.Flags().IntVar(&listOffset, "offset", 0, "skip records")
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(collectionsCmd)
	collectionsCmd.Flags().BoolVar(&collectionsJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(triggersCmd)
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(statsCmd)
	statsCmd.Flags().BoolVar(&statsJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(exportCmd)
	exportCmd.Flags().StringVar(&exportFormat, "format", "json", "output format: json or csv")
	rootCmd.AddCommand(rebuildCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(schemaCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(schemaValidateCmd)
	rootCmd.AddCommand(cleanCmd)
	rootCmd.AddCommand(rmCmd)
	rootCmd.AddCommand(collectionCmd)
	collectionCmd.AddCommand(collectionCreateCmd)
	collectionCmd.AddCommand(collectionDeleteCmd)
	collectionDeleteCmd.Flags().BoolVar(&collectionDeleteForce, "force", false, "confirm deletion")
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(fsckCmd)
	rootCmd.AddCommand(gcCmd)
	rootCmd.AddCommand(templateCmd)
	rootCmd.AddCommand(logCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(searchCmd)
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "output as JSON")
	searchCmd.Flags().StringVar(&searchCollection, "collection", "", "limit to collection")
	rootCmd.AddCommand(watchCmd)
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(migrateCmd)
	watchCmd.Flags().IntVar(&watchInterval, "interval", 2, "poll interval seconds")
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "port to listen on")
	serveCmd.Flags().StringVar(&serveToken, "token", os.Getenv("GIT_DBMS_TOKEN"), "bearer token")
	serveCmd.Flags().StringVar(&serveCert, "tls-cert", "", "TLS certificate file")
	serveCmd.Flags().StringVar(&serveKey, "tls-key", "", "TLS key file")
	serveCmd.Flags().StringVar(&serveCORS, "cors", "", "CORS allowed origin")
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("git-dbms 0.1.0")
	},
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a git-dbms repository (creates gitdb.yaml)",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := &config.RootConfig{Version: 1, DataDir: config.DefaultDataDir, Settings: config.Settings{DefaultBranch: "main", ValidateOnCI: true}}
		if err := config.SaveRoot(repoRoot, c); err != nil {
			return err
		}
		_ = os.MkdirAll(filepath.Join(repoRoot, config.DefaultDataDir), 0o755)
		if _, err := os.Stat(filepath.Join(repoRoot, ".git")); err != nil {
			_ = exec.Command("git", "-C", repoRoot, "init").Run()
		}
		if initExample {
			exDir := filepath.Join(repoRoot, config.DefaultDataDir, "items")
			_ = os.MkdirAll(exDir, 0o755)
			_ = os.WriteFile(filepath.Join(exDir, "schema.yaml"), []byte("name: items\nrecord_file:\n  format: md\n  name: \"{id}.md\"\ncolumns:\n  - name: id\n    type: string\n    required: true\n    unique: true\n  - name: name\n    type: string\n    required: true\n"), 0o644)
			_ = os.WriteFile(filepath.Join(exDir, "hello.md"), []byte("---\nid: hello\nname: Hello\n---\nBody\n"), 0o644)
			// register collection
			c.Collections = append(c.Collections, config.CollectionRef{Name: "items", Path: "items"})
			_ = config.SaveRoot(repoRoot, c)
		}
		fmt.Println("initialized git-dbms repo at", repoRoot)
		return nil
	},
}

var validateFrom, validateTo string
var validateJSON bool
var validateStrict bool

var validateCmd = &cobra.Command{
	Use:   "validate [collection]",
	Short: "Validate records against schemas",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		collections := root.Collections
		if len(args) > 0 {
			filtered := []config.CollectionRef{}
			for _, c := range collections {
				if c.Name == args[0] {
					filtered = append(filtered, c)
				}
			}
			collections = filtered
		}

		var changed map[string]bool
		if validateFrom != "" || validateTo != "" {
			cf, err := gitutil.ChangedFiles(repoRoot, validateFrom, validateTo)
			if err != nil {
				return err
			}
			changed = map[string]bool{}
			for _, f := range cf {
				changed[f] = true
			}
		}
		anyInvalid := false
		allResults := []validate.Result{}
		for _, ref := range collections {
			def, err := config.LoadCollectionDefinition(repoRoot, root.DataDir, ref)
			if err != nil {
				return err
			}
			st := store.New(repoRoot, def)
			recs, err := st.LoadAll()
			if err != nil {
				return err
			}
			if changed != nil {
				filteredRecs := []*record.Record{}
				for _, r := range recs {
					rel, _ := filepath.Rel(repoRoot, r.Path)
					if changed[rel] || changed[r.Path] {
						filteredRecs = append(filteredRecs, r)
					}
				}
				recs = filteredRecs
			}
			if validateStrict {
				colNames := map[string]bool{}
				for _, c := range def.Columns {
					colNames[c.Name] = true
				}
				for _, r := range recs {
					for k := range r.Fields {
						if !colNames[k] {
							fmt.Printf("unknown field %q in %s\n", k, r.ID)
							anyInvalid = true
						}
					}
				}
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
			allResults = append(allResults, results...)
			msg := validate.FormatErrors(results)
			if msg != "" {
				anyInvalid = true
				if !validateJSON {
					fmt.Print(msg)
				}
			} else if !validateJSON {
				fmt.Printf("%s: %d records OK\n", ref.Name, len(recs))
			}
		}
		if validateJSON {
			b, _ := json.MarshalIndent(allResults, "", "  ")
			fmt.Println(string(b))
		}
		if anyInvalid {
			return fmt.Errorf("validation failed")
		}
		return nil
	},
}

var listCmd = &cobra.Command{
	Use:   "list <collection>",
	Short: "List record ids in a collection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		def, err := findDef(root, args[0])
		if err != nil {
			return err
		}
		st := store.New(repoRoot, def)
		recs, err := st.LoadAll()
		if err != nil {
			return err
		}
		if listOffset > len(recs) {
			listOffset = len(recs)
		}
		end := len(recs)
		if listLimit > 0 && listOffset+listLimit < end {
			end = listOffset + listLimit
		}
		recs = recs[listOffset:end]
		if listJSON {
			ids := []string{}
			for _, r := range recs {
				ids = append(ids, r.ID)
			}
			b, _ := json.MarshalIndent(ids, "", "  ")
			fmt.Println(string(b))
			return nil
		}
		for _, r := range recs {
			fmt.Println(r.ID)
		}
		return nil
	},
}

var getCmd = &cobra.Command{
	Use:   "get <collection> <id>",
	Short: "Print a record as JSON",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		def, err := findDef(root, args[0])
		if err != nil {
			return err
		}
		st := store.New(repoRoot, def)
		r, err := st.Get(args[1])
		if err != nil {
			return err
		}
		b, _ := json.MarshalIndent(r.Fields, "", "  ")
		fmt.Println(string(b))
		if r.Body != "" {
			fmt.Println("---")
			fmt.Print(r.Body)
		}
		return nil
	},
}

var createCmd = &cobra.Command{
	Use:   "create <collection> <id> [json-file]",
	Short: "Create a record from JSON on stdin or file",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		def, err := findDef(root, args[0])
		if err != nil {
			return err
		}
		st := store.New(repoRoot, def)
		fields := map[string]any{}
		if len(args) == 3 {
			b, err := os.ReadFile(args[2])
			if err != nil {
				return err
			}
			if err := json.Unmarshal(b, &fields); err != nil {
				return err
			}
		} else {
			// read stdin
			dec := json.NewDecoder(os.Stdin)
			if err := dec.Decode(&fields); err != nil {
				return err
			}
		}
		r, err := st.Create(args[1], fields, "")
		if err != nil {
			return err
		}
		fmt.Println("created", r.ID, r.Path)
		return nil
	},
}

var updateCmd = &cobra.Command{
	Use:   "update <collection> <id> [json-file]",
	Short: "Update a record",
	Args:  cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		def, err := findDef(root, args[0])
		if err != nil {
			return err
		}
		st := store.New(repoRoot, def)
		fields := map[string]any{}
		if len(args) == 3 {
			b, err := os.ReadFile(args[2])
			if err != nil {
				return err
			}
			if err := json.Unmarshal(b, &fields); err != nil {
				return err
			}
		} else {
			dec := json.NewDecoder(os.Stdin)
			if err := dec.Decode(&fields); err != nil {
				return err
			}
		}
		r, err := st.Update(args[1], fields, nil)
		if err != nil {
			return err
		}
		fmt.Println("updated", r.ID)
		return nil
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete <collection> <id>",
	Short: "Delete a record",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		def, err := findDef(root, args[0])
		if err != nil {
			return err
		}
		st := store.New(repoRoot, def)
		if err := st.Delete(args[1]); err != nil {
			return err
		}
		fmt.Println("deleted", args[1])
		return nil
	},
}

var buildCmd = &cobra.Command{
	Use:   "build [db-path]",
	Short: "Build derived SQLite database from collections",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath := filepath.Join(repoRoot, ".gitdb", "data.db")
		if len(args) > 0 {
			dbPath = args[0]
		}
		// env override
		if v := os.Getenv("GIT_DBMS_DB"); v != "" {
			dbPath = v
		}
		if err := buildDerivedDB(repoRoot, dbPath); err != nil {
			return err
		}
		fmt.Println("built", dbPath)
		return nil
	},
}

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run MCP server over stdio",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mcp.New(repoRoot)
		return s.Run()
	},
}

var collectionsCmd = &cobra.Command{
	Use:   "collections",
	Short: "List collections",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		if collectionsJSON {
			names := []string{}
			for _, c := range root.Collections {
				names = append(names, c.Name)
			}
			b, _ := json.MarshalIndent(names, "", "  ")
			fmt.Println(string(b))
			return nil
		}
		for _, c := range root.Collections {
			fmt.Println(c.Name, c.Path)
		}
		return nil
	},
}

func findDef(root *config.RootConfig, name string) (*config.CollectionDefinition, error) {
	for _, ref := range root.Collections {
		if ref.Name == name {
			return config.LoadCollectionDefinition(repoRoot, root.DataDir, ref)
		}
	}
	// maybe collection dir exists but not in config; synthesize
	return nil, fmt.Errorf("collection not found: %s", name)
}

var triggersCmd = &cobra.Command{
	Use:   "triggers run <event>",
	Short: "Run triggers for an event (created|updated|deleted|any)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		if args[0] != "run" {
			return fmt.Errorf("usage: triggers run <event>")
		}
		return triggers.Run(repoRoot, args[1])
	},
}

func buildDerivedDB(repoRoot, dbPath string) error {
	root, err := config.LoadRoot(repoRoot)
	if err != nil {
		return err
	}
	b := sqlite.New(repoRoot, dbPath)
	type coll struct {
		Def   *config.CollectionDefinition
		Store *store.Store
	}
	colls := make([]struct {
		Def   *config.CollectionDefinition
		Store *store.Store
	}, 0, len(root.Collections))
	for _, ref := range root.Collections {
		def, err := config.LoadCollectionDefinition(repoRoot, root.DataDir, ref)
		if err != nil {
			return err
		}
		colls = append(colls, struct {
			Def   *config.CollectionDefinition
			Store *store.Store
		}{Def: def, Store: store.New(repoRoot, def)})
	}
	return b.Build(colls)
}

var queryCmd = &cobra.Command{
	Use:   "query <sql>",
	Short: "Run a read-only SQL query against the derived SQLite DB",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath := filepath.Join(repoRoot, ".gitdb", "data.db")
		if v := os.Getenv("GIT_DBMS_DB"); v != "" {
			dbPath = v
		}
		if _, err := os.Stat(dbPath); err != nil {
			if err := buildDerivedDB(repoRoot, dbPath); err != nil {
				return err
			}
		}
		out, err := sqlite.Query(dbPath, args[0])
		if err != nil {
			return err
		}
		fmt.Println(out)
		return nil
	},
}

var statsJSON bool

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Print collection record counts",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		stats := map[string]int{}
		for _, ref := range root.Collections {
			if searchCollection != "" && ref.Name != searchCollection {
				continue
			}
			def, err := config.LoadCollectionDefinition(repoRoot, root.DataDir, ref)
			if err != nil {
				return err
			}
			st := store.New(repoRoot, def)
			recs, err := st.LoadAll()
			if err != nil {
				return err
			}
			stats[ref.Name] = len(recs)
			if !statsJSON {
				fmt.Printf("%s: %d records\n", ref.Name, len(recs))
			}
		}
		if statsJSON {
			b, _ := json.MarshalIndent(stats, "", "  ")
			fmt.Println(string(b))
		}
		return nil
	},
}

var exportCmd = &cobra.Command{
	Use:   "export <collection>",
	Short: "Export a collection as JSON array",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		def, err := findDef(root, args[0])
		if err != nil {
			return err
		}
		st := store.New(repoRoot, def)
		recs, err := st.LoadAll()
		if err != nil {
			return err
		}
		if exportFormat == "csv" {
			w := csv.NewWriter(os.Stdout)
			// header from first record or columns
			header := []string{}
			for _, c := range def.Columns {
				header = append(header, c.Name)
			}
			_ = w.Write(header)
			for _, r := range recs {
				row := []string{}
				for _, c := range def.Columns {
					row = append(row, fmt.Sprintf("%v", r.Fields[c.Name]))
				}
				_ = w.Write(row)
			}
			w.Flush()
			return nil
		}
		out := []map[string]any{}
		for _, r := range recs {
			out = append(out, r.Fields)
		}
		b, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(b))
		return nil
	},
}

var rebuildCmd = &cobra.Command{
	Use:   "rebuild",
	Short: "Validate then build derived SQLite database",
	RunE: func(cmd *cobra.Command, args []string) error {
		// validate
		if err := validateCmd.RunE(cmd, nil); err != nil {
			return err
		}
		dbPath := filepath.Join(repoRoot, ".gitdb", "data.db")
		if v := os.Getenv("GIT_DBMS_DB"); v != "" {
			dbPath = v
		}
		return buildDerivedDB(repoRoot, dbPath)
	},
}

var servePort int
var serveToken string
var serveCert, serveKey string
var serveCORS string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run HTTP API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		srv := httpserver.New(repoRoot, serveToken)
		addr := fmt.Sprintf(":%d", servePort)
		fmt.Printf("git-dbms serve listening on %s\n", addr)
		handler := srv.Handler()
		if serveCORS != "" {
			handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Access-Control-Allow-Origin", serveCORS)
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Authorization,Content-Type")
				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusNoContent)
					return
				}
				srv.Handler().ServeHTTP(w, r)
			})
		}
		httpSrv := &http.Server{Addr: addr, Handler: handler}
		go func() {
			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			<-sigCh
			_ = httpSrv.Shutdown(context.Background())
		}()
		if serveCert != "" && serveKey != "" {
			return httpSrv.ListenAndServeTLS(serveCert, serveKey)
		}
		return httpSrv.ListenAndServe()
	},
}

var schemaCmd = &cobra.Command{
	Use:   "schema <collection>",
	Short: "Print a collection schema",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		def, err := findDef(root, args[0])
		if err != nil {
			return err
		}
		b, _ := json.MarshalIndent(def, "", "  ")
		fmt.Println(string(b))
		return nil
	},
}

var diffCmd = &cobra.Command{
	Use:   "diff <from> [to]",
	Short: "List records changed between commits",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		to := ""
		if len(args) == 2 {
			to = args[1]
		}
		files, err := gitutil.ChangedFiles(repoRoot, args[0], to)
		if err != nil {
			return err
		}
		for _, f := range files {
			fmt.Println(f)
		}
		return nil
	},
}

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Alias for doctor",
	RunE:  doctorCmd.RunE,
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Print root configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		b, _ := json.MarshalIndent(root, "", "  ")
		fmt.Println(string(b))
		return nil
	},
}

var listJSON bool
var listLimit, listOffset int
var searchJSON bool
var searchCollection string
var collectionsJSON bool

var searchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search all records across collections",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		q := strings.ToLower(args[0])
		results := []map[string]any{}
		for _, ref := range root.Collections {
			if searchCollection != "" && ref.Name != searchCollection {
				continue
			}
			def, err := config.LoadCollectionDefinition(repoRoot, root.DataDir, ref)
			if err != nil {
				return err
			}
			st := store.New(repoRoot, def)
			recs, err := st.LoadAll()
			if err != nil {
				return err
			}
			for _, r := range recs {
				jb, _ := json.Marshal(r.Fields)
				if strings.Contains(strings.ToLower(r.ID), q) || strings.Contains(strings.ToLower(string(jb)), q) || strings.Contains(strings.ToLower(r.Body), q) {
					if searchJSON {
						entry := map[string]any{"collection": ref.Name}
						for k, v := range r.Fields {
							entry[k] = v
						}
						results = append(results, entry)
					} else {
						fmt.Printf("%s/%s\n", ref.Name, r.ID)
					}
				}
			}
		}
		if searchJSON {
			b, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(b))
		}
		return nil
	},
}

var watchInterval int

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch data files and rebuild derived DB on change",
	RunE: func(cmd *cobra.Command, args []string) error {
		last := map[string]int64{}
		scan := func() map[string]int64 {
			m := map[string]int64{}
			root, err := config.LoadRoot(repoRoot)
			if err != nil {
				return m
			}
			for _, ref := range root.Collections {
				def, err := config.LoadCollectionDefinition(repoRoot, root.DataDir, ref)
				if err != nil {
					continue
				}
				dir := def.DataDir
				if !filepath.IsAbs(dir) {
					dir = filepath.Join(repoRoot, dir)
				}
				filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
					if err != nil || info.IsDir() {
						return nil
					}
					m[p] = info.ModTime().UnixNano()
					return nil
				})
			}
			return m
		}
		fmt.Println("watching...")
		for {
			cur := scan()
			changed := false
			for p, t := range cur {
				if last[p] != t {
					changed = true
					break
				}
			}
			if len(cur) != len(last) {
				changed = true
			}
			if changed && len(last) > 0 {
				fmt.Println("change detected, rebuilding...")
				if err := rebuildCmd.RunE(cmd, nil); err != nil {
					fmt.Println("rebuild error:", err)
				} else {
					fmt.Println("rebuilt")
				}
			}
			last = cur
			time.Sleep(time.Duration(watchInterval) * time.Second)
		}
	},
}

var backupCmd = &cobra.Command{
	Use:   "backup [file]",
	Short: "Create a tar.gz backup of the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		out := filepath.Join(repoRoot, "git-dbms-backup.tar.gz")
		if len(args) > 0 {
			out = args[0]
		}
		f, err := os.Create(out)
		if err != nil {
			return err
		}
		defer f.Close()
		gz := gzip.NewWriter(f)
		defer gz.Close()
		tw := tar.NewWriter(gz)
		defer tw.Close()
		err = filepath.Walk(repoRoot, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				if filepath.Base(path) == ".git" {
					return filepath.SkipDir
				}
				return nil
			}
			rel, err := filepath.Rel(repoRoot, path)
			if err != nil {
				return err
			}
			hdr, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			hdr.Name = rel
			if err := tw.WriteHeader(hdr); err != nil {
				return err
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			_, err = tw.Write(data)
			return err
		})
		if err != nil {
			return err
		}
		fmt.Println("backup written to", out)
		return nil
	},
}

var migrateCmd = &cobra.Command{
	Use:   "migrate <collection> <source-dir>",
	Short: "Import records from a directory into a collection",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		def, err := findDef(root, args[0])
		if err != nil {
			return err
		}
		src := args[1]
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			ext := strings.ToLower(filepath.Ext(e.Name()))
			if ext != ".md" && ext != ".json" && ext != ".yaml" && ext != ".yml" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(src, e.Name()))
			if err != nil {
				return err
			}
			dest := filepath.Join(def.DataDir, e.Name())
			if !filepath.IsAbs(dest) {
				dest = filepath.Join(repoRoot, dest)
			}
			if err := os.WriteFile(dest, data, 0o644); err != nil {
				return err
			}
			fmt.Println("imported", e.Name())
		}
		return validateCmd.RunE(cmd, []string{args[0]})
	},
}

var fsckCmd = &cobra.Command{
	Use:   "fsck",
	Short: "Run git fsck on the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := exec.Command("git", "-C", repoRoot, "fsck", "--full")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var schemaValidateCmd = &cobra.Command{
	Use:   "schema-validate",
	Short: "Validate collection schemas",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		for _, ref := range root.Collections {
			if searchCollection != "" && ref.Name != searchCollection {
				continue
			}
			def, err := config.LoadCollectionDefinition(repoRoot, root.DataDir, ref)
			if err != nil {
				return fmt.Errorf("%s: %w", ref.Name, err)
			}
			if def.Name == "" {
				return fmt.Errorf("%s: missing name", ref.Name)
			}
			fmt.Printf("OK %s (%d columns)\n", ref.Name, len(def.Columns))
		}
		return nil
	},
}

var cleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Remove derived database and build artifacts",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath := filepath.Join(repoRoot, ".gitdb", "data.db")
		if v := os.Getenv("GIT_DBMS_DB"); v != "" {
			dbPath = v
		}
		if err := os.Remove(dbPath); err != nil && !os.IsNotExist(err) {
			return err
		}
		fmt.Println("cleaned", dbPath)
		return nil
	},
}

var exportFormat string

var rmCmd = &cobra.Command{
	Use:   "rm <collection> <id>",
	Short: "Alias for delete",
	Args:  cobra.ExactArgs(2),
	RunE:  deleteCmd.RunE,
}

var collectionCmd = &cobra.Command{
	Use:   "collection",
	Short: "Manage collections",
}

var collectionCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new collection scaffold",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		dir := filepath.Join(repoRoot, root.DataDir, name)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		schema := fmt.Sprintf("name: %s\nrecord_file:\n  format: md\n  name: \"{id}.md\"\ncolumns:\n  - name: id\n    type: string\n    required: true\n    unique: true\n  - name: name\n    type: string\n    required: true\n", name)
		if err := os.WriteFile(filepath.Join(dir, "schema.yaml"), []byte(schema), 0o644); err != nil {
			return err
		}
		// register if missing
		found := false
		for _, c := range root.Collections {
			if c.Name == name {
				found = true
				break
			}
		}
		if !found {
			root.Collections = append(root.Collections, config.CollectionRef{Name: name, Path: name})
			if err := config.SaveRoot(repoRoot, root); err != nil {
				return err
			}
		}
		fmt.Println("created collection", name)
		return nil
	},
}

var gcCmd = &cobra.Command{
	Use:   "gc",
	Short: "Run git gc on the repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := exec.Command("git", "-C", repoRoot, "gc", "--prune=now")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var templateCmd = &cobra.Command{
	Use:   "template",
	Short: "Print a sample schema.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print(`name: items
record_file:
  format: md
  name: "{id}.md"
columns:
  - name: id
    type: string
    required: true
    unique: true
  - name: name
    type: string
    required: true
`)
		return nil
	},
}

var editCmd = &cobra.Command{
	Use:   "edit <collection> <id>",
	Short: "Open a record in $EDITOR",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		def, err := findDef(root, args[0])
		if err != nil {
			return err
		}
		st := store.New(repoRoot, def)
		r, err := st.Get(args[1])
		if err != nil {
			return err
		}
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		c := exec.Command(editor, r.Path)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var logCmd = &cobra.Command{
	Use:   "log [count]",
	Short: "Show recent git commits",
	RunE: func(cmd *cobra.Command, args []string) error {
		n := "10"
		if len(args) > 0 {
			n = args[0]
		}
		c := exec.Command("git", "-C", repoRoot, "log", "--oneline", "-n", n)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show git status",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := exec.Command("git", "-C", repoRoot, "status", "--short")
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var collectionDeleteForce bool

var collectionDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a collection (requires --force)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !collectionDeleteForce {
			return fmt.Errorf("refusing to delete without --force")
		}
		name := args[0]
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		def, err := findDef(root, name)
		if err != nil {
			return err
		}
		dir := def.DataDir
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(repoRoot, dir)
		}
		if err := os.RemoveAll(dir); err != nil {
			return err
		}
		// remove from config
		filtered := []config.CollectionRef{}
		for _, c := range root.Collections {
			if c.Name != name {
				filtered = append(filtered, c)
			}
		}
		root.Collections = filtered
		if err := config.SaveRoot(repoRoot, root); err != nil {
			return err
		}
		fmt.Println("deleted collection", name)
		return nil
	},
}
