package afero

import "testing"

func TestNilBasePathFsStat(t *testing.T) {
	var b *BasePathFs
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panicked: %v", r)
		}
	}()
	_, err := b.Stat("x")
	if err == nil {
		t.Fatal("want error")
	}
}
