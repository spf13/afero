package afero

import (
	"testing"
)

func TestFileContainsBytesReadBoundaries(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		needle  string
		want    bool
	}{
		{"zero padding is not file data", "text", "\x00", false},
		{"padding after final byte", "abc", "c\x00", false},
		{"stale tail is not file data", "ABCDEFGHIJKLMN", "NIJ", false},
		{"actual null byte", "abc\x00def", "\x00", true},
		{"short match", "abc", "bc", true},
		{"across read boundary", "abcdefghijklmnopqrst", "ijklmn", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := NewMemMapFs()
			if err := WriteFile(fs, "input", []byte(tc.content), 0o600); err != nil {
				t.Fatal(err)
			}
			got, err := FileContainsBytes(fs, "input", []byte(tc.needle))
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("FileContainsBytes = %v, want %v", got, tc.want)
			}
			got, err = FileContainsAnyBytes(fs, "input", [][]byte{[]byte(tc.needle)})
			if err != nil || got != tc.want {
				t.Fatalf("FileContainsAnyBytes = %v, %v, want %v", got, err, tc.want)
			}
		})
	}
}
