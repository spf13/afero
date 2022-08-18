package rclonefs

import (
	"context"
	"fmt"
	"os/user"
	"path/filepath"
	"strings"

	_ "github.com/rclone/rclone/backend/all"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configfile"
	"github.com/rclone/rclone/vfs"
)

type RCFS struct {
	Fs *vfs.VFS
	Cwd string
}

func (rcfs *RCFS) CreateRCFS(path string) (*RCFS, error) {
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

	rfs, e := fs.NewFs(context.Background(), rootdir)
	if e != nil {
		return nil, e
	}

	vfs := vfs.New(rfs, nil)

	data, _ := vfs.ReadFile("mock.json")
	fmt.Printf("%s\n", string(data))

	return &RCFS{Fs: vfs, Cwd: cwd}, nil
}
