package gate

import (
	"strings"
	"testing"
)

func TestParsePage(t *testing.T) {
	tests := []struct {
		name     string
		page     string
		wantID   string
		wantBody string
		wantErr  string
	}{
		{
			name:     "valid",
			page:     "---\nid: R07\nproof:\n  kind: race\n  signature: [\"WARNING: DATA RACE\"]\n---\n# Body\n",
			wantID:   "R07",
			wantBody: "# Body\n",
		},
		{
			// A Windows checkout without .gitattributes would hand the gate CRLF.
			name:     "crlf",
			page:     "---\r\nid: R07\r\n---\r\n# Body\r\n",
			wantID:   "R07",
			wantBody: "# Body\n",
		},
		{
			// Some Windows editors save UTF-8 with a byte order mark. Zensical
			// strips it and reads the front matter, so the gate must too.
			name:     "byte order mark",
			page:     string(rune(0xFEFF)) + "---\nid: R07\n---\n# Body\n",
			wantID:   "R07",
			wantBody: "# Body\n",
		},
		{name: "no front matter", page: "# Body\n", wantErr: "does not start with"},
		{name: "unclosed", page: "---\nid: R07\n# Body\n", wantErr: "not closed"},
		{name: "empty", page: "---\n\n---\n# Body\n", wantErr: "empty"},
		{
			// The misspelling must not be dropped silently: proof.signature
			// would then validate as missing at best, and at worst a default
			// would pass for a declaration.
			name:    "unknown key",
			page:    "---\nid: R07\nproof:\n  sigature: [x]\n---\n",
			wantErr: "sigature",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body, err := ParsePage([]byte(tt.page))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to mention %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if fm.ID != tt.wantID {
				t.Errorf("ID = %q, want %q", fm.ID, tt.wantID)
			}
			if string(body) != tt.wantBody {
				t.Errorf("body = %q, want %q", body, tt.wantBody)
			}
		})
	}
}
