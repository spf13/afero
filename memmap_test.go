package afero

import "testing"

func TestNormalizePath(t *testing.T) {
	type test struct {
		input    string
		expected string
	}

	data := []test{
		{".", "/"},
		{".", "/"},
		{"./", "/"},
		{"..", "/"},
		{"../", "/"},
		{"./..", "/"},
		{"./../", "/"},
	}

	for i, d := range data {
		cpath := normalizePath(d.input)
		if d.expected != cpath {
			t.Errorf("Test %d failed. Expected %t got %t", i, d.expected, cpath)
		}
	}
}
