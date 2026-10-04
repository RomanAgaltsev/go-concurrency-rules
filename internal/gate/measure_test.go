package gate

import (
	"strings"
	"testing"
)

const goodMeasurement = `# rule: R01
# date: 2026-10-03
# go: go1.27.1
# os: linux/amd64
# cpu: AMD Ryzen 5 3600 6-Core Processor
# gomaxprocs: 1,4
# command: go test -run '^$' -bench '.' -benchmem -count 10 -cpu 1,4 [-tags broken] ./rules/r01-x; benchstat broken.txt fixed.txt

goos: linux
goarch: amd64
pkg: github.com/RomanAgaltsev/go-concurrency-rules/rules/r01-x
cpu: AMD Ryzen 5 3600 6-Core Processor
        │   broken.txt   │              fixed.txt              │
        │     sec/op     │   sec/op     vs base                │
Sum       24706.00n ± 4%   39.23n ± 2%  -99.84% (p=0.000 n=10)
Sum-4     38643.50n ± 2%   38.69n ± 4%  -99.90% (p=0.000 n=10)
geomean      30.90µ        38.95n       -99.87%

        │  broken.txt  │                 fixed.txt                 │
        │     B/op     │     B/op      vs base                     │
Sum       5.500Ki ± 0%   0.000Ki ± 0%  -100.00% (p=0.000 n=10)
geomean   5.500Ki                      ?                       ¹ ²
¹ summaries must be >0 to compute geomean
² ratios must be >0 to compute geomean
`

func TestCheckMeasurement(t *testing.T) {
	tests := []struct {
		name   string
		file   string // the artifact's file name; "" = the one measure.sh writes
		mutate func(string) string
		want   string // substring of the one expected problem; "" = clean
	}{
		// The file name is measure.sh's account of when and where: it must
		// agree with the header, or one of them was edited by hand.
		{
			name: "file dated another day", file: "2026-10-04-linux-12cpu.txt", mutate: func(s string) string { return s },
			want: "file name says 2026-10-04, header says 2026-10-03",
		},
		{
			name: "file names another os", file: "2026-10-03-darwin-12cpu.txt", mutate: func(s string) string { return s },
			want: "file name says darwin, header says linux/amd64",
		},
		{
			name: "file not named by measure.sh", file: "numbers.txt", mutate: func(s string) string { return s },
			want: "file name is not <date>-<goos>-<N>cpu.txt",
		},
		// A renamed or deleted benchmark leaves numbers nothing can reproduce.
		{name: "benchmark not in the package", mutate: func(s string) string {
			return strings.ReplaceAll(s, "\nSum", "\nTotal")
		}, want: "row Total: the package has no BenchmarkTotal"},
		{name: "sub-benchmark of a benchmark in the package", mutate: func(s string) string {
			return strings.Replace(s, "\nSum-4 ", "\nSum/n=10-4 ", 1)
		}},
		{name: "valid", mutate: func(s string) string { return s }},
		{name: "missing cpu", mutate: func(s string) string {
			return strings.Replace(s, "# cpu: AMD Ryzen 5 3600 6-Core Processor\n", "", 1)
		}, want: `header lacks "# cpu:"`},
		{name: "empty command", mutate: func(s string) string {
			lines := strings.Split(s, "\n")
			for i, l := range lines {
				if strings.HasPrefix(l, "# command:") {
					lines[i] = "# command: "
				}
			}
			return strings.Join(lines, "\n")
		}, want: `header lacks "# command:"`},
		{name: "another rule's numbers", mutate: func(s string) string {
			return strings.Replace(s, "# rule: R01", "# rule: R27", 1)
		}, want: "header says rule R27, but it belongs to R01"},
		// benchstat's own pkg: line is the witness the header cannot fake: an
		// artifact copied from another rule, header edited, still names the
		// package it measured.
		{name: "measured another package", mutate: func(s string) string {
			return strings.Replace(s, "/rules/r01-x\n", "/rules/r27-y\n", 1)
		}, want: `measured package "github.com/RomanAgaltsev/go-concurrency-rules/rules/r27-y", not rules/r01-x`},
		{name: "no pkg line", mutate: func(s string) string {
			return strings.Replace(s, "pkg: github.com/RomanAgaltsev/go-concurrency-rules/rules/r01-x\n", "", 1)
		}, want: "no pkg: line"},
		{name: "bad date", mutate: func(s string) string {
			return strings.Replace(s, "2026-10-03", "03.10.2026", 1)
		}, want: `header date "03.10.2026"`},
		{name: "raw benchmark output, not a comparison", mutate: func(s string) string {
			return strings.ReplaceAll(s, "vs base", "")
		}, want: "no benchstat comparison"},
		{name: "too few samples", mutate: func(s string) string {
			return strings.Replace(s, "n=10)\nSum-4", "n=6)\nSum-4", 1)
		}, want: "a comparison has n=6"},
		// benchstat prints n=A+B when the two sides have different counts: the
		// smaller side is what the comparison rests on.
		{name: "unequal sample counts", mutate: func(s string) string {
			return strings.Replace(s, "-99.90% (p=0.000 n=10)", "-99.90% (p=0.030 n=10+2)", 1)
		}, want: "a comparison has n=2"},
		// A benchmark measured on one side only prints no (p=… n=…) at all.
		{name: "benchmark on one side only", mutate: func(s string) string {
			return strings.Replace(s, "Sum-4 ", "OnlyBroken      505.5n ± 1%\nSum-4 ", 1)
		}, want: `row "OnlyBroken" has no comparison`},
		// "~" rows: benchstat found no significant difference. The gate
		// still requires the sample count; G3 review rejects quoting it.
		{name: "insignificant row still counted", mutate: func(s string) string {
			return strings.Replace(s, "-99.90% (p=0.000 n=10)", "~ (p=0.481 n=10)", 1)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := tt.file
			if file == "" {
				file = "2026-10-03-linux-12cpu.txt"
			}
			got := checkMeasurement("R01", "rules/r01-x", file, []byte(tt.mutate(goodMeasurement)), []string{"Sum"})
			if tt.want == "" {
				if len(got) != 0 {
					t.Fatalf("want no problems, got %v", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tt.want) {
				t.Fatalf("want one problem mentioning %q, got %v", tt.want, got)
			}
		})
	}
}
