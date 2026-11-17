package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheck(t *testing.T) {
	bad, err := check([]string{filepath.Join("testdata", "bad")})
	if err != nil {
		t.Fatalf("check(bad) = %v", err)
	}
	fired := map[string]bool{}
	for _, f := range bad.findings {
		fired[f.rule] = true
	}
	for rule := range ruleDocs {
		if !fired[rule] {
			t.Errorf("check(testdata/bad) did not report rule %q", rule)
		}
	}

	good, err := check([]string{filepath.Join("testdata", "good")})
	if err != nil {
		t.Fatalf("check(good) = %v", err)
	}
	for _, f := range good.findings {
		t.Errorf("check(testdata/good) reported %s: %s %s", f.pos, f.rule, f.msg)
	}
}

func TestIgnoreDirectives(t *testing.T) {
	tests := []struct {
		name string
		src  string
		want int
	}{
		{
			name: "valid ignore with reason",
			src: `package widget
// sdkcheck:ignore constructor-stutter keeps legacy constructor name
func NewWidget() {}
`,
			want: 0,
		},
		{
			name: "missing reason does not suppress",
			src: `package widget
// sdkcheck:ignore constructor-stutter
func NewWidget() {}
`,
			want: 1,
		},
		{
			name: "unknown rule does not suppress",
			src: `package widget
// sdkcheck:ignore all reviewed
func NewWidget() {}
`,
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "x.go"), []byte(tt.src), 0o644); err != nil {
				t.Fatal(err)
			}
			c, err := check([]string{dir})
			if err != nil {
				t.Fatalf("check(%s) = %v", dir, err)
			}
			if got := len(c.findings); got != tt.want {
				t.Errorf("check() returned %d findings, want %d", got, tt.want)
			}
		})
	}
}
