package afero

import (
    "context"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "time"

    "github.com/google/go-github/v62/github"
)

// GitHubFS implements afero.Fs for a GitHub repository or a base filesystem
type GitHubFS struct {
    client *github.Client
    owner  string
    repo   string
    branch string // Optional, defaults to "main"
    token  string // Optional, for authenticated requests
    base   Fs     // Optional base filesystem for testing
}

// NewGitHubFS creates a new GitHubFS instance for a GitHub repository
func NewGitHubFS(owner, repo, branch, token string) *GitHubFS {
    client := github.NewClient(nil)
    if token != "" {
        client = github.NewClient(nil).WithAuthToken(token)
    }
    if branch == "" {
        branch = "main" // Default branch
    }
    return &GitHubFS{
        client: client,
        owner:  owner,
        repo:   repo,
        branch: branch,
        token:  token,
    }
}

// NewGitHubFSWithBase creates a GitHubFS instance with a base filesystem for testing
func NewGitHubFSWithBase(base Fs) *GitHubFS {
    return &GitHubFS{
        base: base,
    }
}

// Open opens a file from the base filesystem or GitHub repository
func (fs *GitHubFS) Open(name string) (File, error) {
    cleanPath := filepath.Clean(name)
    if cleanPath == "." || cleanPath == "/" || cleanPath == "" {
        cleanPath = "" // Normalize root path
    }

    // Use base filesystem if provided (for testing)
    if fs.base != nil {
        return fs.base.Open(cleanPath)
    }

    // Otherwise, fetch from GitHub
    ctx := context.Background()
    fileContent, dirContent, resp, err := fs.client.Repositories.GetContents(ctx, fs.owner, fs.repo, cleanPath, &github.RepositoryContentGetOptions{Ref: fs.branch})
    if err != nil {
        return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
    }
    if resp.StatusCode != 200 {
        return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
    }

    if fileContent != nil {
        content, err := fileContent.GetContent()
        if err != nil {
            return nil, err
        }
        return &GitHubFile{fs: fs, path: cleanPath, content: []byte(content)}, nil
    }
    if dirContent != nil {
        return &GitHubFile{fs: fs, path: cleanPath, isDir: true, dirEntries: dirContent}, nil
    }
    return nil, &os.PathError{Op: "open", Path: name, Err: os.ErrNotExist}
}

// Stat returns file info for a path
func (fs *GitHubFS) Stat(name string) (os.FileInfo, error) {
    file, err := fs.Open(name)
    if err != nil {
        return nil, err
    }
    defer file.Close()
    return file.Stat()
}

// Name returns the filesystem name
func (fs *GitHubFS) Name() string { return "GitHubFS" }

// Create is not supported (read-only)
func (fs *GitHubFS) Create(name string) (File, error) {
    return nil, fmt.Errorf("GitHubFS is read-only")
}

// Mkdir is not supported (read-only)
func (fs *GitHubFS) Mkdir(name string, perm os.FileMode) error {
    return fmt.Errorf("GitHubFS is read-only")
}

// MkdirAll is not supported (read-only)
func (fs *GitHubFS) MkdirAll(path string, perm os.FileMode) error {
    return fmt.Errorf("GitHubFS is read-only")
}

// OpenFile is not supported for writing (read-only)
func (fs *GitHubFS) OpenFile(name string, flag int, perm os.FileMode) (File, error) {
    if flag&(os.O_WRONLY|os.O_RDWR|os.O_CREATE|os.O_TRUNC) != 0 {
        return nil, fmt.Errorf("GitHubFS is read-only")
    }
    return fs.Open(name)
}

// Remove is not supported (read-only)
func (fs *GitHubFS) Remove(name string) error {
    return fmt.Errorf("GitHubFS is read-only")
}

// RemoveAll is not supported (read-only)
func (fs *GitHubFS) RemoveAll(path string) error {
    return fmt.Errorf("GitHubFS is read-only")
}

// Rename is not supported (read-only)
func (fs *GitHubFS) Rename(oldname, newname string) error {
    return fmt.Errorf("GitHubFS is read-only")
}

// Chmod is not supported (read-only)
func (fs *GitHubFS) Chmod(name string, mode os.FileMode) error {
    return fmt.Errorf("GitHubFS is read-only")
}

// Chtimes is not supported (read-only)
func (fs *GitHubFS) Chtimes(name string, atime, mtime time.Time) error {
    return fmt.Errorf("GitHubFS is read-only")
}

// GitHubFile represents a file or directory in GitHubFS
type GitHubFile struct {
    fs         *GitHubFS
    path       string
    content    []byte
    isDir      bool
    dirEntries []*github.RepositoryContent
    offset     int64
}

// Close does nothing (no resources to release)
func (f *GitHubFile) Close() error { return nil }

// Read reads file content
func (f *GitHubFile) Read(b []byte) (int, error) {
    if f.isDir {
        return 0, fmt.Errorf("cannot read from directory")
    }
    if f.offset >= int64(len(f.content)) {
        return 0, io.EOF
    }
    n := copy(b, f.content[f.offset:])
    f.offset += int64(n)
    return n, nil
}

// ReadAt reads file content at an offset
func (f *GitHubFile) ReadAt(b []byte, off int64) (int, error) {
    if f.isDir {
        return 0, fmt.Errorf("cannot read from directory")
    }
    if off >= int64(len(f.content)) {
        return 0, io.EOF
    }
    n := copy(b, f.content[off:])
    return n, nil
}

// Seek adjusts the read offset
func (f *GitHubFile) Seek(offset int64, whence int) (int64, error) {
    if f.isDir {
        return 0, fmt.Errorf("cannot seek in directory")
    }
    switch whence {
    case io.SeekStart:
        f.offset = offset
    case io.SeekCurrent:
        f.offset += offset
    case io.SeekEnd:
        f.offset = int64(len(f.content)) + offset
    }
    if f.offset < 0 {
        f.offset = 0
    }
    if f.offset > int64(len(f.content)) {
        f.offset = int64(len(f.content))
    }
    return f.offset, nil
}

// Write is not supported (read-only)
func (f *GitHubFile) Write(b []byte) (int, error) {
    return 0, fmt.Errorf("GitHubFS is read-only")
}

// WriteAt is not supported (read-only)
func (f *GitHubFile) WriteAt(b []byte, off int64) (int, error) {
    return 0, fmt.Errorf("GitHubFS is read-only")
}

// Name returns the file or directory name
func (f *GitHubFile) Name() string {
    return filepath.Base(f.path)
}

// Readdir reads directory entries
func (f *GitHubFile) Readdir(count int) ([]os.FileInfo, error) {
    if !f.isDir {
        return nil, fmt.Errorf("not a directory")
    }
    var infos []os.FileInfo
    for _, entry := range f.dirEntries {
        if entry.Name == nil || entry.Type == nil || entry.Size == nil {
            continue // Skip invalid entries
        }
        infos = append(infos, &GitHubFileInfo{
            name:    *entry.Name,
            size:    int64(*entry.Size),
            isDir:   *entry.Type == "dir",
            modTime: time.Now(), // Placeholder, GitHub API lacks mod time
        })
    }
    if count <= 0 || count >= len(infos) {
        return infos, nil
    }
    return infos[:count], nil
}

// Readdirnames reads directory entry names
func (f *GitHubFile) Readdirnames(count int) ([]string, error) {
    infos, err := f.Readdir(count)
    if err != nil {
        return nil, err
    }
    names := make([]string, len(infos))
    for i, info := range infos {
        names[i] = info.Name()
    }
    return names, nil
}

// Stat returns file info
func (f *GitHubFile) Stat() (os.FileInfo, error) {
    if f.isDir {
        return &GitHubFileInfo{name: f.Name(), isDir: true}, nil
    }
    return &GitHubFileInfo{name: f.Name(), size: int64(len(f.content))}, nil
}

// Sync is not supported (read-only)
func (f *GitHubFile) Sync() error { return fmt.Errorf("GitHubFS is read-only") }

// Truncate is not supported (read-only)
func (f *GitHubFile) Truncate(size int64) error { return fmt.Errorf("GitHubFS is read-only") }

// WriteString is not supported (read-only)
func (f *GitHubFile) WriteString(s string) (int, error) {
    return 0, fmt.Errorf("GitHubFS is read-only")
}

// GitHubFileInfo implements os.FileInfo for GitHubFS
type GitHubFileInfo struct {
    name    string
    size    int64
    isDir   bool
    modTime time.Time
}

func (i *GitHubFileInfo) Name() string       { return i.name }
func (i *GitHubFileInfo) Size() int64        { return i.size }
func (i *GitHubFileInfo) Mode() os.FileMode {
    if i.isDir {
        return os.ModeDir | 0755
    }
    return 0644
}
func (i *GitHubFileInfo) ModTime() time.Time { return i.modTime }
func (i *GitHubFileInfo) IsDir() bool        { return i.isDir }
func (i *GitHubFileInfo) Sys() interface{}   { return nil }