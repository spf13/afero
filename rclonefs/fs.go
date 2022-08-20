package rclonefs

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
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
	// TODO
	return nil
}

func (rcfs *RCFS) MkdirAll(name string, perm os.FileMode) error {
	// TODO
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
	afero.Walk(rcfs, path, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		} else {
			rcfs.Remove(path)
			return nil
		}
	})

	return nil
}

func (rcfs *RCFS) Rename(oldname, newname string) error {
	oldname = rcfs.AbsPath(oldname)
	newname = rcfs.AbsPath(newname)

	return rcfs.Fs.Rename(oldname, newname)
}

func (rcfs *RCFS) Chmod(name string, mode os.FileMode) error {
	// TODO
	return nil
}

func (rcfs *RCFS) Chown(name string, uid, gid int) error {
	// TODO
	return nil
}

func (rcfs *RCFS) Chtimes(name string, atime time.Time, mtime time.Time) error {
	name = rcfs.AbsPath(name)

	return rcfs.Fs.Chtimes(name, atime, mtime)
}

func printNode(path string, info fs.FileInfo, err error) error {
	if err != nil {
		fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
		return err
	}
	if info.IsDir() {
		fmt.Printf("visited dir: %q\n", path)
		return nil
	} else {
		fmt.Printf("visited file: %q\n", path)
		return nil
	}
}
/*
func rmNode(path string, info fs.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if info.IsDir() {
		return nil
	} else {
		a
	}
}
*/



















