package rclonefs

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"time"

	_ "github.com/rclone/rclone/backend/all"
	rclfs "github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configfile"
	"github.com/rclone/rclone/vfs"
	"github.com/spf13/afero"
)

type RCFS struct {
	Fs	*vfs.VFS
	Cwd	string
}

func CreateRCFS(path string) (*RCFS, error) {
	u, e := user.Current()
	if e != nil {
		return nil, e
	}

	cfgpath := filepath.Join(u.HomeDir, ".config/rclone/rclone.conf")

	e = config.SetConfigPath(cfgpath)
	if e != nil {
		return nil, e
	}

	configfile.Install()

	rootdir, cwd, _ := strings.Cut(path, ":")
	rootdir += ":"
	cwd = filepath.Join("/", cwd)

	rfs, e := rclfs.NewFs(context.Background(), rootdir)
	if e != nil {
		return nil, e
	}

	vfs := vfs.New(rfs, nil)

	return &RCFS{Fs: vfs, Cwd: cwd}, nil
}

func (rcfs *RCFS) AbsPath(name string) string {
	if !filepath.IsAbs(name) {
		name = filepath.Join(rcfs.Cwd, name)
	}

	return name
}

func (rcfs *RCFS) Name() string { return "RClone virtual filesystem" }

func (rcfs *RCFS) Create(name string) (afero.File, error) {
	name = rcfs.AbsPath(name)

	return rcfs.Fs.Create(name)
}

func (rcfs *RCFS) Mkdir(name string, perm os.FileMode) error {
	name = rcfs.AbsPath(name)

	return rcfs.Fs.Mkdir(name, perm)
}

func (rcfs *RCFS) MkdirAll(name string, perm os.FileMode) error {
	name = rcfs.AbsPath(name)
	name = strings.Trim(name, "/")

	var (
		front string = "/"
		current string
		back string
		ok bool
	)

	for {
		current, back, ok = strings.Cut(name, "/")
		if !ok {
			front = front + name
			if e := rcfs.Fs.Mkdir(front, perm); e != nil && !os.IsExist(e) {
				return e
			}
			break
		}

		front = front + current + "/"

		if e := rcfs.Fs.Mkdir(front, perm); e != nil && !os.IsExist(e) {
			return e
		}

		name = back
	}

	return nil
}

func (rcfs *RCFS) Open(name string) (afero.File, error) {
	name = rcfs.AbsPath(name)

	f, e := rcfs.Fs.Open(name)
	if f == nil {
		return nil, e
	}
	return f, e
}

func (rcfs *RCFS) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	name = rcfs.AbsPath(name)

	return rcfs.Fs.OpenFile(name, flag, perm)
}

func (rcfs *RCFS) Stat(name string) (os.FileInfo, error) {
	name = rcfs.AbsPath(name)

	return rcfs.Fs.Stat(name)
}

func (rcfs *RCFS) Remove(name string) error {
	name = rcfs.AbsPath(name)

	return rcfs.Fs.Remove(name)
}

func (rcfs *RCFS) RemoveAll(path string) error {
	path = rcfs.AbsPath(path)

	fileList := make([]string, 0)
	dirList := make([]string, 0)

	e := afero.Walk(rcfs, path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			dirList = append(dirList, path)
		} else {
			fileList = append(fileList, path)
		}

		return nil
	})

	if e != nil {
		return e
	}

	for _, f := range fileList {
		if e := rcfs.Remove(f); e != nil {
			return e
		}
	}

	sort.Slice(dirList, func(i, j int) bool {
		return len(dirList[i]) > len(dirList[j])
	})

	for _, d := range dirList {
		if e := rcfs.Remove(d); e != nil {
			return e
		}
	}

	rcfs.Remove(path)
	return nil
}

func (rcfs *RCFS) Rename(oldname, newname string) error {
	oldname = rcfs.AbsPath(oldname)
	newname = rcfs.AbsPath(newname)

	return rcfs.Fs.Rename(oldname, newname)
}

func (rcfs *RCFS) Chmod(name string, mode os.FileMode) error {
	return errors.New("Chmod is not supported")
}

func (rcfs *RCFS) Chown(name string, uid, gid int) error {
	return errors.New("Chown is not supported")
}

func (rcfs *RCFS) Chtimes(name string, atime time.Time, mtime time.Time) error {
	name = rcfs.AbsPath(name)

	return rcfs.Fs.Chtimes(name, atime, mtime)
}
