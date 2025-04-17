package afero

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGitHubFS(t *testing.T) {
	mockFs := NewMemMapFs()
	err := mockFs.Mkdir("testdir", 0755)
	if err != nil {
		t.Fatalf("Failed to setup mock directory: %v", err)
	}
	for _, filename := range []string{"README.md", "LICENSE"} {
		f, err := mockFs.Create(filename)
		if err != nil {
			t.Fatalf("Failed to create mock file %s: %v", filename, err)
		}
		_, err = f.Write([]byte("mock content"))
		if err != nil {
			t.Fatalf("Failed to write to mock file %s: %v", filename, err)
		}
		f.Close()
	}

	fs := NewGitHubFSWithBase(mockFs)

	tests := []struct {
		name      string
		path      string
		wantFile  bool
		wantDir   bool
		wantNames []string
		wantErr   bool
	}{
		{
			name:      "Root directory",
			path:      "",
			wantDir:   true,
			wantNames: []string{"README.md", "LICENSE", "testdir"},
			wantErr:   false,
		},
		{
			name:     "README file",
			path:     "README.md",
			wantFile: true,
			wantErr:  false,
		},
		{
			name:    "Non-existent file",
			path:    "nonexistent.txt",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := fs.Open(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, file)
				return
			}
			if err != nil {
				t.Fatalf("Failed to open %s: %v", tt.path, err)
			}
			if file == nil {
				t.Fatal("Open returned nil file handle without an error")
			}

			info, err := file.Stat()
			assert.NoError(t, err)
			if tt.wantFile {
				assert.False(t, info.IsDir())
				content, err := io.ReadAll(file)
				assert.NoError(t, err)
				assert.NotEmpty(t, content)
			}
			if tt.wantDir {
				assert.True(t, info.IsDir())
				names, err := file.Readdirnames(0)
				assert.NoError(t, err)
				for _, wantName := range tt.wantNames {
					assert.Contains(t, names, wantName, "Directory should contain %s", wantName)
				}
			}
			err = file.Close()
			assert.NoError(t, err)
		})
	}

	t.Run("Write operations fail", func(t *testing.T) {
		_, err := fs.Create("test.txt")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "read-only")

		err = fs.Mkdir("testdir", 0755)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "read-only")
	})
}
