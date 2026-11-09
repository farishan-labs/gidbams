// Package cli implements the git-dbms command line.
package cli

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

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
	rootCmd.AddCommand(validateCmd)
	validateCmd.Flags().StringVar(&validateFrom, "from", "", "validate files changed since this commit")
	validateCmd.Flags().StringVar(&validateTo, "to", "", "validate files changed up to this commit")
	validateCmd.Flags().BoolVar(&validateJSON, "json", false, "output results as JSON")
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(getCmd)
	rootCmd.AddCommand(createCmd)
	rootCmd.AddCommand(updateCmd)
	rootCmd.AddCommand(deleteCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(mcpCmd)
	rootCmd.AddCommand(collectionsCmd)
	rootCmd.AddCommand(triggersCmd)
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(queryCmd)
	rootCmd.AddCommand(statsCmd)
	rootCmd.AddCommand(exportCmd)
	rootCmd.AddCommand(rebuildCmd)
	rootCmd.AddCommand(serveCmd)
	rootCmd.AddCommand(schemaCmd)
	rootCmd.AddCommand(diffCmd)
	rootCmd.AddCommand(checkCmd)
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "port to listen on")
	serveCmd.Flags().StringVar(&serveToken, "token", os.Getenv("GIT_DBMS_TOKEN"), "bearer token")
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
		fmt.Println("initialized git-dbms repo at", repoRoot)
		return nil
	},
}

var validateFrom, validateTo string
var validateJSON bool

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

var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Print collection record counts",
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := config.LoadRoot(repoRoot)
		if err != nil {
			return err
		}
		for _, ref := range root.Collections {
			def, err := config.LoadCollectionDefinition(repoRoot, root.DataDir, ref)
			if err != nil {
				return err
			}
			st := store.New(repoRoot, def)
			recs, err := st.LoadAll()
			if err != nil {
				return err
			}
			fmt.Printf("%s: %d records\n", ref.Name, len(recs))
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

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run HTTP API server",
	RunE: func(cmd *cobra.Command, args []string) error {
		srv := httpserver.New(repoRoot, serveToken)
		addr := fmt.Sprintf(":%d", servePort)
		fmt.Printf("git-dbms serve listening on %s\n", addr)
		return http.ListenAndServe(addr, srv.Handler())
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
