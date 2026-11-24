package afero

import (
	"os"
	"testing"
)

func TestCopyOnWriteReadlinkLayers(t *testing.T) {
	base := NewBasePathFs(NewOsFs(), t.TempDir())
	if err := base.(Linker).SymlinkIfPossible("base-target", "link"); err != nil {
		t.Fatal(err)
	}
	baseTarget, err := base.(LinkReader).ReadlinkIfPossible("link")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		layer Fs
	}{
		{"disk overlay", NewBasePathFs(NewOsFs(), t.TempDir())},
		{"memory overlay", NewMemMapFs()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := NewCopyOnWriteFs(NewReadOnlyFs(base), tc.layer)
			reader := fs.(LinkReader)
			target, err := reader.ReadlinkIfPossible("link")
			if err != nil || target != baseTarget {
				t.Fatalf("base link: target = %q, error = %v", target, err)
			}
			if err := WriteFile(tc.layer, "link", []byte("overlay file"), 0o600); err != nil {
				t.Fatal(err)
			}
			if target, err := reader.ReadlinkIfPossible("link"); err == nil {
				t.Fatalf("regular overlay file must hide the base link, got %q", target)
			}
			if err := tc.layer.Remove("link"); err != nil {
				t.Fatal(err)
			}
			if linker, ok := tc.layer.(Linker); ok {
				if err := linker.SymlinkIfPossible("overlay-target", "link"); err != nil {
					t.Fatal(err)
				}
				overlayTarget, err := tc.layer.(LinkReader).ReadlinkIfPossible("link")
				if err != nil {
					t.Fatal(err)
				}
				target, err := reader.ReadlinkIfPossible("link")
				if err != nil || target != overlayTarget {
					t.Fatalf("overlay link: target = %q, error = %v", target, err)
				}
			}
			if _, err := reader.ReadlinkIfPossible("missing"); !os.IsNotExist(err) {
				t.Fatalf("missing entry: %v", err)
			}
		})
	}
}
