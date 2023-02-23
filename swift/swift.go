package swift

import (
	"bytes"
	"fmt"
	"github.com/ncw/swift"
	"log"
	"os"
	"time"
)

type Fs struct {
	Connection    *swift.Connection
	containerName string
}

func NewSwiftFs(conn *swift.Connection, containerName string) (fs *Fs, err error) {
	if containerName == "" {
		return fs, fmt.Errorf("container name cannot be empty")
	}
	fs = &Fs{
		Connection:    conn,
		containerName: containerName,
	}
	err = fs.Connection.Authenticate()
	return
}

func (s *Fs) Name() string {
	return "swiftfs"
}

func (s *Fs) Create(name string) (file File, err error) {
	objectfile, err := s.Connection.ObjectCreate(s.containerName, name, true, "", "", swift.Headers{})
	if err != nil {
		return file, err
	}
	file.objectCreateFile = objectfile

	return
}

func (s *Fs) Mkdir(name string, perm os.FileMode) (err error) {
	return s.MkdirAll(name, perm)
}

func (s *Fs) MkdirAll(path string, perm os.FileMode) (err error) {
	// Swift usually creates all directories deep down, no matter if any of these directories already exist
	// Objects created with application/directory are so called "pseudo-directories".
	_, err = s.Connection.ObjectPut(s.containerName, path+"/", bytes.NewReader([]byte{}), true, "", "application/directory", swift.Headers{})
	return
}

func (s *Fs) Open(name string) (file *File, err error) {
	file = new(File)
	file.conn = s.Connection
	file.containerName = s.containerName

	f, headers, err := s.Connection.ObjectOpen(s.containerName, name, true, swift.Headers{})
	if err != nil {
		return
	}
	err = file.putFileInfoTogetherFromOpenObject(f, headers, name)
	return
}

func (*Fs) OpenFile(name string, flag int, perm os.FileMode) (*File, error) {
	return nil, nil
}

func (s *Fs) Remove(name string) error {
	return s.Connection.ObjectDelete(s.containerName, name)
}

func (s *Fs) RemoveAll(path string) (err error) {
	sinfo, err := s.Connection.QueryInfo()
	if err != nil {
		return
	}

	// Get all objects we want to delte
	objects, err := s.Connection.ObjectsAll(s.containerName, &swift.ObjectsOpts{Prefix: path})
	if err != nil {
		return
	}

	var objectnames []string
	for _, o := range objects {
		// If the server supports bulk delete, add it to a list to bulkdelete the items later,
		// otherwise delete them directly
		if sinfo.SupportsBulkDelete() {
			objectnames = append(objectnames, o.Name)
		} else {
			err = s.Connection.ObjectDelete(s.containerName, o.Name)
			if err != nil {
				return
			}
		}
	}
	if sinfo.SupportsBulkDelete() {
		_, err = s.Connection.BulkDelete(s.containerName, objectnames)
		if err != nil {
			return
		}
	}

	return err
}

func (s *Fs) Rename(oldname, newname string) error {
	return s.Connection.ObjectMove(s.containerName, oldname, s.containerName, newname)
}

func (s *Fs) Stat(name string) (os.FileInfo, error) {
	file, err := s.Open(name)
	if err != nil {
		return nil, err
	}
	return file.Stat()
}

func (*Fs) Chmod(name string, mode os.FileMode) error {
	log.Println("Swift does not support file modes.")
	return nil
}

func (s *Fs) Chtimes(name string, _, mtime time.Time) error {
	// While the uncommented code should work, the problem is that we have no way to update the time on
	// the server without modifying the time. Whenever we send a request to swift to update the last modified time,
	// we modify the file thus setting the last modified time to now...
	// Maybe I'll find a solution for this in the future, but for now I'll just diable it because of the reasons above.
	return nil
	/*
		_, headers, err := s.Connection.ObjectOpen(s.containerName, name, true, swift.Headers{})

		if err != nil {
			return err
		}
		headers.ObjectMetadata().SetModTime(mtime) // Swift does not support access time so we won't use that.
		return s.Connection.ObjectUpdate(s.containerName, name, headers)
	*/
}
