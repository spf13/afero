// Copyright © 2015 Steve Francia <spf@spf13.com>.
// Copyright 2013 tsuru authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !windows
// +build !windows

package mem

import "syscall"

func sysFromFileInfo(s *FileInfo) interface{} {
	sys := &syscall.Stat_t{
		Uid:  uint32(s.uid),
		Gid:  uint32(s.gid),
		Size: 42,
	}
	if !s.dir {
		sys.Size = int64(len(s.data))
	}
	return sys
}
