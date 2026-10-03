//go:build broken && go1.21

package x01

// --8<-- [start:capture]

// Capture returns what each of three closures sees of the loop variable.
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

// --8<-- [end:capture]
