package s3

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/spf13/afero"
)

// S3Fs is an FS object backed by S3.
type S3Fs struct {
	bucket string
	s3API  s3iface.S3API
}

var _ afero.Fs = (*S3Fs)(nil)

// NewS3Fs creates a new S3Fs object writing files to a given S3 bucket.
func NewS3Fs(bucket string, s3API s3iface.S3API) *S3Fs {
	return &S3Fs{bucket: bucket, s3API: s3API}
}

// Name returns the type of FS object this is: S3Fs.
func (S3Fs) Name() string { return "S3Fs" }

// Create a file.
func (fs S3Fs) Create(name string) (afero.File, error) {
	file, err := fs.Open(name)
	if err != nil {
		return file, err
	}
	// Create(), like all of S3, is eventually consistent.
	// To protect against unexpected behavior, have this method
	// wait until S3 reports the object exists.
	if s3Client, ok := fs.s3API.(*s3.S3); ok {
		return file, s3Client.WaitUntilObjectExists(&s3.HeadObjectInput{
			Bucket: aws.String(fs.bucket),
			Key:    aws.String(name),
		})
	}
	return file, err
}

// Mkdir makes a directory in S3.
func (fs S3Fs) Mkdir(name string, perm os.FileMode) error {
	_, err := fs.OpenFile(fmt.Sprintf("%s/", filepath.Clean(name)), os.O_CREATE, perm)
	return err
}

// MkdirAll creates a directory and all parent directories if necessary.
func (fs S3Fs) MkdirAll(path string, perm os.FileMode) error {
	return fs.Mkdir(path, perm)
}

// Open a file for reading.
// If the file doesn't exist, Open will create the file.
func (fs S3Fs) Open(name string) (afero.File, error) {
	if _, err := fs.Stat(name); err != nil {
		if os.IsNotExist(err) {
			return fs.OpenFile(name, os.O_CREATE, 0777)
		}
		return (*S3File)(nil), err
	}
	return NewS3File(fs.bucket, name, fs.s3API, fs), nil
}

// OpenFile opens a file.
func (fs S3Fs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	file := NewS3File(fs.bucket, name, fs.s3API, fs)
	if flag&os.O_APPEND != 0 {
		return file, errors.New("S3 is eventually consistent. Appending files will lead to trouble.")
	}
	if flag&os.O_CREATE != 0 {
		if _, err := file.WriteString(""); err != nil {
			return file, err
		}
	}
	return file, nil
}

// Remove a file.
func (fs S3Fs) Remove(name string) error {
	if _, err := fs.Stat(name); err != nil {
		return err
	}
	_, err := fs.s3API.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(name),
	})
	return err
}

// ForceRemove doesn't error if a file does not exist.
func (fs S3Fs) ForceRemove(name string) error {
	_, err := fs.s3API.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(name),
	})
	return err
}

// RemoveAll removes a path.
func (fs S3Fs) RemoveAll(path string) error {
	s3dir := NewS3File(fs.bucket, path, fs.s3API, fs)
	fis, err := s3dir.Readdir(0)
	if err != nil {
		return err
	}
	for _, fi := range fis {
		fullpath := filepath.Join(s3dir.Name(), fi.Name())
		if fi.IsDir() {
			if err := fs.RemoveAll(fullpath); err != nil {
				return err
			}
		} else {
			if err := fs.ForceRemove(fullpath); err != nil {
				return err
			}
		}
	}
	// finally remove the "file" representing the directory
	if err := fs.ForceRemove(s3dir.Name() + "/"); err != nil {
		return err
	}
	return nil
}

// Rename a file.
// There is no method to directly rename an S3 object, so the Rename
// will copy the file to an object with the new name and then delete
// the original.
func (fs S3Fs) Rename(oldname, newname string) error {
	if oldname == newname {
		return nil
	}
	_, err := fs.s3API.CopyObject(&s3.CopyObjectInput{
		Bucket:               aws.String(fs.bucket),
		CopySource:           aws.String(fs.bucket + oldname),
		Key:                  aws.String(newname),
		ServerSideEncryption: aws.String("AES256"),
	})
	if err != nil {
		return err
	}
	_, err = fs.s3API.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(oldname),
	})
	return err
}

func hasTrailingSlash(s string) bool {
	return len(s) > 0 && s[len(s)-1] == '/'
}

func trimLeadingSlash(s string) string {
	if len(s) > 0 && s[0] == '/' {
		return s[1:]
	}
	return s
}

// Stat returns a FileInfo describing the named file.
// If there is an error, it will be of type *os.PathError.
func (fs S3Fs) Stat(name string) (os.FileInfo, error) {
	nameClean := filepath.Clean(name)
	out, err := fs.s3API.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(fs.bucket),
		Key:    aws.String(nameClean),
	})
	if err != nil {
		if strings.Contains(err.Error(), "404") {
			statDir, err := fs.statDirectory(name)
			return statDir, err
		}
		return S3FileInfo{}, &os.PathError{
			Op:   "stat",
			Path: name,
			Err:  err,
		}
	} else if err == nil && hasTrailingSlash(name) {
		// user asked for a directory, but this is a file
		return S3FileInfo{}, &os.PathError{
			Op:   "stat",
			Path: name,
			//Err:  errors.New("not a directory"),
			Err: os.ErrNotExist,
		}
	}
	return NewS3FileInfo(filepath.Base(name), false, *out.ContentLength, *out.LastModified), nil
}

func (fs S3Fs) statDirectory(name string) (os.FileInfo, error) {
	nameClean := filepath.Clean(name)
	out, err := fs.s3API.ListObjectsV2(&s3.ListObjectsV2Input{
		Bucket:  aws.String(fs.bucket),
		Prefix:  aws.String(trimLeadingSlash(nameClean)),
		MaxKeys: aws.Int64(1),
	})
	if err != nil {
		return S3FileInfo{}, &os.PathError{
			Op:   "stat",
			Path: name,
			Err:  err,
		}
	}
	if *out.KeyCount == 0 && name != "" {
		return S3FileInfo{}, &os.PathError{
			Op:   "stat",
			Path: name,
			//Err:  errors.New("no such file or directory"),
			Err: os.ErrNotExist,
		}
	}
	return NewS3FileInfo(filepath.Base(name), true, 0, time.Time{}), nil
}

// Chmod is TODO
func (S3Fs) Chmod(name string, mode os.FileMode) error {
	panic("implement Chmod")
	return nil
}

// Chtimes is TODO
func (S3Fs) Chtimes(name string, atime time.Time, mtime time.Time) error {
	panic("implement Chtimes")
	return nil
}
