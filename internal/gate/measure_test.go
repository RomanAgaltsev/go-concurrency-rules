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

        │   broken.txt   │              fixed.txt              │
        │     sec/op     │   sec/op     vs base                │
Sum       24706.00n ± 4%   39.23n ± 2%  -99.84% (p=0.000 n=10)
Sum-4     38643.50n ± 2%   38.69n ± 4%  -99.90% (p=0.000 n=10)
`

func TestCheckMeasurement(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(string) string
		want   string // substring of the one expected problem; "" = clean
	}{
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
		{name: "bad date", mutate: func(s string) string {
			return strings.Replace(s, "2026-10-03", "03.10.2026", 1)
		}, want: `header date "03.10.2026"`},
		{name: "raw benchmark output, not a comparison", mutate: func(s string) string {
			return strings.ReplaceAll(s, "vs base", "")
		}, want: "no benchstat comparison"},
		{name: "too few samples", mutate: func(s string) string {
			return strings.Replace(s, "n=10)\nSum-4", "n=6)\nSum-4", 1)
		}, want: "a comparison has n=6"},
		// "~" rows: benchstat found no significant difference. The gate
		// still requires the sample count; G3 review rejects quoting it.
		{name: "insignificant row still counted", mutate: func(s string) string {
			return strings.Replace(s, "-99.90% (p=0.000 n=10)", "~ (p=0.481 n=10)", 1)
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := checkMeasurement("R01", []byte(tt.mutate(goodMeasurement)))
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
