package paths

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name       string
		configured []string
		defaults   []string
		want       []string
	}{
		{
			name:       "empty configured falls back to defaults",
			configured: nil,
			defaults:   []string{"/default/a", "/default/b"},
			want:       []string{"/default/a", "/default/b"},
		},
		{
			name:       "single configured root replaces defaults entirely",
			configured: []string{"/custom"},
			defaults:   []string{"/default/a", "/default/b"},
			want:       []string{"/custom"},
		},
		{
			name:       "multiple configured roots replace defaults entirely",
			configured: []string{"/a", "/b"},
			defaults:   []string{"/default"},
			want:       []string{"/a", "/b"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Resolve(tt.configured, tt.defaults...)
			if len(got) != len(tt.want) {
				t.Fatalf("Resolve() = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("Resolve() = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

func TestMissing(t *testing.T) {
	dir := t.TempDir()
	exists := filepath.Join(dir, "exists")
	if err := os.MkdirAll(exists, 0o750); err != nil {
		t.Fatal(err)
	}
	absent := filepath.Join(dir, "absent")

	got := Missing([]string{exists, absent})
	if len(got) != 1 || got[0] != absent {
		t.Fatalf("Missing() = %v, want [%s]", got, absent)
	}
}

func TestMissingNoneAbsent(t *testing.T) {
	dir := t.TempDir()
	if got := Missing([]string{dir}); len(got) != 0 {
		t.Fatalf("Missing() = %v, want none missing", got)
	}
}
