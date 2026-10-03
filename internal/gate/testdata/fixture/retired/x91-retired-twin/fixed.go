//go:build !broken

package x91

// Capture returns what each of three closures sees of the loop variable. This
// file's language version is the module's, so each iteration has its own.
func Capture() []int {
	var fs []func() int
	for i := 0; i < 3; i++ {
		fs = append(fs, func() int { return i })
	}
	out := make([]int, 0, len(fs))
	for _, f := range fs {
		out = append(out, f())
	}
	return out
}
