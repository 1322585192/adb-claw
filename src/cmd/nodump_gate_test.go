package cmd

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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

func TestAppProfilesUseImageOnlyAgentCommands(t *testing.T) {
	root := findRepoRoot(t)
	profiles, err := filepath.Glob(filepath.Join(root, "skills", "apps", "*.md"))
	if err != nil {
		t.Fatal(err)
	}
	forbidden := map[string]*regexp.Regexp{
		"raw pixel tap":       regexp.MustCompile(`(?m)^\s*adb-claw\s+tap\s+\d+\s+\d+`),
		"removed locator":     regexp.MustCompile(`adb-claw[^\n]*--(?:text|id|index|desc)\b`),
		"host Python":         regexp.MustCompile(`(?m)^\s*(?:keyword=.*)?python3\b|json\.load\s*\(`),
		"UI dump command":     regexp.MustCompile(`(?mi)^\s*(?:adb-claw\s+shell\s+)?["']?uiautomator\b`),
		"IME mutation":        regexp.MustCompile(`(?mi)^\s*(?:adb-claw\s+shell\s+)?["']?ime\s+(?:set|enable|disable|reset)\b`),
		"clipboard service":   regexp.MustCompile(`(?mi)^\s*(?:adb-claw\s+shell\s+)?["']?service\s+call\s+clipboard\b`),
		"fixed sleep":         regexp.MustCompile(`(?mi)^\s*sleep\s+[0-9]`),
		"duplicate wait/read": regexp.MustCompile(`(?m)adb-claw wait --changed[^\n]*\nadb-claw observe`),
	}

	for _, path := range profiles {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		text := string(data)
		for name, pattern := range forbidden {
			if match := pattern.FindString(text); match != "" {
				rel, _ := filepath.Rel(root, path)
				t.Errorf("%s contains %s: %q", rel, name, match)
			}
		}
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
