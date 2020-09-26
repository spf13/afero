package swift

import (
	"bytes"
	"fmt"
	"github.com/ncw/swift"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type File struct {
	objectCreateFile *swift.ObjectCreateFile
	objectOpenFile   *swift.ObjectOpenFile
	tmpContent       []byte
	fileInfo         *FileInfo

	seekStart int64

	conn          *swift.Connection // The connection to be able to do things like updating the file. This is probably not the best way to do it.
	containerName string
}

func (file *File) Close() error {
	if file.objectCreateFile != nil {
		return file.objectCreateFile.Close()
	}

	if file.objectOpenFile != nil {
		return file.objectOpenFile.Close()
	}

	log.Println("nothing to close")
	return nil
}

func (file *File) Read(p []byte) (n int, err error) {
	return file.ReadAt(p, file.seekStart)
}

func (file *File) ReadAt(p []byte, off int64) (n int, err error) {
	if file.objectOpenFile != nil {
		return file.objectOpenFile.Read(p)
	}

	if file.tmpContent != nil {
		// If we have an offset > 0, we just strip the first offset n bytes from the
		// input and put that into p
		if off > 0 {
			file.tmpContent = file.tmpContent[off:]
		}

		if len(file.tmpContent) > len(p) {
			return 0, io.EOF
		}
		return copy(p, file.tmpContent), nil
	}

	return 0, fmt.Errorf("nothing to read")
}

func (file *File) Seek(offset int64, whence int) (int64, error) {
	if file.tmpContent == nil {
		return 0, fmt.Errorf("no open file")
	}

	switch whence {
	case io.SeekStart:
		file.seekStart = offset
	case io.SeekCurrent:
		file.seekStart += offset
	case io.SeekEnd:
		file.seekStart = int64(len(file.tmpContent)) - offset
	default:
		return 0, fmt.Errorf("unkown whence")
	}

	return file.seekStart, nil
}

func (file *File) Write(p []byte) (n int, err error) {
	return file.WriteAt(p, file.seekStart)
}

func (file *File) WriteAt(p []byte, off int64) (n int, err error) {
	if file.objectCreateFile != nil {
		return file.objectCreateFile.Write(p)
	}

	// If offset is > 0, we need to modify our p slightly and add the len(off) bytes to p before it, since it is not
	// possible to write to a swift file with an offset
	if off > 0 {
		tmp := make([]byte, off)
		_, err := file.Read(tmp)
		if err != nil {
			return 0, err
		}
		p = append(tmp, p...)
	}

	// If we have an open file, write to it.
	if file.objectOpenFile != nil {
		_, err = file.conn.ObjectPut(file.containerName, file.Name(), bytes.NewReader(p), true, "", "", swift.Headers{})
		return int(len(p)), err // Apparently swift does not return the length it wrote. So we just return the size to not break anything.
	}

	// If we're at this point, that means no other file type thing (like objectCreateFile) exists.
	// This usually happens when we want to get a file and then get its content.
	// In that case put that content in here temporarily so we can access it later.
	file.tmpContent = p
	return len(p), nil
}

func (file *File) Name() string {
	return file.fileInfo.Name()
}

func (file *File) Readdir(count int) (res []os.FileInfo, err error) {
	if !file.fileInfo.IsDir() {
		return nil, fmt.Errorf("%s is not a dir", file.fileInfo.Name())
	}

	files, err := file.conn.ObjectsAll(file.containerName, &swift.ObjectsOpts{
		Limit:   count,
		Headers: map[string]string{"Content-Type": "application/directory"},
	})
	if err != nil {
		return nil, err
	}

	// This could probably be optimized slightly to consume less memory
	res = make([]os.FileInfo, count)
	for i, f := range files {
		// Sometimes filtering directly in the request does not work, so we need to manually filder here
		if f.ContentType != "application/directory" {
			continue
		}
		res[i] = FileInfo{
			name:    f.Name,
			size:    f.Bytes,
			modTime: f.LastModified,
			mode:    os.ModeDir, // At this point we can be sure this is always a folder
		}
	}

	return res, nil
}

func (file *File) Readdirnames(n int) (names []string, err error) {
	infs, err := file.Readdir(n)
	if err != nil {
		return
	}

	names = make([]string, len(infs))
	for i, f := range infs {
		_, names[i] = filepath.Split(f.Name())
	}

	return
}

func (file *File) Stat() (os.FileInfo, error) {
	return file.fileInfo, nil
}

func (file *File) Sync() error {
	return nil
}

func (file *File) Truncate(size int64) error {
	_, err := file.conn.ObjectPut(file.containerName, file.Name(), bytes.NewReader([]byte{}), true, "", "", swift.Headers{})
	return err
}

func (file *File) WriteString(s string) (ret int, err error) {
	return file.Write([]byte(s))
}

func (file *File) putFileInfoTogetherFromOpenObject(f *swift.ObjectOpenFile, headers swift.Headers, name string) (err error) {
	file.objectOpenFile = f

	// Put the file info together
	size, err := f.Length()
	if err != nil {
		return
	}
	var modtime time.Time
	modtime, err = headers.ObjectMetadata().GetModTime()
	if err != nil {
		// Try parsing it directly if the method fails
		modtime, err = time.Parse("Mon, 02 Jan 2006 15:04:05 MST", headers["Last-Modified"])
		if err == nil {
			goto putFileInfoTogether
		}

		return err
	}

putFileInfoTogether:

	var mode os.FileMode = os.ModePerm
	if headers["Content-Type"] == "application/directory" {
		mode = os.ModeDir
	}

	file.fileInfo = &FileInfo{
		name:    name,
		size:    size,
		modTime: modtime,
		mode:    mode, // Apparently there is no way to get the mode of an object. Or am I missing something?
	}
	return nil
}

// =====================
// File Info starts here
// =====================

type FileInfo struct {
	name    string
	size    int64
	mode    os.FileMode
	modTime time.Time
}

func (fi FileInfo) Name() string {
	return fi.name
}

func (fi FileInfo) Size() int64 {
	if fi.IsDir() {
		return int64(42)
	}
	return fi.size
}

func (fi FileInfo) Mode() os.FileMode {
	return fi.mode
}

func (fi FileInfo) ModTime() time.Time {
	return fi.modTime
}

func (fi FileInfo) IsDir() bool {
	return fi.Mode().IsDir()
}

func (fi FileInfo) Sys() interface{} {
	return &fi
}
