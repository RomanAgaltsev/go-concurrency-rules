//go:build !broken

package x01

// --8<-- [start:capture]

// Capture returns what each of three closures sees of the loop variable.
func Capture() []int {
	var fs []func() int
	for i := 0; i < 3; i++ { //nolint:modernize // the same loop as broken.go on purpose: the proof is one code, two language versions
		fs = append(fs, func() int { return i })
	}
	out := make([]int, 0, len(fs))
	for _, f := range fs {
		out = append(out, f())
	}
	return out
}

// --8<-- [end:capture]
