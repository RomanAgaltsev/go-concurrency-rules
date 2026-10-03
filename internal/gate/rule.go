package gate

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Rule is one rule page together with the code directory it describes.
type Rule struct {
	FrontMatter

	// Slug is the page's base name without .md, e.g. r07-race-is-a-missing-edge.
	// The code directory carries the same name.
	Slug string
	// Page and Dir are slash-separated and relative to the repository root.
	Page string
	Dir  string
	// Body is the page after its front matter.
	Body []byte
}

// Violation is one reason a rule is refused.
type Violation struct {
	// Rule is the rule's ID, or its page slug when the ID could not be read.
	Rule string
	// Gate is the gate that refused it: G1, G2 or G6.
	Gate string
	Msg  string
}

func (v Violation) String() string {
	return fmt.Sprintf("%s %s: %s", v.Rule, v.Gate, v.Msg)
}

// collection pairs a directory of pages with the directory of code they describe.
type collection struct{ docs, code string }

var collections = []collection{
	{docs: "docs/rules", code: "rules"},
	{docs: "docs/retired", code: "retired"},
}

// Load reads every rule page under root.
//
// A page that cannot be parsed is a G6 violation rather than an error, so that
// one broken page is reported alongside every other finding instead of hiding
// them.
func Load(root string) ([]Rule, []Violation, error) {
	var (
		rules []Rule
		vs    []Violation
	)
	for _, c := range collections {
		pages, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(c.docs), "*.md"))
		if err != nil {
			return nil, nil, err
		}
		for _, p := range pages {
			name := filepath.Base(p)
			if name == "index.md" {
				continue
			}
			slug := strings.TrimSuffix(name, ".md")
			rel := path.Join(c.docs, name)

			data, err := os.ReadFile(p)
			if err != nil {
				return nil, nil, err
			}
			fm, body, err := ParsePage(data)
			if err != nil {
				vs = append(vs, Violation{Rule: slug, Gate: "G6", Msg: fmt.Sprintf("%s: %v", rel, err)})
				continue
			}
			rules = append(rules, Rule{
				FrontMatter: fm,
				Slug:        slug,
				Page:        rel,
				Dir:         path.Join(c.code, slug),
				Body:        body,
			})
		}
	}
	return rules, vs, nil
}
