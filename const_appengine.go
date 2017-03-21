// Copyright © 2016 Steve Francia <spf@spf13.com>.
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

// +build appengine

package afero

import (
	"strconv"
)

// An Errno is an unsigned number describing an error condition.
// It implements the error interface. The zero Errno is by convention
// a non-error, so code to convert from Errno to error should use:
//	err = nil
//	if errno != 0 {
//		err = errno
//	}
type Errno uintptr

func (e Errno) Error() string {
	return "errno " + strconv.Itoa(int(e))
}

const EPERM = Errno(0x1)
const ENOENT = Errno(0x2)
const EIO = Errno(0x5)
const EEXIST = Errno(0x11)
const ENOTDIR = Errno(0x14)
const EINVAL = Errno(0x16)
const BADFD = Errno(0x4d)

const O_RDWR = 0x2
