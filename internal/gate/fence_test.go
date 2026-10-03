package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckPage(t *testing.T) {
	const code = "//go:build broken\n\npackage r\n\n// --8<-- [start:counter]\nvar n int\n// --8<-- [end:counter]\n"
	tests := []struct {
		name string
		body string
		want string // substring of the one expected violation; "" = clean
	}{
		{
			name: "embed in a Go fence inside a content tab",
			body: "=== \"Broken\"\n\n    ```go\n    --8<-- \"rules/r/broken.go:counter\"\n    ```\n",
		},
		{name: "whole-file embed", body: "```go\n--8<-- \"rules/r/broken.go\"\n```\n"},
		{name: "non-Go fences may hold anything", body: "```console\n$ go test -race\n```\n\n~~~text\nWARNING: DATA RACE\n~~~\n"},
		{name: "go.mod is not Go", body: "```gomod\nmodule x\n```\n"},
		{name: "hand-typed Go", body: "```go\nvar n int\n```\n", want: "hand-typed Go"},
		{name: "hand-typed Go after an embed", body: "```go\n--8<-- \"rules/r/broken.go:counter\"\nn++\n```\n", want: "hand-typed Go"},
		{name: "golang info string", body: "```golang\nvar n int\n```\n", want: "hand-typed Go"},
		{name: "attribute-style info string", body: "``` {.go title=\"x\"}\nvar n int\n```\n", want: "hand-typed Go"},
		{name: "tilde fence", body: "~~~go\nvar n int\n~~~\n", want: "hand-typed Go"},
		{name: "missing file", body: "```go\n--8<-- \"rules/r/nope.go:counter\"\n```\n", want: "does not exist"},
		// Pygments lowercases lexer names: ```Go is highlighted as Go.
		{name: "capitalised Go info string", body: "```Go\nvar n int\n```\n", want: "hand-typed Go"},
		// A Go block shows code the repository compiles and tests: a .go file,
		// outside testdata/ (which the go command never builds as part of ./...).
		{name: "Go block embedding a non-Go file", body: "```go\n--8<-- \"notes/x.txt\"\n```\n", want: "may only embed .go files outside testdata/"},
		{name: "Go block embedding testdata", body: "```go\n--8<-- \"rules/r/testdata/x.go\"\n```\n", want: "may only embed .go files outside testdata/"},
		{name: "non-Go block may embed any file", body: "```text\n--8<-- \"rules/r/broken.go\"\n```\n"},
		{name: "path escaping the repository", body: "```go\n--8<-- \"../outside.go\"\n```\n", want: "not a path inside the repository"},
		{name: "missing section", body: "```go\n--8<-- \"rules/r/broken.go:total\"\n```\n", want: `no "--8<-- [start:total]" marker`},
		{name: "unclosed fence", body: "```go\n--8<-- \"rules/r/broken.go:counter\"\n", want: "never closed"},
		{
			// A longer fence is closed only by a fence at least as long: the
			// inner ``` is content, so the Go inside is still caught.
			name: "inner short fence does not close an outer long one",
			body: "````go\n```\nvar n int\n````\n",
			want: "hand-typed Go",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.MkdirAll(filepath.Join(root, "rules", "r"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(root, "rules", "r", "broken.go"), []byte(code), 0o644); err != nil {
				t.Fatal(err)
			}
			r := Rule{ID: "R07", Page: "docs/rules/r07-x.md", Body: []byte(tt.body)}

			vs := CheckPage(root, r)
			if tt.want == "" {
				if len(vs) != 0 {
					t.Fatalf("want no violations, got %v", vs)
				}
				return
			}
			if len(vs) != 1 || vs[0].Gate != "G2" || !strings.Contains(vs[0].Msg, tt.want) {
				t.Fatalf("want one G2 violation mentioning %q, got %v", tt.want, vs)
			}
		})
	}
}
