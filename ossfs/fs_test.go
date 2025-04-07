package ossfs

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/spf13/afero/ossfs/internal/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewOssFs(t *testing.T) {
	tests := []struct {
		name            string
		accessKeyId     string
		accessKeySecret string
		region          string
		bucket          string
		expected        *Fs
	}{
		{
			name:            "valid credentials",
			accessKeyId:     "testKeyId",
			accessKeySecret: "testKeySecret",
			region:          "test-region",
			bucket:          "test-bucket",
			expected: &Fs{
				bucketName:  "test-bucket",
				separator:   "/",
				autoSync:    true,
				openedFiles: make(map[string]afero.File),
				preloadFs:   afero.NewMemMapFs(),
				ctx:         context.Background(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewOssFs(tt.accessKeyId, tt.accessKeySecret, tt.region, tt.bucket)
			assert.NotNil(t, got.manager)
			assert.Equal(t, tt.expected.bucketName, got.bucketName)
			assert.Equal(t, tt.expected.separator, got.separator)
			assert.Equal(t, tt.expected.autoSync, got.autoSync)
			assert.NotNil(t, got.openedFiles)
			assert.NotNil(t, got.preloadFs)
			assert.NotNil(t, got.ctx)
		})
	}
}

func TestFsWithContext(t *testing.T) {
	tests := []struct {
		name     string
		fs       *Fs
		ctx      context.Context
		expected *Fs
	}{
		{
			name: "set new context",
			fs: &Fs{
				ctx: context.Background(),
			},
			ctx: context.WithValue(context.Background(), "testKey", "testValue"),
			expected: &Fs{
				ctx: context.WithValue(context.Background(), "testKey", "testValue"),
			},
		},
		{
			name: "set nil context",
			fs: &Fs{
				ctx: context.Background(),
			},
			ctx: nil,
			expected: &Fs{
				ctx: nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fs.WithContext(tt.ctx)
			assert.Equal(t, tt.expected.ctx, got.ctx)
			assert.Equal(t, tt.fs, got)
		})
	}
}

func TestFsCreate(t *testing.T) {
	m := &mocks.ObjectManager{}
	bucket := "test-bucket"
	ctx := context.TODO()

	fs := &Fs{
		manager:    m,
		bucketName: bucket,
		ctx:        ctx,
		separator:  "/",
	}

	t.Run("create simple success", func(t *testing.T) {
		m.On("PutObject", fs.ctx, bucket, "test.txt", strings.NewReader("")).Return(true, nil).Once()
		file, err := fs.Create("test.txt")
		assert.Nil(t, err)
		assert.NotNil(t, file)
		assert.Implements(t, (*afero.File)(nil), file)
		m.AssertExpectations(t)
	})

	t.Run("create prefixed file path success", func(t *testing.T) {
		m.
			On("PutObject", fs.ctx, bucket, "path/to/test.txt", strings.NewReader("")).
			Return(true, nil).
			Once()
		file, err := fs.Create("/path/to/test.txt")
		assert.Nil(t, err)
		assert.NotNil(t, file)
		assert.Implements(t, (*afero.File)(nil), file)
		m.AssertExpectations(t)
	})

	t.Run("create dir path success", func(t *testing.T) {
		m.
			On("PutObject", ctx, bucket, "path/to/test_dir/", strings.NewReader("")).
			Return(true, nil).
			Once()
		file, err := fs.Create("/path/to/test_dir/")
		assert.Nil(t, err)
		assert.NotNil(t, file)
		assert.Implements(t, (*afero.File)(nil), file)
		assert.Equal(t, "path/to/test_dir/", file.Name())
		m.AssertExpectations(t)
	})

	t.Run("create failure", func(t *testing.T) {
		m.On("PutObject", fs.ctx, bucket, "test2.txt", strings.NewReader("")).Return(false, afero.ErrFileNotFound).Once()
		_, err := fs.Create("test2.txt")
		assert.NotNil(t, err)
		assert.ErrorIs(t, err, afero.ErrFileNotFound)
		m.AssertExpectations(t)
	})
}
