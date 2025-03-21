package afero

import (
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnionFileReaddir(t *testing.T) {
	// Setup
	baseFs := NewMemMapFs()
	overlayFs := NewMemMapFs()

	assert.NoError(t, baseFs.Mkdir("testdir", 0755))
	assert.NoError(t, overlayFs.Mkdir("testdir", 0755))

	_, err := baseFs.Create("testdir/base1.txt")
	assert.NoError(t, err)
	_, err = baseFs.Create("testdir/base2.txt")
	assert.NoError(t, err)
	_, err = overlayFs.Create("testdir/base1.txt")
	assert.NoError(t, err)
	_, err = overlayFs.Create("testdir/over1.txt")
	assert.NoError(t, err)

	ufs := NewUnionFs(baseFs, overlayFs)
	dir, err := ufs.Open("testdir")
	assert.NoError(t, err)
	defer dir.Close()

	tests := []struct {
		name      string
		count     int
		wantCount int
		wantNames []string
		wantErr   error
	}{
		{
			name:      "Read all with -1",
			count:     -1,
			wantCount: 3,
			wantNames: []string{"base1.txt", "base2.txt", "over1.txt"},
			wantErr:   nil,
		},
		{
			name:      "Read all with 0",
			count:     0,
			wantCount: 3,
			wantNames: []string{"base1.txt", "base2.txt", "over1.txt"},
			wantErr:   nil,
		},
		{
			name:      "Read 1 entry",
			count:     1,
			wantCount: 1,
			wantNames: []string{"base1.txt"},
			wantErr:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uf := dir.(*UnionFile)
			uf.off = 0
			uf.files = nil // Reset for each test

			infos, err := dir.Readdir(tt.count)
			if tt.wantErr != nil {
				assert.Equal(t, tt.wantErr, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantCount, len(infos))

				var gotNames []string
				for _, info := range infos {
					gotNames = append(gotNames, info.Name())
				}
				assert.ElementsMatch(t, tt.wantNames, gotNames)
			}
		})
	}
}
