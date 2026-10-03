//go:build broken

package r94

import (
	"os"
	"path/filepath"
)

// Answer is right in exactly one process per gate run: the first to create
// the marker. The gate test points FIXTURE_MARKER_DIR at a fresh directory.
// The marker lives here, not in the test, so that only broken processes can
// take it — the fixed run would otherwise claim it first.
func Answer() int {
	if dir := os.Getenv("FIXTURE_MARKER_DIR"); dir != "" {
		f, err := os.OpenFile(filepath.Join(dir, "passed"), os.O_CREATE|os.O_EXCL, 0o644)
		if err == nil {
			f.Close()
			return 42
		}
	}
	return 41
}
