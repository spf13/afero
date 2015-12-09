package afero

import "testing"

func TestNormalizePath(t *testing.T) {
	type test struct {
		input    string
		expected string
	}

	data := []test{
		{".", FilePathSeparator},
		{"./", FilePathSeparator},
		{"..", FilePathSeparator},
		{"../", FilePathSeparator},
		{"./..", FilePathSeparator},
		{"./../", FilePathSeparator},
	}

	for i, d := range data {
		cpath := normalizePath(d.input)
		if d.expected != cpath {
			t.Errorf("Test %d failed. Expected %q got %q", i, d.expected, cpath)
		}
	}
}
