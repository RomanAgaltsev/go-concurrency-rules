//go:build broken && go1.21

package x91

// Capture returns what each of three closures sees of the loop variable. This
// file's language version is go1.21, so the loop shares one variable.
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
