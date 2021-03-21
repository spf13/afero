// Copyright © 2017 Blake Williams <code@shabbyrobe.org>
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
	gorand "math/rand"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"
)

func must(t *testing.T, errs ...error) {
	// Re-enable when Go 1.9 is the minimum
	// t.Helper()
	for _, err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestMountableFsChild(t *testing.T) {
	base := NewMemMapFs()
	mfs := NewMountableFs(base)

	child1 := NewMemMapFs()
	must(t,
		mfs.Mount(filepath.FromSlash("test"), child1),
		WriteFile(mfs, filepath.FromSlash("/foo"), []byte("1"), 0644),
		WriteFile(mfs, filepath.FromSlash("/test/bar"), []byte("2"), 0644))

	if err := checkPaths([]string{"/", "/foo"}, base); err != nil {
		t.Fatal(err)
	}
	if err := checkPaths([]string{"/", "/bar"}, child1); err != nil {
		t.Fatal(err)
	}
	if err := checkPaths([]string{"/", "/foo", "/test", "/test/bar"}, mfs); err != nil {
		t.Fatal(err)
	}
}

func TestMountableFsNestedChild(t *testing.T) {
	base := NewMemMapFs()
	mfs := NewMountableFs(base)

	child := NewMemMapFs()
	must(t,
		mfs.Mount(filepath.FromSlash("test/a/b"), child),
		WriteFile(mfs, filepath.FromSlash("/foo"), []byte("1"), 0644),
		WriteFile(mfs, filepath.FromSlash("/test/a/b/baz"), []byte("3"), 0644))

	if err := checkPaths([]string{"/", "/foo"}, base); err != nil {
		t.Fatal(err)
	}
	if err := checkPaths([]string{"/", "/baz"}, child); err != nil {
		t.Fatal(err)
	}
	if err := checkPaths([]string{"/", "/foo", "/test", "/test/a", "/test/a/b", "/test/a/b/baz"}, mfs); err != nil {
		t.Fatal(err)
	}
}

func TestMountableFsSameFsMultipleMounts(t *testing.T) {
	mfs := NewMountableFs(nil)
	child := NewMemMapFs()
	must(t,
		mfs.Mount(filepath.FromSlash("/foo"), child),
		mfs.Mount(filepath.FromSlash("/bar"), child),
		WriteFile(mfs, filepath.FromSlash("/child/test"), []byte("1"), 0644),
	)

	if err := WriteFile(mfs, filepath.FromSlash("/foo/test"), []byte("1"), 0644); err != nil {
		t.Fatal(err)
	}
	if exists, err := Exists(mfs, filepath.FromSlash("/foo/test")); err != nil || !exists {
		t.Fatal(err, exists)
	}
	if exists, err := Exists(mfs, filepath.FromSlash("/bar/test")); err != nil || !exists {
		t.Fatal(err, exists)
	}
}

func TestMountableFsRecursive(t *testing.T) {
	{ // recursive mount should fail
		mfs := NewMountableFs(nil)
		child := NewMemMapFs()
		must(t, mfs.Mount(filepath.FromSlash("/child"), child))
		if err := mfs.Mount(filepath.FromSlash("/child/nested"), child); !IsErrRecursiveMount(err) {
			t.Fatal(err)
		}
	}
	{ // recursive mount allowed
		mfs := NewMountableFs(nil)
		mfs.AllowRecursiveMount = true
		child := NewMemMapFs()
		must(t, mfs.Mount(filepath.FromSlash("/child"), child),
			mfs.Mount(filepath.FromSlash("/child/nested"), child),
			WriteFile(mfs, filepath.FromSlash("/child/foo"), []byte("1"), 0644),
		)
		if err := WriteFile(mfs, filepath.FromSlash("/child/nested/foo"), []byte("1"), 0644); err != nil {
			t.Fatal(err)
		}
		if exists, err := Exists(mfs, filepath.FromSlash("/child/foo")); err != nil || !exists {
			t.Fatal(err, exists)
		}
		if exists, err := Exists(mfs, filepath.FromSlash("/child/nested/foo")); err != nil || !exists {
			t.Fatal(err, exists)
		}
	}
}

func TestMountableFsNestedChildInsideAnotherMount(t *testing.T) {
	base := NewMemMapFs()
	mfs := NewMountableFs(base)

	child1 := NewMemMapFs()
	child2 := NewMemMapFs()
	must(t,
		mfs.Mount(filepath.FromSlash("test"), child1),
		mfs.Mount(filepath.FromSlash("test/a/b"), child2),
		WriteFile(mfs, filepath.FromSlash("/foo"), []byte("1"), 0644),
		WriteFile(mfs, filepath.FromSlash("/test/bar"), []byte("2"), 0644),
		WriteFile(mfs, filepath.FromSlash("/test/a/b/baz"), []byte("3"), 0644),
	)

	if err := checkPaths([]string{"/", "/foo"}, base); err != nil {
		t.Fatal(err)
	}
	if err := checkPaths([]string{"/", "/bar"}, child1); err != nil {
		t.Fatal(err)
	}
	if err := checkPaths([]string{"/", "/baz"}, child2); err != nil {
		t.Fatal(err)
	}
	if err := checkPaths([]string{"/", "/foo", "/test", "/test/a", "/test/a/b", "/test/a/b/baz", "/test/bar"}, mfs); err != nil {
		t.Fatal(err)
	}
}

func TestMountableFsUnmountFsMountedBetweenMounts(t *testing.T) {
	base := NewMemMapFs()
	mfs := NewMountableFs(base)

	must(t,
		mfs.Mount(filepath.FromSlash("test"), NewMemMapFs()),
		mfs.Mount(filepath.FromSlash("test/foo"), NewMemMapFs()),
		mfs.Mount(filepath.FromSlash("test/foo/bar"), NewMemMapFs()),
		mfs.Mount(filepath.FromSlash("test/bar"), NewMemMapFs()),

		WriteFile(mfs, filepath.FromSlash("/test/a"), []byte("1"), 0644),
		WriteFile(mfs, filepath.FromSlash("/test/foo/a"), []byte("2"), 0644),
		WriteFile(mfs, filepath.FromSlash("/test/foo/bar/a"), []byte("3"), 0644),
		WriteFile(mfs, filepath.FromSlash("/test/bar/a"), []byte("4"), 0644),
	)

	if err := checkPaths([]string{"/", "/test", "/test/a", "/test/foo", "/test/foo/a", "/test/foo/bar", "/test/foo/bar/a", "/test/bar", "/test/bar/a"}, mfs); err != nil {
		t.Fatal(err)
	}
	must(t, mfs.Umount(filepath.FromSlash("test")))
	if err := checkPaths([]string{"/", "/test", "/test/foo", "/test/foo/a", "/test/foo/bar", "/test/foo/bar/a", "/test/bar", "/test/bar/a"}, mfs); err != nil {
		t.Fatal(err)
	}
	must(t, mfs.Umount(filepath.FromSlash("test/foo")))
	if err := checkPaths([]string{"/", "/test", "/test/foo", "/test/foo/bar", "/test/foo/bar/a", "/test/bar", "/test/bar/a"}, mfs); err != nil {
		t.Fatal(err)
	}
	must(t, mfs.Umount(filepath.FromSlash("test/bar")))
	if err := checkPaths([]string{"/", "/test", "/test/foo", "/test/foo/bar", "/test/foo/bar/a"}, mfs); err != nil {
		t.Fatal(err)
	}
	must(t, mfs.Umount(filepath.FromSlash("test/foo/bar")))
	if err := checkPaths([]string{"/"}, mfs); err != nil {
		t.Fatal(err)
	}
}

func TestMountableFsWriteFileOverNodeShouldFail(t *testing.T) {
	base := NewMemMapFs()

	mfs := NewMountableFs(base)
	must(t, mfs.Mount(filepath.FromSlash("test"), NewMemMapFs()))
	if err := WriteFile(mfs, filepath.FromSlash("/test"), []byte("1"), 0644); err != nil {
		t.Fatal(err)
	}

	// Intermediate nodes should fail too
	mfs = NewMountableFs(base)
	must(t, mfs.Mount(filepath.FromSlash("test/test"), NewMemMapFs()))
	if err := WriteFile(mfs, filepath.FromSlash("/test"), []byte("1"), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestMountableFsMountErrors(t *testing.T) {
	{ // Can not mount over an existing mount
		base := NewMemMapFs()
		mfs := NewMountableFs(base)
		if err := mfs.Mount(filepath.FromSlash("test"), NewMemMapFs()); err != nil {
			t.Fatal(err)
		}
		if err := mfs.Mount(filepath.FromSlash("test"), NewMemMapFs()); !IsErrAlreadyMounted(err) {
			t.Fatal(err)
		}
	}

	{ // Can not mount over an existing dir
		base := NewMemMapFs()
		mfs := NewMountableFs(base)
		if err := mfs.Mkdir(filepath.FromSlash("test"), 0777); err != nil {
			t.Fatal(err)
		}
		if err := mfs.Mount(filepath.FromSlash("test"), NewMemMapFs()); !os.IsExist(err) {
			t.Fatal(err)
		}
	}

	{ // Can not mount over an existing file
		base := NewMemMapFs()
		mfs := NewMountableFs(base)
		if err := WriteFile(mfs, filepath.FromSlash("test"), []byte("1"), 0777); err != nil {
			t.Fatal(err)
		}
		if err := mfs.Mount(filepath.FromSlash("test"), NewMemMapFs()); !os.IsExist(err) {
			t.Fatal(err)
		}
	}
}

func TestMountableFsMountOverIntermediateNode(t *testing.T) {
	base := NewMemMapFs()
	mfs := NewMountableFs(base)

	child1 := NewMemMapFs()
	if err := mfs.Mount(filepath.FromSlash("test/a/b"), child1); err != nil {
		t.Fatal(err)
	}

	// intermediate node "test" already exists, but has no fs attached yet
	// so this should still work.
	child2 := NewMemMapFs()
	if err := mfs.Mount(filepath.FromSlash("test"), child2); err != nil {
		t.Fatal(err)
	}

	must(t,
		WriteFile(mfs, filepath.FromSlash("/foo"), []byte("1"), 0644),
		WriteFile(mfs, filepath.FromSlash("/test/a/b/baz"), []byte("3"), 0644))

	if err := checkPaths([]string{"/", "/foo"}, base); err != nil {
		t.Fatal(err)
	}
	if err := checkPaths([]string{"/", "/baz"}, child1); err != nil {
		t.Fatal(err)
	}
	if err := checkPaths([]string{"/", "/foo", "/test", "/test/a", "/test/a/b", "/test/a/b/baz"}, mfs); err != nil {
		t.Fatal(err)
	}
}

func TestMountableFsWriteIntoIntermediateNode(t *testing.T) {
	var end []func()
	defer func() {
		for _, f := range end {
			f()
		}
	}()
	var bases = []func() Fs{
		func() Fs { return NewMemMapFs() },
	}

	if osSupportsPerms() {
		bases = append(bases, func() Fs {
			tfs := newTempFs()
			end = append(end, func() { tfs.Destroy() })
			return tfs
		})
	}

	for idx, base := range bases {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			{ // intermediate nodes are created:
				base := base()
				mfs := NewMountableFs(base)
				child1 := NewMemMapFs()
				must(t, mfs.Mount(filepath.FromSlash("/test/a/b"), child1))
				if exists, _ := Exists(base, filepath.FromSlash("/test/a")); exists {
					t.Fatal()
				}
				must(t, WriteFile(mfs, filepath.FromSlash("/test/a/yep"), []byte("1"), 0755))
				mustStatRealDir(t, base, "/test", 0755)
				mustStatRealDir(t, base, "/test/a", 0755)
			}

			{ // intermediate child of real dir is created:
				base := base()
				mfs := NewMountableFs(base)
				child1 := NewMemMapFs()
				must(t, mfs.Mkdir(filepath.FromSlash("/test"), 0755))
				must(t, mfs.Mount(filepath.FromSlash("/test/a/b"), child1))
				if exists, _ := Exists(base, filepath.FromSlash("/test")); !exists {
					t.Fatal()
				}
				if exists, _ := Exists(base, filepath.FromSlash("/test/a")); exists {
					t.Fatal()
				}
				must(t, WriteFile(mfs, filepath.FromSlash("/test/a/yep"), []byte("1"), 0755))
				mustStatRealDir(t, base, "/test/a", 0755)
			}
		})
	}
}

func TestMountableFsReadDir(t *testing.T) {
	base := NewMemMapFs()

	now := time.Now()
	child := NewMemMapFs()
	must(t,
		child.Chmod(filepath.FromSlash("/"), 0777),
		child.Chtimes(filepath.FromSlash("/"), now, now))

	mfs := NewMountableFs(base)
	must(t, mfs.Mount(filepath.FromSlash("test/test"), child))

	{
		read1, err := ReadDir(mfs, filepath.FromSlash("/"))
		if err != nil {
			t.Fatal(err)
		}
		if len(read1) != 1 {
			t.Fatal()
		}
		if read1[0].Name() != "test" {
			t.Fatal()
		}
		if read1[0].Mode() != 0777|os.ModeDir {
			t.Fatal()
		}
		if read1[0].ModTime().IsZero() {
			t.Fatal()
		}
	}

	{
		read2, err := ReadDir(mfs, filepath.FromSlash("/test"))
		if err != nil {
			t.Fatal(err)
		}
		if len(read2) != 1 {
			t.Fatal()
		}
		if read2[0].Name() != "test" {
			t.Fatal()
		}
		if read2[0].Mode() != 0777|os.ModeDir {
			t.Fatal()
		}
		if read2[0].ModTime() != now {
			t.Fatal()
		}
	}
}

func TestMountableFsRename(t *testing.T) {
	base := NewMemMapFs()
	mfs := NewMountableFs(base)

	input := []byte("1")
	if err := WriteFile(mfs, filepath.FromSlash("/test/foo"), input, 0644); err != nil {
		t.Fatal(err)
	}
	err := mfs.Rename(filepath.FromSlash("/test/foo"), filepath.FromSlash("/bar"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = ReadFile(mfs, filepath.FromSlash("/test/foo"))
	if !os.IsNotExist(err) {
		t.Fatal(err)
	}
	result, err := ReadFile(mfs, filepath.FromSlash("/bar"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(result, input) {
		t.Fatal()
	}
}

func TestMountableFsCrossFsRename(t *testing.T) {
	base := NewMemMapFs()
	mfs := NewMountableFs(base)

	child := NewMemMapFs()
	if err := mfs.Mount(filepath.FromSlash("test"), child); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(mfs, filepath.FromSlash("/test/foo"), []byte("1"), 0644); err != nil {
		t.Fatal(err)
	}
	err := mfs.Rename(filepath.FromSlash("/test/foo"), filepath.FromSlash("/bar"))
	if err != errCrossFsRename {
		t.Fatal(err)
	}
}

func TestMountableFsMemMapFsOverOsFs(t *testing.T) {
	tfs := newTempFs()
	defer tfs.Destroy()

	mfs := NewMountableFs(tfs)
	child := NewMemMapFs()
	if err := mfs.Mount(filepath.FromSlash("/child"), child); err != nil {
		t.Fatal(err)
	}
	in := []byte("1")
	if err := WriteFile(mfs, filepath.FromSlash("/child/foo"), in, 0644); err != nil {
		t.Fatal(err)
	}
	out, err := ReadFile(child, filepath.FromSlash("/foo"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatal()
	}

	out, err = ReadFile(mfs, filepath.FromSlash("/child/foo"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(in, out) {
		t.Fatal()
	}
}

func TestMountableFsRemove(t *testing.T) {
	mfs := NewMountableFs(nil)
	child := NewMemMapFs()
	if err := mfs.Mount(filepath.FromSlash("/child"), child); err != nil {
		t.Fatal(err)
	}
	if err := WriteFile(mfs, filepath.FromSlash("/child/test"), []byte("1"), 0644); err != nil {
		t.Fatal(err)
	}
	if e, err := Exists(mfs, filepath.FromSlash("/child/test")); !e || err != nil {
		t.Fatal(e, err)
	}
	if err := mfs.Remove(filepath.FromSlash("/child/test")); err != nil {
		t.Fatal(err)
	}
	if e, err := Exists(mfs, filepath.FromSlash("/child/test")); e || err != nil {
		t.Fatal(e, err)
	}
}

func TestMountableFsRemoveAll(t *testing.T) {
	mfs := NewMountableFs(nil)
	child1 := NewMemMapFs()
	child2 := NewMemMapFs()
	must(t,
		mfs.Mount(filepath.FromSlash("/mntchild"), child1),
		mfs.Mount(filepath.FromSlash("/mntchild/nested"), child2),
		WriteFile(mfs, filepath.FromSlash("/test"), []byte("1"), 0644),
		WriteFile(mfs, filepath.FromSlash("/child/test"), []byte("1"), 0644),
		WriteFile(mfs, filepath.FromSlash("/mntchild/foo"), []byte("1"), 0644),
		WriteFile(mfs, filepath.FromSlash("/mntchild/bar"), []byte("1"), 0644),
		WriteFile(mfs, filepath.FromSlash("/mntchild/baz/qux"), []byte("1"), 0644),
		WriteFile(mfs, filepath.FromSlash("/mntchild/nested/a"), []byte("1"), 0644),
		WriteFile(mfs, filepath.FromSlash("/mntchild/nested/b/c"), []byte("1"), 0644),
	)
	if err := mfs.RemoveAll(filepath.FromSlash("/")); err != nil {
		t.Fatal(err)
	}
	paths := []string{}
	err := Walk(mfs, filepath.FromSlash("/"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		paths = append(paths, path)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(slashifyAll([]string{"/", "/mntchild", "/mntchild/nested"}), paths) {
		t.Fatal(paths)
	}
}

func TestMountableFsRemoveAllRecursiveMount(t *testing.T) {
	mfs := NewMountableFs(nil)
	mfs.AllowRecursiveMount = true
	child1 := NewMemMapFs()
	must(t,
		mfs.Mount(filepath.FromSlash("/mntchild"), child1),
		mfs.Mount(filepath.FromSlash("/mntchild/nested"), child1),
		WriteFile(mfs, filepath.FromSlash("/mntchild/foo"), []byte("1"), 0644),
		WriteFile(mfs, filepath.FromSlash("/mntchild/nested/bar"), []byte("1"), 0644),
	)
	if err := mfs.RemoveAll(filepath.FromSlash("/")); err != nil {
		t.Fatal(err)
	}
	paths := []string{}
	err := Walk(mfs, filepath.FromSlash("/"), collectPaths(&paths))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(slashifyAll([]string{"/", "/mntchild", "/mntchild/nested"}), paths) {
		t.Fatal(paths)
	}
}

func TestMountableFsStat(t *testing.T) {
	base := NewMemMapFs()
	mfs := NewMountableFs(base)
	child := NewMemMapFs()
	child2 := NewMemMapFs()
	must(t,
		mfs.Mount(filepath.FromSlash("/child"), child),
		mfs.Mount(filepath.FromSlash("/child/nested"), child2),
		WriteFile(mfs, filepath.FromSlash("/child/foo"), []byte("1"), 0644),
		WriteFile(mfs, filepath.FromSlash("/child/nested/baz"), []byte("1"), 0644))

	if i, err := mfs.Stat(filepath.FromSlash("/child/foo")); err != nil {
		t.Fatal(err)
	} else {
		if i.Name() != "foo" || i.IsDir() {
			t.Fatal()
		}
	}
	if i, err := mfs.Stat(filepath.FromSlash("/child/nested/baz")); err != nil {
		t.Fatal(err)
	} else {
		if i.Name() != "baz" || i.IsDir() {
			t.Fatal()
		}
	}
	if i, err := mfs.Stat(filepath.FromSlash("/child/nested")); err != nil {
		t.Fatal(err)
	} else {
		if i.Name() != "nested" || !i.IsDir() {
			t.Fatal()
		}
	}
	if i, err := mfs.Stat(filepath.FromSlash("/")); err != nil {
		t.Fatal(err)
	} else {
		if i.Name() != "" || !i.IsDir() {
			t.Fatal()
		}
	}
}

func TestMountableFsMkdirAll(t *testing.T) {
	base := NewMemMapFs()
	mfs := NewMountableFs(base)
	child1 := NewMemMapFs()
	child1.Chmod(filepath.FromSlash("/"), 0777)
	child2 := NewMemMapFs()
	child2.Chmod(filepath.FromSlash("/"), 0777)

	must(t,
		mfs.Mount(filepath.FromSlash("/child"), child1),
		mfs.Mount(filepath.FromSlash("/child/deeply/nested"), child2))

	if err := mfs.MkdirAll(filepath.FromSlash("/child/deeply/nested/dir"), 0755); err != nil {
		t.Fatal(err)
	}

	mustStatLooseDir(t, mfs, filepath.FromSlash("/child"), 0777|os.ModeDir)
	mustStatLooseDir(t, mfs, filepath.FromSlash("/child/deeply"), 0777|os.ModeDir)
	mustStatLooseDir(t, mfs, filepath.FromSlash("/child/deeply/nested"), 0777|os.ModeDir)
	mustStatLooseDir(t, mfs, filepath.FromSlash("/child/deeply/nested/dir"), 0755|os.ModeDir)

	// directories should be created under the mounts
	mustStatLooseDir(t, base, filepath.FromSlash("/child"), 0755|os.ModeDir)
	mustStatLooseDir(t, child1, filepath.FromSlash("/deeply"), 0755|os.ModeDir)
	mustStatLooseDir(t, child1, filepath.FromSlash("/deeply/nested"), 0755|os.ModeDir)
}

func TestMountableFsChmod(t *testing.T) {
	mfs := NewMountableFs(nil)
	child1 := NewMemMapFs()
	child2 := NewMemMapFs()
	must(t,
		mfs.Mount(filepath.FromSlash("/child"), child1),

		// Ensure there is an intermediate node
		mfs.Mount(filepath.FromSlash("/child/deeply/nested"), child2),

		WriteFile(mfs, filepath.FromSlash("/child/foo"), []byte("1"), 0644),
		mfs.Mkdir(filepath.FromSlash("/child/dir"), 0755))

	// sanity checks - make sure the initial state matches this test's expectation
	mustStatLooseDir(t, mfs, filepath.FromSlash("/child"), os.ModeDir)
	mustStat(t, mfs, filepath.FromSlash("/child/foo"), 0644)
	mustStatLooseDir(t, mfs, filepath.FromSlash("/child/dir"), 0755|os.ModeDir)
	mustStatLooseDir(t, mfs, filepath.FromSlash("/child/deeply"), 0777|os.ModeDir)
	mustStatLooseDir(t, mfs, filepath.FromSlash("/child/deeply/nested"), os.ModeDir)

	if err := mfs.Chmod(filepath.FromSlash("/child"), 0777|os.ModeDir); err != nil {
		t.Fatal(err)
	}
	if err := mfs.Chmod(filepath.FromSlash("/child/foo"), 0777); err != nil {
		t.Fatal(err)
	}

	// This seems weird about MemMapFs - it will return os.ModeDir if you call
	// Mkdir, but it allows you to Chmod this flag away even though the entry
	// remains a directory.
	if err := mfs.Chmod(filepath.FromSlash("/child/dir"), 0777|os.ModeDir); err != nil {
		t.Fatal(err)
	}

	mustStatLooseDir(t, mfs, filepath.FromSlash("/child"), 0777)
	mustStat(t, mfs, filepath.FromSlash("/child/foo"), 0777)
	mustStatLooseDir(t, mfs, filepath.FromSlash("/child/dir"), 0777)
}

func TestMountableFsChtimes(t *testing.T) {
	mfs := NewMountableFs(nil)
	child := NewMemMapFs()
	must(t,
		mfs.Mount(filepath.FromSlash("/child"), child),
		WriteFile(mfs, filepath.FromSlash("/child/foo"), []byte("1"), 0644),
		mfs.Mkdir(filepath.FromSlash("/child/dir"), 0755))

	tm := time.Date(2017, 1, 1, 12, 0, 0, 0, time.UTC)

	must(t,
		mfs.Chtimes(filepath.FromSlash("/child"), time.Time{}, tm),
		mfs.Chtimes(filepath.FromSlash("/child/foo"), time.Time{}, tm),
		mfs.Chtimes(filepath.FromSlash("/child/dir"), time.Time{}, tm))

	mustTime(t, mfs, filepath.FromSlash("/child"), tm)
	mustTime(t, mfs, filepath.FromSlash("/child/foo"), tm)
	mustTime(t, mfs, filepath.FromSlash("/child/dir"), tm)
}

func TestMountableFsPathErrors(t *testing.T) {
	mfs := NewMountableFs(nil)
	child := NewMemMapFs()
	must(t, mfs.Mount(filepath.FromSlash("/nested/child"), child))

	expected := filepath.FromSlash("/nested/child/dir/file")

	funcs := []func() error{
		func() error {
			_, err := mfs.Stat(expected)
			return err
		},
		func() error { return mfs.Chmod(expected, 0644) },
		func() error { return mfs.Chtimes(expected, time.Time{}, time.Time{}) },
		func() error { return mfs.Rename(expected, filepath.FromSlash("/nested/child/flarg")) },
		func() error { return mfs.Remove(expected) },
		func() error { return mfs.RemoveAll(expected) },

		// MemMapFs creates intermediate dirs!
		// func() error { return mfs.Mkdir(expected, 0777) },
	}
	for i, fn := range funcs {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			err := fn()
			if err == nil {
				t.Fatal()
			}
			perr, ok := err.(*os.PathError)
			if !ok {
				t.Fatal(err)
			}
			if perr.Path != expected {
				t.Fatal(perr.Path, "!=", expected)
			}
			if perr.Err == nil || perr.Err != underlyingError(perr) {
				t.Fatal()
			}
		})
	}
}

func mustTime(t *testing.T, fs Fs, path string, tm time.Time) {
	// Re-enable when Go 1.9 is the minimum
	// t.Helper()
	path = filepath.FromSlash(path)
	if i, err := fs.Stat(path); err != nil {
		t.Fatal(err)
	} else {
		if i.ModTime() != tm {
			t.Fatal(i.ModTime(), "!=", tm)
		}
	}
}

func mustStatLooseDir(t *testing.T, fs Fs, dir string, mode os.FileMode) {
	// Re-enable when Go 1.9 is the minimum
	// t.Helper()
	dir = filepath.FromSlash(dir)
	mode |= os.ModeDir
	if i, err := fs.Stat(dir); err != nil {
		t.Fatal(err)
	} else {
		if i.Name() != filepath.Base(dir) {
			t.Fatal(i.Name(), filepath.Base(dir))
		} else if !i.IsDir() {
			t.Fatal("not dir")
		} else if i.Mode() != mode {
			t.Fatal("mode mismatch", mode, i.Mode())
		}
	}
}

func mustStatRealDir(t *testing.T, fs Fs, dir string, mode os.FileMode) {
	// Re-enable when Go 1.9 is the minimum
	// t.Helper()
	dir = filepath.FromSlash(dir)
	mustStatLooseDir(t, fs, dir, mode)
	i, _ := fs.Stat(dir)
	_, ok := i.(*mountedDirInfo)
	if ok {
		t.Fatal()
	}
}

func mustStatMount(t *testing.T, fs Fs, dir string, mode os.FileMode) {
	// Re-enable when Go 1.9 is the minimum
	// t.Helper()
	dir = filepath.FromSlash(dir)
	mustStatLooseDir(t, fs, dir, mode)
	i, _ := fs.Stat(dir)
	_, ok := i.(*mountedDirInfo)
	if !ok {
		t.Fatal()
	}
}

func mustStat(t *testing.T, fs Fs, file string, mode os.FileMode) {
	// Re-enable when Go 1.9 is the minimum
	// t.Helper()
	file = filepath.FromSlash(file)
	if i, err := fs.Stat(file); err != nil {
		t.Fatal(err)
	} else {
		if i.Name() != filepath.Base(file) {
			t.Fatal(i.Name(), filepath.Base(file))
		} else if i.IsDir() {
			t.Fatal("not file")
		} else if i.Mode() != mode {
			t.Fatal("mode mismatch", mode, i.Mode())
		}
	}
}

func checkPaths(expected []string, fs Fs) error {
	result := []string{}
	if err := Walk(fs, filepath.FromSlash("/"), collectPaths(&result)); err != nil {
		return err
	}
	for i, p := range expected {
		expected[i] = filepath.FromSlash(p)
	}

	sort.Strings(result)
	sort.Strings(expected)
	if !reflect.DeepEqual(expected, result) {
		return fmt.Errorf("paths did not match.\nexpect: %v\nactual: %v", expected, result)
	}
	return nil
}

func collectPaths(into *[]string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		*into = append(*into, path)
		return nil
	}
}

func printFile(path string, info os.FileInfo, err error) error {
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(path, info.Name())
	}
	return nil
}

var tempFsRandSource = gorand.NewSource(time.Now().UnixNano())

type tempFs struct {
	Fs
	path string
}

func (t *tempFs) Destroy() {
	if !strings.HasPrefix(t.path, os.TempDir()) {
		panic("path did not start with temp dir")
	}
	if err := os.RemoveAll(t.path); err != nil {
		panic(err)
	}
	if err := os.RemoveAll(t.path); err != nil {
		panic(err)
	}
}

func newTempFs() *tempFs {
	p := fmt.Sprintf("%d.%d", time.Now().UnixNano(), tempFsRandSource.Int63())
	td := filepath.Join(os.TempDir(), p)
	if err := os.Mkdir(td, 0777); err != nil {
		panic(err)
	}
	t := NewBasePathFs(NewOsFs(), td)
	return &tempFs{Fs: t, path: td}
}

var osPermCheck = -1

func osSupportsPerms() bool {
	if osPermCheck < 0 {
		osPermCheck = 0
		tempFs := newTempFs()
		defer tempFs.Destroy()

		if err := WriteFile(tempFs, "test", []byte("1"), 0777); err != nil {
			panic(err)
		}
		info, err := tempFs.Stat("test")
		if err != nil {
			panic(err)
		}
		origMode := info.Mode()
		if err := tempFs.Chmod("test", 0666); err != nil {
			panic(err)
		}
		info, err = tempFs.Stat("test")
		if err != nil {
			panic(err)
		}
		if info.Mode() != origMode {
			osPermCheck = 1
		}
	}
	return osPermCheck == 1
}

func slashifyAll(paths []string) []string {
	for i, p := range paths {
		paths[i] = filepath.FromSlash(p)
	}
	return paths
}
