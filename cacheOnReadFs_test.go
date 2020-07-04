package afero

import "testing"

func TestCacheOnRead_CreateShouldSyncDirectories(t *testing.T) {
	base := &MemMapFs{}
	layer := &MemMapFs{}

	ufs := NewCacheOnReadFs(base, layer, 0)

	base.Mkdir("/data", 0777)

	_, err := ufs.Create("/data/file.txt")
	if err != nil {
		t.Error("Cache should create intermediate directories if base.Create succeeds. Failed:", err)
	}
}
