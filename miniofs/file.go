package minio

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"syscall"

	"github.com/minio/minio-go/v7"
)

// MinioFile is the Afero version adapted for Minio
type MinioFile struct {
	openFlags int
	fhOffset  int64 // File handle specific offset
	closed    bool
	resource  *minioFileResource
}

func NewMinioFile(ctx context.Context, fs *Fs, bucket *minio.BucketInfo, openFlags int,
	// Unused: there is no use to the file mode in GCloud just yet - but we keep it here, just in case we need it
	fileMode os.FileMode,
	name string,
) *MinioFile {
	return &MinioFile{
		openFlags: openFlags,
		fhOffset:  0,
		closed:    false,
		resource: &minioFileResource{
			ctx: ctx,
			fs:  fs,

			bucket:   bucket,
			name:     name,
			fileMode: fileMode,

			currentGcsSize: 0,

			offset: 0,
			reader: nil,
			writer: nil,
		},
	}
}

func NewMinioFileFromOldFH(openFlags int, fileMode os.FileMode, oldFile *minioFileResource) *MinioFile {
	res := &MinioFile{
		openFlags: openFlags,
		fhOffset:  0,
		closed:    false,
		resource:  oldFile,
	}
	res.resource.fileMode = fileMode

	return res
}

func (o *MinioFile) Close() error {
	if o.closed {
		// the afero spec expects the call to Close on a closed file to return an error
		return ErrFileClosed
	}
	o.closed = true
	return o.resource.Close()
}

func (o *MinioFile) Seek(newOffset int64, whence int) (int64, error) {
	if o.closed {
		return 0, ErrFileClosed
	}

	// Since this is an expensive operation; let's make sure we need it
	if (whence == 0 && newOffset == o.fhOffset) || (whence == 1 && newOffset == 0) {
		return o.fhOffset, nil
	}
	log.Printf("WARNING: Seek behavior triggered, highly inefficent. Offset before seek is at %d\n", o.fhOffset)

	// Fore the reader/writers to be reopened (at correct offset)
	err := o.Sync()
	if err != nil {
		return 0, err
	}
	stat, err := o.Stat()
	if err != nil {
		return 0, nil
	}

	switch whence {
	case 0:
		o.fhOffset = newOffset
	case 1:
		o.fhOffset += newOffset
	case 2:
		o.fhOffset = stat.Size() + newOffset
	}
	return o.fhOffset, nil
}

func (o *MinioFile) Read(p []byte) (n int, err error) {
	return o.ReadAt(p, o.fhOffset)
}

func (o *MinioFile) ReadAt(p []byte, off int64) (n int, err error) {
	if o.closed {
		return 0, ErrFileClosed
	}

	read, err := o.resource.ReadAt(p, off)
	o.fhOffset += int64(read)
	return read, err
}

func (o *MinioFile) Write(p []byte) (n int, err error) {
	return o.WriteAt(p, o.fhOffset)
}

func (o *MinioFile) WriteAt(b []byte, off int64) (n int, err error) {
	if o.closed {
		return 0, ErrFileClosed
	}

	if o.openFlags&os.O_RDONLY != 0 {
		return 0, fmt.Errorf("file is opend as read only")
	}

	written, err := o.resource.WriteAt(b, off)
	o.fhOffset += int64(written)
	return written, err
}

func (o *MinioFile) Name() string {
	return filepath.FromSlash(o.resource.name)
}

func (o *MinioFile) readdirImpl(count int) (res []*FileInfo, err error) {
	err = o.Sync()
	if err != nil {
		return nil, err
	}

	var ownInfo os.FileInfo
	ownInfo, err = o.Stat()
	if err != nil {
		return nil, err
	}

	if !ownInfo.IsDir() {
		return nil, syscall.ENOTDIR
	}

	//path := o.resource.fs.ensureTrailingSeparator(o.resource.name)

	opts := minio.ListObjectsOptions{
		Recursive: true,
		Prefix:    o.resource.name,
	}
	objs := o.resource.fs.client.ListObjects(o.resource.ctx, o.resource.bucket.Name, opts)
	for obj := range objs {
		if count > 0 {

		}
		tmp := newFileInfoFromAttrs(obj, o.resource.fileMode)
		if tmp.Name() == "" {
			// neither object.Name, not object.Prefix were present - so let's skip this unknown thing
			continue
		}
		res = append(res, tmp)
	}

	if count > 0 && len(res) > 0 {
		sort.Sort(ByName(res))
		res = res[:count]
	}

	return res, nil
}

func (o *MinioFile) Readdir(count int) ([]os.FileInfo, error) {
	fi, err := o.readdirImpl(count)
	if err != nil {
		return nil, err
	}

	var res []os.FileInfo
	for _, f := range fi {
		res = append(res, f)
	}
	return res, nil
}

func (o *MinioFile) Readdirnames(n int) ([]string, error) {
	fi, err := o.Readdir(n)
	if err != nil && err != io.EOF {
		return nil, err
	}
	names := make([]string, len(fi))

	for i, f := range fi {
		names[i] = f.Name()
	}
	return names, err
}

func (o *MinioFile) Stat() (os.FileInfo, error) {
	err := o.Sync()
	if err != nil {
		return nil, err
	}

	stat, err := o.resource.fs.client.StatObject(o.resource.ctx, o.resource.bucket.Name, o.resource.name, minio.StatObjectOptions{})
	if err != nil {
		return nil, err
	}

	return newFileInfoFromAttrs(stat, o.resource.fileMode), nil
}

func (o *MinioFile) Sync() error {
	return o.resource.maybeCloseIo()
}

func (o *MinioFile) Truncate(wantedSize int64) error {
	if o.closed {
		return ErrFileClosed
	}
	if o.openFlags == os.O_RDONLY {
		return fmt.Errorf("file was opened as read only")
	}
	return o.resource.Truncate(wantedSize)
}

func (o *MinioFile) WriteString(s string) (ret int, err error) {
	return o.Write([]byte(s))
}
