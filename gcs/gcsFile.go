// Copyright © 2018 Mikael Rapp, github.com/zatte
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

package gcs

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"sort"

	"cloud.google.com/go/storage"

	"google.golang.org/api/iterator"
)

// GcsFs is the Afero version adapted for GCS
type GcsFile struct {
	openFlags int
	fileMode  os.FileMode
	fhoffset  int64 //File handle specific offset
	closed    bool
	ReadDirIt *storage.ObjectIterator
	resource  *gcsFileResource
}

func NewGcsFile(
	ctx context.Context,
	fs *GcsFs,
	obj *storage.ObjectHandle,
	openFlags int,
	fileMode os.FileMode,
	name string,
) *GcsFile {
	return &GcsFile{
		openFlags: openFlags,
		fileMode:  fileMode,
		fhoffset:  0,
		closed:    false,
		ReadDirIt: nil,
		resource: &gcsFileResource{
			ctx: ctx,
			fs:  fs,

			obj:  obj,
			name: name,

			currentGcsSize: 0,

			offset: 0,
			reader: nil,
			writer: nil,
		},
	}
}

func NewGcsFileFromOldFH(
	openFlags int,
	fileMode os.FileMode,
	oldFile *gcsFileResource,
) *GcsFile {
	return &GcsFile{
		openFlags: openFlags,
		fileMode:  fileMode,
		fhoffset:  0,
		closed:    false,
		ReadDirIt: nil,

		resource: oldFile,
	}
}

func (o *GcsFile) Close() error {
	// Threre shouldn't be a case where both are open at the same time
	// but the check is omitted at this time.
	o.closed = true
	return o.resource.Close()
}

func (o *GcsFile) Seek(newOffset int64, whence int) (int64, error) {
	if o.closed {
		return 0, ErrFileClosed
	}

	//Since this is an expensive operation; let's make sure we need it
	if (whence == 0 && newOffset == o.fhoffset) || (whence == 1 && newOffset == 0) {
		return o.fhoffset, nil
	}
	log.Printf("WARNING; Seek beavhior triggerd, highly inefficent. Offset before seek is at %d\n", o.fhoffset)

	//Fore the reader/writers to be reopened (at correct offset)
	o.Sync()
	stat, err := o.Stat()
	if err != nil {
		return 0, nil
	}

	switch whence {
	case 0:
		o.fhoffset = newOffset
	case 1:
		o.fhoffset += newOffset
	case 2:
		o.fhoffset = stat.Size() + newOffset
	}
	return o.fhoffset, nil
}

func (o *GcsFile) Read(p []byte) (n int, err error) {
	return o.ReadAt(p, o.fhoffset)
}

func (o *GcsFile) ReadAt(p []byte, off int64) (n int, err error) {
	if o.closed {
		return 0, ErrFileClosed
	}

	read, err := o.resource.ReadAt(p, off)
	o.fhoffset += int64(read)
	return read, err
}

func (o *GcsFile) Write(p []byte) (n int, err error) {
	return o.WriteAt(p, o.fhoffset)
}

func (o *GcsFile) WriteAt(b []byte, off int64) (n int, err error) {
	if o.closed {
		return 0, ErrFileClosed
	}

	if o.openFlags == os.O_RDONLY {
		return 0, fmt.Errorf("File is opend as read only")
	}

	written, err := o.resource.WriteAt(b, off)
	o.fhoffset += int64(written)
	return written, err
}

func (o *GcsFile) Name() string {
	return o.resource.name
}

func (o *GcsFile) readdir(count int) ([]*fileInfo, error) {
	o.Sync()
	//normSeparators should maybe not be here; adds
	path := o.resource.fs.ensureTrailingSeparator(normSeparators(o.Name(), o.resource.fs.separator))
	if o.ReadDirIt == nil {
		//log.Printf("Querying path : %s\n", path)
		o.ReadDirIt = o.resource.fs.bucket.Objects(
			o.resource.ctx, &storage.Query{o.resource.fs.separator, path, false})
	}
	var res []*fileInfo
	for {
		object, err := o.ReadDirIt.Next()
		if err == iterator.Done {
			if len(res) > 0 || count <= 0 {
				return res, nil
			}
			return res, io.EOF
		}
		if err != nil {
			return res, err
		}

		tmp := fileInfo{object, o.resource.fs}

		// Since we create "virtual folders which are empty objects they can sometimes be returned twice
		// when we do a query (As the query will also return GCS version of "virtual folders" buy they only
		// have a .Prefix, and not .Name)
		if object.Name == "" {
			continue
		}

		res = append(res, &tmp)
		if count > 0 && len(res) >= count {
			break
		}
	}
	return res, nil
}

func (o *GcsFile) Readdir(count int) ([]os.FileInfo, error) {
	fi, err := o.readdir(count)
	if len(fi) > 0 {
		sort.Sort(ByName(fi))
	}

	var res []os.FileInfo
	for _, f := range fi {
		res = append(res, f)
	}
	return res, err
}

func (o *GcsFile) Readdirnames(n int) ([]string, error) {
	fi, err := o.Readdir(n)
	if err != nil && err != io.EOF {
		return nil, err
	}
	names := make([]string, len(fi))

	for i, f := range fi {
		names[i] = f.Name()
	}
	return names, err
}

func (o *GcsFile) Stat() (os.FileInfo, error) {
	o.Sync()
	objAttrs, err := o.resource.obj.Attrs(o.resource.ctx)
	if err != nil {
		if err.Error() == "storage: object doesn't exist" {
			return nil, os.ErrNotExist //works with os.IsNotExist check
		}
		return nil, err
	}
	return &fileInfo{objAttrs, o.resource.fs}, nil
}

func (o *GcsFile) Sync() error {
	return o.resource.maybeCloseIo()
}

func (o *GcsFile) Truncate(wantedSize int64) error {
	if o.closed {
		return ErrFileClosed
	}
	if o.openFlags == os.O_RDONLY {
		return fmt.Errorf("File is opend as read only")
	}
	return o.resource.Truncate(wantedSize)
}

func (o *GcsFile) WriteString(s string) (ret int, err error) {
	return o.Write([]byte(s))
}
