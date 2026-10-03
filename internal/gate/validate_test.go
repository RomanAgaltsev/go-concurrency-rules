package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// validRule returns a rule that passes G6, with its code directory created
// under root.
func validRule(t *testing.T, root string) Rule {
	t.Helper()
	r := Rule{
		ID:        "R07",
		Title:     "A data race is a missing happens-before edge",
		Statement: "Fix a race by adding the missing edge.",
		Group:     "shared-memory",
		Tags:      []string{"read", "write"},
		Since:     "go1.0",
		Status:    "active",
		Proof: Proof{
			Kind:      "race",
			Test:      "TestConcurrentInc",
			Signature: []string{"WARNING: DATA RACE"},
			Runs:      20,
		},
		Sources: []string{"https://go.dev/ref/mem"},
		Slug:    "r07-race",
		Page:    "docs/rules/r07-race.md",
		Dir:     "rules/r07-race",
	}
	writeTwin(t, root, r.Dir, "//go:build broken", "//go:build !broken")
	return r
}

func writeTwin(t *testing.T, root, dir, brokenTag, fixedTag string) {
	t.Helper()
	d := filepath.Join(root, filepath.FromSlash(dir))
	if err := os.MkdirAll(d, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, tag := range map[string]string{"broken.go": brokenTag, "fixed.go": fixedTag} {
		src := tag + "\n\npackage r\n"
		if err := os.WriteFile(filepath.Join(d, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(t *testing.T, root string, r *Rule)
		want   string // substring of the one expected violation; "" = valid
	}{
		{name: "valid", mutate: func(*testing.T, string, *Rule) {}},
		{name: "bad id", mutate: func(_ *testing.T, _ string, r *Rule) { r.ID = "R7" }, want: "not R or X followed by two digits"},
		{name: "page does not match id", mutate: func(_ *testing.T, _ string, r *Rule) { r.Slug = "r08-race" }, want: "does not start with r07-"},
		{name: "empty title", mutate: func(_ *testing.T, _ string, r *Rule) { r.Title = " " }, want: "title is empty"},
		{name: "empty statement", mutate: func(_ *testing.T, _ string, r *Rule) { r.Statement = "" }, want: "statement is empty"},
		{name: "unknown group", mutate: func(_ *testing.T, _ string, r *Rule) { r.Group = "locks" }, want: `group "locks"`},
		{name: "no tags", mutate: func(_ *testing.T, _ string, r *Rule) { r.Tags = nil }, want: "tags is empty"},
		{name: "unknown tag", mutate: func(_ *testing.T, _ string, r *Rule) { r.Tags = []string{"read", "debug"} }, want: `tag "debug"`},
		{name: "duplicate tag", mutate: func(_ *testing.T, _ string, r *Rule) { r.Tags = []string{"read", "read"} }, want: "listed twice"},
		{name: "bad since", mutate: func(_ *testing.T, _ string, r *Rule) { r.Since = "1.22" }, want: "not a go1.N version"},
		{name: "unknown status", mutate: func(_ *testing.T, _ string, r *Rule) { r.Status = "draft" }, want: `status "draft"`},
		{name: "active with retired_in", mutate: func(_ *testing.T, _ string, r *Rule) { r.RetiredIn = "go1.23" }, want: "has no retired_in"},
		{name: "retired without replaced_by", mutate: func(_ *testing.T, _ string, r *Rule) {
			r.Status, r.RetiredIn = "retired", "go1.23"
		}, want: "replaced_by"},
		{name: "unknown proof kind", mutate: func(_ *testing.T, _ string, r *Rule) { r.Proof.Kind = "vibes" }, want: `proof.kind "vibes"`},
		{name: "bad test name", mutate: func(_ *testing.T, _ string, r *Rule) { r.Proof.Test = "concurrentInc" }, want: "not a test function name"},
		{name: "no signature", mutate: func(_ *testing.T, _ string, r *Rule) { r.Proof.Signature = nil }, want: "proof.signature is empty"},
		{name: "blank signature entry", mutate: func(_ *testing.T, _ string, r *Rule) {
			r.Proof.Signature = []string{"WARNING: DATA RACE", " "}
		}, want: "empty entry"},
		{name: "generic signature only", mutate: func(_ *testing.T, _ string, r *Rule) {
			r.Proof.Signature = []string{"--- FAIL:", "panic:"}
		}, want: "matches any failure at all"},
		{name: "generic marker beside a specific one", mutate: func(_ *testing.T, _ string, r *Rule) {
			r.Proof.Signature = []string{"--- FAIL: TestConcurrentInc", "WARNING: DATA RACE"}
		}},
		{name: "too few runs", mutate: func(_ *testing.T, _ string, r *Rule) { r.Proof.Runs = 3 }, want: "proof.runs is 3"},
		{name: "field of another kind", mutate: func(_ *testing.T, _ string, r *Rule) { r.Proof.Analyzer = "copylocks" }, want: "do not apply to kind race"},
		{name: "dangling alternative", mutate: func(_ *testing.T, _ string, r *Rule) { r.Alternatives = []string{"R99"} }, want: "alternative R99 does not exist"},
		{name: "self alternative", mutate: func(_ *testing.T, _ string, r *Rule) { r.Alternatives = []string{"R07"} }, want: "lists the rule itself"},
		{name: "no sources", mutate: func(_ *testing.T, _ string, r *Rule) { r.Sources = nil }, want: "sources is empty"},
		{name: "http source", mutate: func(_ *testing.T, _ string, r *Rule) { r.Sources = []string{"http://go.dev/ref/mem"} }, want: "not an https URL"},
		{name: "no code directory", mutate: func(_ *testing.T, _ string, r *Rule) { r.Dir = "rules/missing" }, want: "does not exist"},
		{name: "broken.go without tag", mutate: func(t *testing.T, root string, r *Rule) {
			writeTwin(t, root, r.Dir, "// no tag", "//go:build !broken")
		}, want: "broken.go: has no //go:build line"},
		{name: "fixed.go selected by both builds", mutate: func(t *testing.T, root string, r *Rule) {
			writeTwin(t, root, r.Dir, "//go:build broken", "//go:build linux || !linux")
		}, want: "fixed.go: build constraint"},
		{name: "retired twin with a go1.21 term", mutate: func(t *testing.T, root string, r *Rule) {
			writeTwin(t, root, r.Dir, "//go:build broken && go1.21", "//go:build !broken")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			r := validRule(t, root)
			tt.mutate(t, root, &r)

			vs := Validate(root, []Rule{r})
			if tt.want == "" {
				if len(vs) != 0 {
					t.Fatalf("want no violations, got %v", vs)
				}
				return
			}
			if len(vs) != 1 {
				t.Fatalf("want exactly one violation mentioning %q, got %d: %v", tt.want, len(vs), vs)
			}
			if vs[0].Gate != "G6" || !strings.Contains(vs[0].Msg, tt.want) {
				t.Fatalf("got %v, want a G6 violation mentioning %q", vs[0], tt.want)
			}
		})
	}
}

func TestValidateDuplicateID(t *testing.T) {
	root := t.TempDir()
	a := validRule(t, root)
	b := a
	b.Slug, b.Page = "r07-other", "docs/rules/r07-other.md"
	vs := Validate(root, []Rule{a, b})
	if len(vs) != 2 || !strings.Contains(vs[0].Msg, "used by 2 pages") {
		t.Fatalf("want both pages flagged as sharing R07, got %v", vs)
	}
}
