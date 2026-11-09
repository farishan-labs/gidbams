package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/farishan-labs/git-dbms/internal/config"
)

// doctorCmd is defined in cli.go? We'll add via init in this file's init.
func init() {
	// register after rootCmd exists
}

var doctorJSON bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check repository health",
	RunE: func(cmd *cobra.Command, args []string) error {
		type checkResult struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Error  string `json:"error,omitempty"`
		}
		ok := true
		results := []checkResult{}
		record := func(name string, err error) {
			if err != nil {
				ok = false
				results = append(results, checkResult{Name: name, Status: "FAIL", Error: err.Error()})
				if !doctorJSON {
					fmt.Printf("FAIL %s: %v\n", name, err)
				}
			} else {
				results = append(results, checkResult{Name: name, Status: "OK"})
				if !doctorJSON {
					fmt.Printf("OK   %s\n", name)
				}
			}
		}
		root, err := config.LoadRoot(repoRoot)
		record("load root config", err)
		if err == nil {
			for _, ref := range root.Collections {
				def, derr := config.LoadCollectionDefinition(repoRoot, root.DataDir, ref)
				record("collection "+ref.Name, derr)
				if derr == nil {
					dir := def.DataDir
					if !filepath.IsAbs(dir) {
						dir = filepath.Join(repoRoot, dir)
					}
					if _, err := os.Stat(dir); err != nil {
						record("data dir "+ref.Name, err)
					} else {
						record("data dir "+ref.Name, nil)
					}
				}
			}
		}
		if _, err := os.Stat(filepath.Join(repoRoot, ".git")); err != nil {
			results = append(results, checkResult{Name: "git repository", Status: "WARN", Error: "not a git repository"})
			if !doctorJSON {
				fmt.Println("WARN not a git repository")
			}
		} else {
			results = append(results, checkResult{Name: "git repository", Status: "OK"})
			if !doctorJSON {
				fmt.Println("OK   git repository")
			}
			// remote check
			remoteOut, remoteErr := exec.Command("git", "-C", repoRoot, "remote", "get-url", "origin").Output()
			if remoteErr != nil {
				results = append(results, checkResult{Name: "git remote", Status: "WARN", Error: "no origin remote"})
				if !doctorJSON {
					fmt.Println("WARN no origin remote")
				}
			} else {
				results = append(results, checkResult{Name: "git remote", Status: "OK", Error: strings.TrimSpace(string(remoteOut))})
				if !doctorJSON {
					fmt.Printf("OK   origin %s\n", strings.TrimSpace(string(remoteOut)))
				}
			}
			// check clean working tree
			out, err := exec.Command("git", "-C", repoRoot, "status", "--porcelain").Output()
			if err == nil && strings.TrimSpace(string(out)) != "" {
				results = append(results, checkResult{Name: "working tree", Status: "WARN", Error: "dirty"})
				if !doctorJSON {
					fmt.Println("WARN working tree dirty")
				}
			} else if err == nil {
				results = append(results, checkResult{Name: "working tree", Status: "OK"})
				if !doctorJSON {
					fmt.Println("OK   working tree clean")
				}
			}
		}
		// derived DB freshness
		dbPath := filepath.Join(repoRoot, ".gitdb", "data.db")
		if info, err := os.Stat(dbPath); err == nil {
			stale := false
			root, _ := config.LoadRoot(repoRoot)
			if root != nil {
				for _, ref := range root.Collections {
					def, _ := config.LoadCollectionDefinition(repoRoot, root.DataDir, ref)
					if def == nil {
						continue
					}
					dir := def.DataDir
					if !filepath.IsAbs(dir) {
						dir = filepath.Join(repoRoot, dir)
					}
					filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
						if err != nil || fi.IsDir() {
							return nil
						}
						if fi.ModTime().After(info.ModTime()) {
							stale = true
						}
						return nil
					})
				}
			}
			if stale {
				results = append(results, checkResult{Name: "derived db", Status: "WARN", Error: "stale (run rebuild)"})
				if !doctorJSON {
					fmt.Println("WARN derived db stale (run rebuild)")
				}
			} else {
				results = append(results, checkResult{Name: "derived db", Status: "OK"})
				if !doctorJSON {
					fmt.Println("OK   derived db fresh")
				}
			}
		} else {
			results = append(results, checkResult{Name: "derived db", Status: "INFO", Error: "not built"})
			if !doctorJSON {
				fmt.Println("INFO derived db not built")
			}
		}
		if doctorJSON {
			b, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(b))
		}
		if !ok {
			return fmt.Errorf("doctor found issues")
		}
		return nil
	},
}
