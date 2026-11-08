package conf

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

type goldenFixture struct {
	Checkout struct {
		Timeout    Getter[time.Duration] `conf:"name=ttl,format=scalar"`
		MaxRetries Getter[int]           `conf:"name=max_retry,format=scalar"`
	} `conf:"folder=checkouts"`
	KillSwitch Getter[bool] `conf:"name=toggle_switch,format=scalar"`
}

type simpleFixture struct {
	MaxRetries Getter[int] `conf:"format=scalar"`
}

type initialismFixture struct {
	DBHost      Getter[string]        `conf:"format=scalar"`
	HTTPTimeout Getter[time.Duration] `conf:"format=scalar"`
}

type fileFixture struct {
	A Getter[int]
}

type fileFormatTagFixture struct {
	A Getter[int] `conf:"format=scalar"`
}

type skipStructFixture struct {
	A Getter[int] `conf:"format=scalar"`
	B struct {
		C Getter[int] `conf:"format=scalar"`
	} `conf:"-"`
}

type ignoredFieldFixture struct {
	A    Getter[int] `conf:"format=scalar"`
	Note string
}

type missingFormatFixture struct {
	A Getter[int]
}

type invalidFormatFixture struct {
	A Getter[int] `conf:"format=xml"`
}

type skipGetterFixture struct {
	A Getter[int] `conf:"-"`
}

type nameOnStructFixture struct {
	A struct {
		B Getter[int] `conf:"format=scalar"`
	} `conf:"name=a"`
}

type folderOnGetterFixture struct {
	A Getter[int] `conf:"folder=a,format=scalar"`
}

type formatOnStructFixture struct {
	A struct {
		B Getter[int] `conf:"format=scalar"`
	} `conf:"format=json"`
}

type unknownTagFixture struct {
	A Getter[int] `conf:"env=A,format=scalar"`
}

type duplicateTagFixture struct {
	A Getter[int] `conf:"name=a,name=b,format=scalar"`
}

type badSegmentFixture struct {
	A Getter[int] `conf:"name=Bad,format=scalar"`
}

type duplicateKeyFixture struct {
	A Getter[int] `conf:"name=x,format=scalar"`
	B Getter[int] `conf:"name=x,format=scalar"`
}

type mapFolderFixture struct {
	A map[string]string `conf:"folder=a"`
}

type unexportedFixture struct {
	a Getter[int] `conf:"format=scalar"`
}

func TestDerive(t *testing.T) {
	_ = unexportedFixture{}.a
	tests := []struct {
		name       string
		value      any
		ownsFormat bool
		want       []string
		wantErr    error
	}{
		{
			name:  "snake case",
			value: simpleFixture{},
			want:  []string{"max_retries"},
		},
		{
			name:  "initialisms",
			value: initialismFixture{},
			want:  []string{"db_host", "http_timeout"},
		},
		{
			name:       "file mode",
			value:      fileFixture{},
			ownsFormat: true,
			want:       []string{"a"},
		},
		{
			name:       "file mode rejects format tag",
			value:      fileFormatTagFixture{},
			ownsFormat: true,
			wantErr:    ErrInvalidTag,
		},
		{
			name:  "skip subtree",
			value: skipStructFixture{},
			want:  []string{"a"},
		},
		{
			name:  "ignored non-config field",
			value: ignoredFieldFixture{},
			want:  []string{"a"},
		},
		{
			name:    "missing format",
			value:   missingFormatFixture{},
			wantErr: ErrMissingFormat,
		},
		{
			name:    "invalid format",
			value:   invalidFormatFixture{},
			wantErr: ErrInvalidFormat,
		},
		{
			name:    "skip getter",
			value:   skipGetterFixture{},
			wantErr: ErrInvalidTag,
		},
		{
			name:    "name on struct",
			value:   nameOnStructFixture{},
			wantErr: ErrInvalidTag,
		},
		{
			name:    "folder on getter",
			value:   folderOnGetterFixture{},
			wantErr: ErrInvalidTag,
		},
		{
			name:    "format on struct",
			value:   formatOnStructFixture{},
			wantErr: ErrInvalidTag,
		},
		{
			name:    "unknown tag",
			value:   unknownTagFixture{},
			wantErr: ErrInvalidTag,
		},
		{
			name:    "duplicate tag",
			value:   duplicateTagFixture{},
			wantErr: ErrInvalidTag,
		},
		{
			name:    "bad segment",
			value:   badSegmentFixture{},
			wantErr: ErrInvalidTag,
		},
		{
			name:    "duplicate key",
			value:   duplicateKeyFixture{},
			wantErr: ErrDuplicateKey,
		},
		{
			name:    "map folder",
			value:   mapFolderFixture{},
			wantErr: ErrInvalidType,
		},
		{
			name:    "unexported getter",
			value:   unexportedFixture{},
			wantErr: ErrInvalidType,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			leaves, err := derive(reflect.TypeOf(tt.value), tt.ownsFormat)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("derive(%T) error = %v, want %v", tt.value, err, tt.wantErr)
			}
			if tt.wantErr != nil {
				return
			}
			got := make([]string, len(leaves))
			for i, l := range leaves {
				got[i] = l.key
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("derive(%T) keys = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestDeriveFormats(t *testing.T) {
	leaves, err := derive(reflect.TypeFor[goldenFixture](), false)
	if err != nil {
		t.Fatalf("derive(goldenFixture) error = %v", err)
	}
	want := map[string]Format{
		"checkouts/ttl":       FormatScalar,
		"checkouts/max_retry": FormatScalar,
		"toggle_switch":       FormatScalar,
	}
	for _, l := range leaves {
		if l.format != want[l.key] {
			t.Errorf("derive(goldenFixture) key %q format = %q, want %q", l.key, l.format, want[l.key])
		}
	}

	leaves, err = derive(reflect.TypeFor[fileFixture](), true)
	if err != nil {
		t.Fatalf("derive(fileFixture) error = %v", err)
	}
	for _, l := range leaves {
		if l.format != "" {
			t.Errorf("derive(fileFixture) key %q format = %q, want empty (provider owns parsing)", l.key, l.format)
		}
	}
}

func TestSnake(t *testing.T) {
	tests := map[string]string{
		"Checkout":    "checkout",
		"MaxRetries":  "max_retries",
		"DBHost":      "db_host",
		"HTTPTimeout": "http_timeout",
		"KillSwitch":  "kill_switch",
		"ID":          "id",
	}
	for in, want := range tests {
		if got := snake(in); got != want {
			t.Errorf("snake(%q) = %q, want %q", in, got, want)
		}
	}
}

var updateGolden = flag.Bool("update", false, "update golden files")

func TestGoldenKeys(t *testing.T) {
	leaves, err := derive(reflect.TypeFor[goldenFixture](), false)
	if err != nil {
		t.Fatalf("derive(goldenFixture) error = %v", err)
	}
	keys := make([]string, len(leaves))
	for i, l := range leaves {
		keys[i] = l.key
	}
	got := strings.Join(keys, "\n") + "\n"

	path := filepath.Join("testdata", "keys.golden")
	if *updateGolden {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatalf("MkdirAll(testdata): %v", err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("WriteFile(%s): %v", path, err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s): %v", path, err)
	}
	if string(want) != got {
		t.Errorf("golden keys (-want +got):\n%s", got)
	}
}
