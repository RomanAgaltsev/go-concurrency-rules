package gate

import (
	"os"
	"path/filepath"
	"testing"
)

// TestLoadRootWithGlobMeta: a repository can live under any directory name. A
// "[" in the root path is a character class to filepath.Glob, which then
// matches nothing — and the gate would report a repository full of rules as
// having none.
func TestLoadRootWithGlobMeta(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo[1]")
	page := filepath.Join(root, "docs", "rules", "r07-x.md")
	if err := os.MkdirAll(filepath.Dir(page), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(page, []byte("---\nid: R07\n---\n# T\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rules, unparsed, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) != 1 || rules[0].ID != "R07" || len(unparsed) != 0 {
		t.Fatalf("Load(%q) = %d rules %v, %v; want R07", root, len(rules), rules, unparsed)
	}
}
