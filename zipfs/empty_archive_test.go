package zipfs

import (
	"archive/zip"
	"bytes"
	"os"
	"reflect"
	"testing"

	"github.com/spf13/afero"
)

func TestEmptyArchiveRoot(t *testing.T) {
	var data bytes.Buffer
	writer := zip.NewWriter(&data)
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	reader, err := zip.NewReader(bytes.NewReader(data.Bytes()), int64(data.Len()))
	if err != nil {
		t.Fatal(err)
	}
	fs := New(reader)
	for _, root := range []string{"/", "."} {
		t.Run(root, func(t *testing.T) {
			entries, err := afero.ReadDir(fs, root)
			if err != nil || len(entries) != 0 {
				t.Fatalf("ReadDir(%q) = %v, %v; want an empty directory", root, entries, err)
			}
			var visited []string
			err = afero.Walk(fs, root, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				visited = append(visited, path)
				return nil
			})
			if err != nil || !reflect.DeepEqual(visited, []string{root}) {
				t.Fatalf("Walk(%q) visited %v, error %v", root, visited, err)
			}
		})
	}
}
