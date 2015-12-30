package afero

import (
    "testing"
)

func TestBasePath(t *testing.T) {
    baseFs := &MemMapFs{}
    baseFs.MkdirAll("/base/path/tmp", 0777)
    bp := NewBasePathFs(baseFs, "/base/path")

    if _, err := bp.Create("/tmp/foo"); err != nil {
        t.Errorf("Failed to set real path")
    }

    if fh, err := bp.Create("../tmp/bar"); err == nil {
        t.Errorf("succeeded in creating %s ...", fh.Name())
    }
}
