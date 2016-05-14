package s3

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/spf13/afero"
)

// S3File represents a file in S3.
// It is not threadsafe.
type S3File struct {
	bucket string
	name   string
	s3Fs   afero.Fs
	s3API  s3iface.S3API

	// state
	offset int
	closed bool

	// readdir state
	readdirContinuationToken *string
	readdirNotTruncated      bool
}

var _ afero.File = (*S3File)(nil)

// NewS3File initializes an S3File object.
func NewS3File(bucket, name string, s3API s3iface.S3API, s3Fs afero.Fs) *S3File {
	return &S3File{
		bucket: bucket,
		name:   name,
		s3API:  s3API,
		s3Fs:   s3Fs,
		offset: 0,
		closed: false,
	}
}

// Name returns the filename, i.e. S3 path without the bucket name.
func (f *S3File) Name() string { return f.name }

// Readdir reads the contents of the directory associated with file and
// returns a slice of up to n FileInfo values, as would be returned
// by ListObjects, in directory order. Subsequent calls on the same file will yield further FileInfos.
//
// If n > 0, Readdir returns at most n FileInfo structures. In this case, if
// Readdir returns an empty slice, it will return a non-nil error
// explaining why. At the end of a directory, the error is io.EOF.
//
// If n <= 0, Readdir returns all the FileInfo from the directory in
// a single slice. In this case, if Readdir succeeds (reads all
// the way to the end of the directory), it returns the slice and a
// nil error. If it encounters an error before the end of the
// directory, Readdir returns the FileInfo read until that point
// and a non-nil error.
func (f *S3File) Readdir(n int) ([]os.FileInfo, error) {
	if f.readdirNotTruncated {
		return nil, io.EOF
	}
	if n <= 0 {
		return f.ReaddirAll()
	}
	// ListObjects treats leading slashes as part of the directory name
	// It also needs a trailing slash to list contents of a directory.
	name := trimLeadingSlash(f.Name()) + "/"
	output, err := f.s3API.ListObjectsV2(&s3.ListObjectsV2Input{
		ContinuationToken: f.readdirContinuationToken,
		Bucket:            aws.String(f.bucket),
		Prefix:            aws.String(name),
		Delimiter:         aws.String("/"),
		MaxKeys:           aws.Int64(int64(n)),
	})
	if err != nil {
		return nil, err
	}
	f.readdirContinuationToken = output.NextContinuationToken
	if !(*output.IsTruncated) {
		f.readdirNotTruncated = true
	}
	fis := []os.FileInfo{}
	for _, subfolder := range output.CommonPrefixes {
		fis = append(fis, NewS3FileInfo(filepath.Base("/"+*subfolder.Prefix), true, 0, time.Time{}))
	}
	for _, fileObject := range output.Contents {
		if hasTrailingSlash(*fileObject.Key) {
			// S3 includes <name>/ in the Contents listing for <name>
			continue
		}

		fis = append(fis, NewS3FileInfo(filepath.Base("/"+*fileObject.Key), false, *fileObject.Size, *fileObject.LastModified))
	}

	return fis, nil
}

func (f *S3File) ReaddirAll() ([]os.FileInfo, error) {
	fileInfos := []os.FileInfo{}
	for {
		infos, err := f.Readdir(100)
		fileInfos = append(fileInfos, infos...)
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return nil, err
			}
		}
	}
	return fileInfos, nil
}

// Readdirnames reads and returns a slice of names from the directory f.
//
// If n > 0, Readdirnames returns at most n names. In this case, if
// Readdirnames returns an empty slice, it will return a non-nil error
// explaining why. At the end of a directory, the error is io.EOF.
//
// If n <= 0, Readdirnames returns all the names from the directory in
// a single slice. In this case, if Readdirnames succeeds (reads all
// the way to the end of the directory), it returns the slice and a
// nil error. If it encounters an error before the end of the
// directory, Readdirnames returns the names read until that point and
// a non-nil error.
func (f *S3File) Readdirnames(n int) ([]string, error) {
	fi, err := f.Readdir(n)
	names := make([]string, len(fi))
	for i, f := range fi {
		_, names[i] = filepath.Split(f.Name())
	}
	return names, err
}

// Stat returns the FileInfo structure describing file.
// If there is an error, it will be of type *PathError.
func (f *S3File) Stat() (os.FileInfo, error) {
	return f.s3Fs.Stat(f.Name())
}

// Sync is a noop.
func (f *S3File) Sync() error {
	return nil
}

// Truncate changes the size of the file.
// It does not change the I/O offset.
// If there is an error, it will be of type *PathError.
func (f *S3File) Truncate(size int64) error {
	panic("implement Truncate")
	return nil
}

// WriteString is like Write, but writes the contents of string s rather than
// a slice of bytes.
func (f *S3File) WriteString(s string) (int, error) {
	return f.Write([]byte(s))
}

// Close closes the File, rendering it unusable for I/O.
// It returns an error, if any.
func (f *S3File) Close() error {
	f.closed = true
	return nil
}

// Read reads up to len(b) bytes from the File.
// It returns the number of bytes read and an error, if any.
// EOF is signaled by a zero count with err set to io.EOF.
func (f *S3File) Read(p []byte) (int, error) {
	if f.closed {
		// mimic os.File's read after close behavior
		panic("read after close")
	}
	if f.offset != 0 {
		panic("TODO: non-offset == 0 read")
	}
	if len(p) == 0 {
		return 0, nil
	}
	output, err := f.s3API.GetObject(&s3.GetObjectInput{
		Bucket: aws.String(f.bucket),
		Key:    aws.String(f.name),
	})
	if err != nil {
		return 0, err
	}
	defer output.Body.Close()
	n, err := output.Body.Read(p)
	f.offset += n
	return n, err
}

// ReadAt reads len(p) bytes from the file starting at byte offset off.
// It returns the number of bytes read and the error, if any.
// ReadAt always returns a non-nil error when n < len(b).
// At end of file, that error is io.EOF.
func (f *S3File) ReadAt(p []byte, off int64) (n int, err error) {
	_, err = f.Seek(off, 0)
	if err != nil {
		return
	}
	n, err = f.Read(p)
	return
}

// Seek sets the offset for the next Read or Write on file to offset, interpreted
// according to whence: 0 means relative to the origin of the file, 1 means
// relative to the current offset, and 2 means relative to the end.
// It returns the new offset and an error, if any.
// The behavior of Seek on a file opened with O_APPEND is not specified.
func (f *S3File) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case 0:
		f.offset = int(offset)
	case 1:
		f.offset += int(offset)
	case 2:
		// can probably do this if we had GetObjectOutput (ContentLength)
		panic("TODO: whence == 2 seek")
	}
	return int64(f.offset), nil
}

// Write writes len(b) bytes to the File.
// It returns the number of bytes written and an error, if any.
// Write returns a non-nil error when n != len(b).
func (f *S3File) Write(p []byte) (int, error) {
	if f.closed {
		// mimic os.File's write after close behavior
		panic("write after close")
	}
	if f.offset != 0 {
		panic("TODO: non-offset == 0 write")
	}
	readSeeker := bytes.NewReader(p)
	size := int(readSeeker.Size())
	if _, err := f.s3API.PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(f.bucket),
		Key:                  aws.String(f.name),
		Body:                 readSeeker,
		ServerSideEncryption: aws.String("AES256"),
	}); err != nil {
		return 0, err
	}
	f.offset += size
	return size, nil
}

// WriteAt writes len(p) bytes to the file starting at byte offset off.
// It returns the number of bytes written and an error, if any.
// WriteAt returns a non-nil error when n != len(p).
func (f *S3File) WriteAt(p []byte, off int64) (n int, err error) {
	_, err = f.Seek(off, 0)
	if err != nil {
		return
	}
	n, err = f.Write(p)
	return
}
