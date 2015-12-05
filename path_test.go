// Copyright © 2014 Steve Francia <spf@spf13.com>.
// Copyright 2009 The Go Authors. All rights reserved.
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

package afero

import (
	"fmt"
	"os"
	"testing"
)

func TestWalk(t *testing.T) {
	var Fss = []Fs{&MemMapFs{}, &OsFs{}}
	var testDir = "/tmp/afero"
	var testName = "test.txt"
	for _, fs := range Fss {
		path := testDir + "/" + testName
		if err := fs.MkdirAll(testDir, 0777); err != nil {
			t.Fatal(fs.Name(), "unable to create dir", err)
		}

		f, err := fs.Create(path)
		if err != nil {
			t.Fatal(fs.Name(), "create failed:", err)
		}
		defer f.Close()
		f.WriteString("Lorem ipsum dolor sit amet, consectetur adipisicing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.")
	}

	outputs := make([]string, len(Fss))
	for i, fs := range Fss {
		walkFn := func(path string, info os.FileInfo, err error) error {
			var size int64
			if !info.IsDir() {
				size = info.Size()
			}
			outputs[i] += fmt.Sprintln(path, info.Name(), size, info.IsDir(), err)
			return nil
		}
		err := Walk(testDir, walkFn, fs)
		if err != nil {
			t.Error(err)
		}
	}
	fail := false
	for i, o := range outputs {
		if i == 0 {
			continue
		}
		if o != outputs[i-1] {
			fail = true
			break
		}
	}
	if fail {
		t.Log("Walk outputs not equal!")
		for i, o := range outputs {
			t.Log(Fss[i].Name())
			t.Log(o)
		}
		t.Fail()
	}
}
