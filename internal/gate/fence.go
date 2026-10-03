package gate

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var (
	// fenceOpenRe matches a fence line: indentation, a run of ``` or ~~~, and
	// the info string. Content tabs indent their fences, so indentation is free.
	fenceOpenRe = regexp.MustCompile("^(\\s*)(`{3,}|~{3,})(.*)$")
	// snippetRe matches a pymdownx.snippets line: --8<-- "path" or "path:section".
	snippetRe = regexp.MustCompile(`^\s*--8<--\s+"([^"]+)"\s*$`)
	sectionRe = regexp.MustCompile(`^[A-Za-z][\w-]*$`)
)

// CheckPage enforces G2 on one page: every Go code block holds nothing but
// snippet lines, and every snippet line on the page resolves to a file under
// root — and, when it names a section, to a section that file declares.
//
// "Nothing on the site is hand-typed Go" is only true if the page cannot carry
// Go that the repository does not compile and test.
func CheckPage(root string, r Rule) []Violation {
	var vs []Violation
	bad := func(line int, format string, args ...any) {
		vs = append(vs, Violation{
			Rule: r.ID, Gate: "G2",
			Msg: fmt.Sprintf("%s:%d: %s", r.Page, line, fmt.Sprintf(format, args...)),
		})
	}

	lines := strings.Split(strings.ReplaceAll(string(r.Body), "\r\n", "\n"), "\n")
	var (
		fence     string // the open fence's run of ``` or ~~~; "" when outside a fence
		fenceLine int
		inGo      bool // the open block is a Go block
		reported  bool // this block's hand-typed Go was already reported
	)
	for i, line := range lines {
		n := i + 1 // 1-based, relative to the body
		if m := snippetRe.FindStringSubmatch(line); m != nil {
			if msg := resolveSnippet(root, m[1], fence != "" && inGo); msg != "" {
				bad(n, "%s", msg)
			}
			continue
		}
		if fence != "" {
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, fence[:1]) && strings.Trim(trimmed, fence[:1]) == "" && len(trimmed) >= len(fence) {
				fence = ""
				continue
			}
			if inGo && !reported && trimmed != "" {
				bad(n, "hand-typed Go in a code block; embed it from the repository with --8<--")
				reported = true // one report per block is enough
			}
			continue
		}
		if m := fenceOpenRe.FindStringSubmatch(line); m != nil {
			fence, fenceLine = m[2], n
			inGo, reported = fenceLang(m[3]) == "go", false
		}
	}
	if fence != "" {
		bad(fenceLine, "code block is never closed")
	}
	return vs
}

// fenceLang returns the language of a fence info string: ```go, ``` go,
// ```{.go}, ```go title="x", ```golang and ```Go all name Go — Pygments
// matches lexer names case-insensitively, so the gate must too.
func fenceLang(info string) string {
	f := strings.Fields(strings.NewReplacer("{", " ", "}", " ").Replace(info))
	if len(f) == 0 {
		return ""
	}
	lang := strings.ToLower(strings.TrimPrefix(f[0], "."))
	if lang == "golang" {
		return "go"
	}
	return lang
}

// resolveSnippet returns why a snippet reference does not resolve, or "".
// Inside a Go block it must name a .go file outside testdata/: the page's
// promise is that its Go is code the repository compiles and tests.
func resolveSnippet(root, ref string, inGo bool) string {
	file, section, hasSection := strings.Cut(ref, ":")
	if !filepath.IsLocal(filepath.FromSlash(file)) {
		return fmt.Sprintf("snippet %q: %s is not a path inside the repository", ref, file)
	}
	if inGo && (path.Ext(file) != ".go" || slices.Contains(strings.Split(file, "/"), "testdata")) {
		return fmt.Sprintf("snippet %q: a Go code block may only embed .go files outside testdata/", ref)
	}
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
	if err != nil {
		return fmt.Sprintf("snippet %q: file %s does not exist", ref, file)
	}
	if !hasSection {
		return ""
	}
	if !sectionRe.MatchString(section) {
		return fmt.Sprintf("snippet %q: %q is not a section name", ref, section)
	}
	for _, marker := range []string{"--8<-- [start:" + section + "]", "--8<-- [end:" + section + "]"} {
		if !bytes.Contains(data, []byte(marker)) {
			return fmt.Sprintf("snippet %q: %s has no %q marker", ref, file, marker)
		}
	}
	return ""
}
