package gate

import (
	"strings"
	"testing"
)

func TestPosnFile(t *testing.T) {
	for in, want := range map[string]string{
		"/src/rules/r12-x/broken.go:25:9":           "broken.go",
		`C:\Users\a\rules\r12-x\broken.go:25:9`:     "broken.go",
		"rules/r12-x/extra.go:9:19":                 "extra.go",
		"broken.go:1:1":                             "broken.go",
		`E:\GIT\go-concurrency-rules\fixed.go:24:9`: "fixed.go",
	} {
		if got := posnFile(in); got != want {
			t.Errorf("posnFile(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseVetJSON(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    map[string]int // analyzer → diagnostic count
		wantErr string
	}{
		// What go1.27.1 prints for a clean package.
		{name: "clean", in: "{}\n", want: map[string]int{}},
		{name: "no output at all", in: "", want: map[string]int{}},
		{
			name: "one analyzer",
			in:   `{"p": {"copylocks": [{"posn": "a.go:1:2", "message": "x"}, {"posn": "a.go:3:4", "message": "y"}]}}`,
			want: map[string]int{"copylocks": 2},
		},
		{
			// One object per package; ./... prints several.
			name: "a stream of packages",
			in:   `{"p": {"copylocks": [{"posn": "a.go:1:2", "message": "x"}]}}` + "\n" + `{"q": {"printf": [{"posn": "b.go:1:2", "message": "y"}]}}`,
			want: map[string]int{"copylocks": 1, "printf": 1},
		},
		{
			// An analyzer that failed has not found the bug.
			name:    "analyzer error",
			in:      `{"p": {"copylocks": {"error": "boom"}}}`,
			wantErr: "analyzer copylocks failed",
		},
		{name: "not JSON", in: "vet: a.go:1:2: undefined: x\n", wantErr: "cannot read go vet -json output"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseVetJSON([]byte(tt.in))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to mention %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d analyzers %v, want %v", len(got), got, tt.want)
			}
			for an, n := range tt.want {
				if len(got[an]) != n {
					t.Errorf("%s: %d diagnostics, want %d", an, len(got[an]), n)
				}
			}
		})
	}
}
