package afero

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestReadPastEndContract(t *testing.T) {
	for _, backend := range []string{"memory", "os"} {
		t.Run(backend, func(t *testing.T) {
			for _, scenario := range []string{
				"seek-read", "truncate-read", "second-handle-truncate-read", "truncate-copy",
				"readat-past-end", "readat-empty-past-end", "read-empty-past-end",
				"readat-at-end", "partial-readat", "partial-read", "read-in-range", "truncate-offset-write",
			} {
				t.Run(scenario, func(t *testing.T) {
					var fs Fs = NewMemMapFs()
					if backend == "os" {
						fs = NewOsFs()
					}
					name := filepath.Join(t.TempDir(), "file")
					f, err := fs.OpenFile(name, os.O_CREATE|os.O_RDWR, 0o600)
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(func() { _ = f.Close() })
					if n, err := f.Write([]byte("abc")); n != 3 || err != nil {
						t.Fatalf("Write = %d, %v", n, err)
					}
					var n int
					var wantN int
					var wantErr error = io.EOF
					buffer := make([]byte, 4)
					switch scenario {
					case "seek-read":
						if _, err := f.Seek(9, io.SeekStart); err != nil {
							t.Fatal(err)
						}
						n, err = f.Read(buffer)
					case "truncate-read", "truncate-copy":
						if err := f.Truncate(0); err != nil {
							t.Fatal(err)
						}
						if scenario == "truncate-copy" {
							var sink bytes.Buffer
							var copied int64
							copied, err = io.Copy(&sink, f)
							if copied != 0 || sink.Len() != 0 {
								t.Fatalf("copied %d bytes, sink=%q", copied, sink.Bytes())
							}
							wantErr = nil
						} else {
							n, err = f.Read(buffer)
						}
					case "second-handle-truncate-read":
						other, openErr := fs.OpenFile(name, os.O_RDWR, 0)
						if openErr != nil {
							t.Fatal(openErr)
						}
						t.Cleanup(func() { _ = other.Close() })
						if err := other.Truncate(0); err != nil {
							t.Fatal(err)
						}
						n, err = f.Read(buffer)
					case "readat-past-end":
						n, err = f.ReadAt(buffer, 9)
					case "readat-empty-past-end":
						n, err = f.ReadAt(nil, 9)
						wantErr = nil
					case "read-empty-past-end":
						if _, err := f.Seek(9, io.SeekStart); err != nil {
							t.Fatal(err)
						}
						n, err = f.Read(nil)
						wantErr = nil
					case "readat-at-end":
						n, err = f.ReadAt(buffer, 3)
					case "partial-readat":
						n, err = f.ReadAt(buffer, 1)
						wantN = 2
					case "partial-read":
						if _, err := f.Seek(1, io.SeekStart); err != nil {
							t.Fatal(err)
						}
						n, err = f.Read(buffer)
						wantN, wantErr = 2, nil
					case "read-in-range":
						if _, err := f.Seek(0, io.SeekStart); err != nil {
							t.Fatal(err)
						}
						n, err = f.Read(buffer[:2])
						wantN, wantErr = 2, nil
					case "truncate-offset-write":
						if err := f.Truncate(0); err != nil {
							t.Fatal(err)
						}
						if offset, err := f.Seek(0, io.SeekCurrent); offset != 3 || err != nil {
							t.Fatalf("offset after truncate = %d, %v; want 3, nil", offset, err)
						}
						if n, err := f.Write([]byte("z")); n != 1 || err != nil {
							t.Fatalf("write after truncate = %d, %v", n, err)
						}
						if n, err := f.ReadAt(buffer, 0); n != 4 || err != nil || !bytes.Equal(buffer, []byte{0, 0, 0, 'z'}) {
							t.Fatalf("write did not preserve sparse offset: %d, %v, %q", n, err, buffer)
						}
						return
					}
					if n != wantN || !errors.Is(err, wantErr) {
						t.Fatalf("%s: got (%d, %v), want (%d, %v)", scenario, n, err, wantN, wantErr)
					}
					if wantN > 0 {
						want := "bc"
						if scenario == "read-in-range" {
							want = "ab"
						}
						if string(buffer[:n]) != want {
							t.Fatalf("data = %q, want %q", buffer[:n], want)
						}
					}
				})
			}
		})
	}
}
