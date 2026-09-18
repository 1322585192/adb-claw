package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNoDumpModelInSource(t *testing.T) {
	root := findRepoRoot(t)
	forbidden := []string{
		"Dump" + "UITree",
		"uiautomator" + " dump",
		"Parse" + "UITree",
		"FindBy" + "Index",
		"FindBy" + "Text",
		"FindBy" + "ID",
		"ADBClaw" + "Monitor",
		"pkg/" + "monitor",
	}
	roots := []string{
		filepath.Join(root, "src"),
		filepath.Join(root, "helper"),
		filepath.Join(root, "skills"),
	}
	var hits []string
	for _, dir := range roots {
		_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			base := filepath.Base(path)
			if base == "nodump_gate_test.go" {
				return nil
			}
			ext := filepath.Ext(path)
			switch ext {
			case ".go", ".java", ".md", ".json":
			default:
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(data)
			rel, _ := filepath.Rel(root, path)
			for _, token := range forbidden {
				if strings.Contains(text, token) {
					hits = append(hits, rel+": "+token)
				}
			}
			return nil
		})
	}
	if len(hits) > 0 {
		t.Fatalf("dump/text-tree leftovers:\n%s", strings.Join(hits, "\n"))
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "src", "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("repo root not found")
	return ""
}
