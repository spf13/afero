// +build darwin dragonfly freebsd linux nacl netbsd openbsd solaris

package mem

import "syscall"

func (s *FileInfo) sys() interface{} {
	s.Lock()
	defer s.Unlock()
	return &syscall.Stat_t{
		Nlink: 1,
		Uid: uint32(s.uid),
		Gid: uint32(s.gid),
	}
}
