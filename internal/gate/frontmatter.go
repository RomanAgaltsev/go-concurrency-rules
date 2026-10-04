// Package gate is the admission gate for go-concurrency-rules. A rule merges
// only when its page's front matter is valid (G6), every Go snippet on its page
// is an embed that resolves (G2), and its proof holds in both directions (G1).
package gate

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"go.yaml.in/yaml/v3"
)

// FrontMatter is the YAML header of a rule page.
type FrontMatter struct {
	ID           string   `yaml:"id"`
	Title        string   `yaml:"title"`
	Statement    string   `yaml:"statement"`
	Group        string   `yaml:"group"`
	Tags         []string `yaml:"tags"`
	Since        string   `yaml:"since"`
	Status       string   `yaml:"status"`
	RetiredIn    string   `yaml:"retired_in"`
	ReplacedBy   string   `yaml:"replaced_by"`
	Proof        Proof    `yaml:"proof"`
	Alternatives []string `yaml:"alternatives"`
	Undergo      []string `yaml:"undergo"`
	Sources      []string `yaml:"sources"`
}

// Proof declares how a rule proves itself.
type Proof struct {
	// Kind selects the proof procedure: race or test.
	Kind string `yaml:"kind"`
	// Test is the one test function that is the proof, e.g. TestConcurrentInc.
	Test string `yaml:"test"`
	// Signature lists substrings that must ALL appear in the output of EVERY
	// failing run of the broken variant. The gate adds nothing of its own: a
	// panic in a goroutine the test started prints no "--- FAIL" line at all.
	Signature []string `yaml:"signature"`
	// Analyzer is the vet analyzer a vet-kind rule trips. Unused by race/test.
	Analyzer string `yaml:"analyzer"`
	// Runs is N: the broken variant must fail in N separate processes out of N.
	Runs int `yaml:"runs"`
	// Reason says why a retired entry carries no proof. Unused by race/test.
	Reason string `yaml:"reason"`
}

var (
	openDelim  = []byte("---\n")
	closeDelim = []byte("\n---\n")
)

// ParsePage splits a Markdown page into its front matter and its body.
//
// Unknown keys are an error. A misspelt key would otherwise be dropped in
// silence, and the field it was meant to set would then validate as empty —
// or, worse, as a valid default.
func ParsePage(page []byte) (FrontMatter, []byte, error) {
	// A UTF-8 byte order mark (some Windows editors write one) is not content:
	// Zensical strips it and reads the front matter behind it, so the gate does.
	page = bytes.TrimPrefix(page, []byte{0xEF, 0xBB, 0xBF})
	page = bytes.ReplaceAll(page, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(page, openDelim) {
		return FrontMatter{}, nil, errors.New("page does not start with a --- front matter block")
	}
	rest := page[len(openDelim):]
	end := bytes.Index(rest, closeDelim)
	if end < 0 {
		return FrontMatter{}, nil, errors.New("front matter block is not closed by a --- line")
	}

	var fm FrontMatter
	dec := yaml.NewDecoder(bytes.NewReader(rest[:end+1]))
	dec.KnownFields(true)
	if err := dec.Decode(&fm); err != nil {
		if errors.Is(err, io.EOF) {
			return FrontMatter{}, nil, errors.New("front matter block is empty")
		}
		return FrontMatter{}, nil, fmt.Errorf("front matter: %w", err)
	}
	return fm, rest[end+len(closeDelim):], nil
}
