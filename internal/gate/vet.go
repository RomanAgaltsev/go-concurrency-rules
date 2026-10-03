package gate

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os/exec"
	"slices"
	"strings"
)

// vetDiag is one diagnostic in `go vet -json` output.
type vetDiag struct {
	Posn    string `json:"posn"`
	Message string `json:"message"`
}

// proveVet proves a vet rule: the fixed variant has no go vet diagnostics at
// all; the broken variant has at least one from the declared analyzer.
//
// Both directions are judged on the parsed JSON, never on the exit code:
// plain go vet names no analyzer, and go vet -json exits 0 even when it
// reports diagnostics (spec §15 P6).
func (p Prover) proveVet(ctx context.Context, r Rule) ([]Violation, string) {
	bad := func(format string, args ...any) []Violation {
		return []Violation{{Rule: r.ID, Gate: "G1", Msg: fmt.Sprintf(format, args...)}}
	}
	pkg := "./" + r.Dir
	an := r.Proof.Analyzer

	fixed, err := p.vet(ctx, pkg)
	if err != nil {
		return bad("fixed variant %v", err), ""
	}
	if len(fixed) > 0 {
		return bad("fixed variant has go vet diagnostics: %s", describeVet(fixed)), ""
	}

	broken, err := p.vet(ctx, "-tags", "broken", pkg)
	if err != nil {
		return bad("broken variant %v", err), ""
	}
	if len(broken[an]) == 0 {
		found := "nothing"
		if len(broken) > 0 {
			found = strings.Join(slices.Sorted(maps.Keys(broken)), ", ")
		}
		return bad("go vet -tags broken reported no %s diagnostic (it reported: %s)", an, found), ""
	}
	return nil, fmt.Sprintf("vet: fixed is clean; broken trips %s (%d diagnostic(s))", an, len(broken[an]))
}

// vet runs `go vet -json` and returns diagnostics by analyzer. A non-zero
// exit means vet could not analyse the package — it does not build — since
// diagnostics alone never make go vet -json fail.
func (p Prover) vet(ctx context.Context, args ...string) (map[string][]vetDiag, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout())
	defer cancel()
	//nolint:gosec // G204: the go command with arguments built from validated front matter
	cmd := exec.CommandContext(ctx, p.goBin(), append([]string{"vet", "-json"}, args...)...)
	cmd.Dir = p.Root
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("go vet did not finish: %w", ctx.Err())
		}
		return nil, fmt.Errorf("does not build (go vet -json: %w):\n%s", err, excerpt(stderr.Bytes()))
	}
	return parseVetJSON(stdout.Bytes())
}

// parseVetJSON reads the stream of objects go vet -json prints, one per
// package: {"pkg": {"analyzer": [diagnostics]}}. An analyzer that failed
// prints {"error": ...} in place of its list, which is an error here: an
// analyzer that did not run has not found the bug.
func parseVetJSON(data []byte) (map[string][]vetDiag, error) {
	diags := map[string][]vetDiag{}
	dec := json.NewDecoder(bytes.NewReader(data))
	for {
		var pkgs map[string]map[string]json.RawMessage
		if err := dec.Decode(&pkgs); errors.Is(err, io.EOF) {
			return diags, nil
		} else if err != nil {
			return nil, fmt.Errorf("cannot read go vet -json output: %w", err)
		}
		for _, analyzers := range pkgs {
			for name, raw := range analyzers {
				var ds []vetDiag
				if err := json.Unmarshal(raw, &ds); err != nil {
					return nil, fmt.Errorf("analyzer %s failed: %s", name, raw)
				}
				diags[name] = append(diags[name], ds...)
			}
		}
	}
}

// describeVet names each analyzer with its first diagnostic.
func describeVet(diags map[string][]vetDiag) string {
	var parts []string
	for _, name := range slices.Sorted(maps.Keys(diags)) {
		d := diags[name][0]
		parts = append(parts, fmt.Sprintf("%s: %s: %s", name, d.Posn, d.Message))
	}
	return strings.Join(parts, "; ")
}
