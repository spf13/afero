<img src="https://cloud.githubusercontent.com/assets/173412/11490338/d50e16dc-97a5-11e5-8b12-019a300d0fcb.png" alt="afero logo-sm"/>


[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/spf13/afero/ci.yaml?branch=master&amp;style=flat-square)](https://github.com/spf13/afero/actions?query=workflow%3ACI)
[![GoDoc](https://pkg.go.dev/badge/mod/github.com/spf13/afero)](https://pkg.go.dev/mod/github.com/spf13/afero)
[![Go Report Card](https://goreportcard.com/badge/github.com/spf13/afero)](https://goreportcard.com/report/github.com/spf13/afero)
![Go Version](https://img.shields.io/badge/go%20version-%3E=1.23-61CFDD.svg?style=flat-square")


# Afero: The Universal Filesystem Abstraction for Go

Afero is a powerful and extensible filesystem abstraction system for Go. It provides a single, unified API for interacting with diverse filesystems—including the local disk, memory, archives, and network storage.

Afero acts as a drop-in replacement for the standard `os` package, enabling you to write modular code that is agnostic to the underlying storage, dramatically simplifies testing, and allows for sophisticated architectural patterns through filesystem composition.

## Why Afero?

Afero elevates filesystem interaction beyond simple file reading and writing, offering solutions for testability, flexibility, and advanced architecture.

🔑 **Key Features:**

*   **Universal API:** Write your code once. Run it against the local OS, in-memory storage, ZIP/TAR archives, or remote systems (SFTP, GCS).
*   **Ultimate Testability:** Utilize `MemMapFs`, a fully concurrent-safe, read/write in-memory filesystem. Write fast, isolated, and reliable unit tests without touching the physical disk or worrying about cleanup.
*   **Powerful Composition:** Afero's hidden superpower. Layer filesystems on top of each other to create sophisticated behaviors:
    *   **Sandboxing:** Use `CopyOnWriteFs` to create temporary scratch spaces that isolate changes from the base filesystem.
    *   **Caching:** Use `CacheOnReadFs` to automatically layer a fast cache (like memory) over a slow backend (like a network drive).
    *   **Security Jails:** Use `BasePathFs` to restrict application access to a specific subdirectory (chroot).
*   **`os` Package Compatibility:** Afero mirrors the functions in the standard `os` package, making adoption and refactoring seamless.
*   **Bidirectional `io/fs` Bridge:** Fully compatible with the Go standard library's `io/fs` interfaces—in both directions. Export any Afero filesystem as an `fs.FS` with `NewIOFS`, or import any `fs.FS` (including `embed.FS`) into Afero with `NewBridgeIOFS`.

## Installation

```bash
go get github.com/spf13/afero
```

```go
import "github.com/spf13/afero"
```

## Quick Start: The Power of Abstraction

The core of Afero is the `afero.Fs` interface. By designing your functions to accept this interface rather than calling `os.*` functions directly, your code instantly becomes more flexible and testable.

### 1. Refactor Your Code

Change functions that rely on the `os` package to accept `afero.Fs`.

```go
// Before: Coupled to the OS and difficult to test
// func ProcessConfiguration(path string) error {
//     data, err := os.ReadFile(path)
//     ...
// }

import "github.com/spf13/afero"

// After: Decoupled, flexible, and testable
func ProcessConfiguration(fs afero.Fs, path string) error {
    // Use Afero utility functions which mirror os/ioutil
    data, err := afero.ReadFile(fs, path)
    // ... process the data
    return err
}
```

### 2. Usage in Production

In your production environment, inject the `OsFs` backend, which wraps the standard operating system calls.

```go
func main() {
    // Use the real OS filesystem
    AppFs := afero.NewOsFs()
    ProcessConfiguration(AppFs, "/etc/myapp.conf")
}
```

### 3. Usage in Testing

In your tests, inject `MemMapFs`. This provides a blazing-fast, isolated, in-memory filesystem that requires no disk I/O and no cleanup.

```go
func TestProcessConfiguration(t *testing.T) {
    // Use the in-memory filesystem
    AppFs := afero.NewMemMapFs()
    
    // Pre-populate the memory filesystem for the test
    configPath := "/test/config.json"
    afero.WriteFile(AppFs, configPath, []byte(`{"feature": true}`), 0644)

    // Run the test entirely in memory
    err := ProcessConfiguration(AppFs, configPath)
    if err != nil {
        t.Fatal(err)
    }
}
```

## Afero's Superpower: Composition

Afero's most unique feature is its ability to combine filesystems. This allows you to build complex behaviors out of simple components, keeping your application logic clean.

### Example 1: Sandboxing with Copy-on-Write

Create a temporary environment where an application can "modify" system files without affecting the actual disk.

```go
// 1. The base layer is the real OS, made read-only for safety.
baseFs := afero.NewReadOnlyFs(afero.NewOsFs())

// 2. The overlay layer is a temporary in-memory filesystem for changes.
overlayFs := afero.NewMemMapFs()

// 3. Combine them. Reads fall through to the base; writes only hit the overlay.
sandboxFs := afero.NewCopyOnWriteFs(baseFs, overlayFs)

// The application can now "modify" /etc/hosts, but the changes are isolated in memory.
afero.WriteFile(sandboxFs, "/etc/hosts", []byte("127.0.0.1 sandboxed-app"), 0644)

// The real /etc/hosts on disk is untouched.
```

### Example 2: Caching a Slow Filesystem

Improve performance by layering a fast cache (like memory) over a slow backend (like a network drive or cloud storage).

```go
import "time"

// Assume 'remoteFs' is a slow backend (e.g., SFTP or GCS)
var remoteFs afero.Fs 

// 'cacheFs' is a fast in-memory backend
cacheFs := afero.NewMemMapFs()

// Create the caching layer. Cache items for 5 minutes upon first read.
cachedFs := afero.NewCacheOnReadFs(remoteFs, cacheFs, 5*time.Minute)

// The first read is slow (fetches from remote, then caches)
data1, _ := afero.ReadFile(cachedFs, "data.json")

// The second read is instant (serves from memory cache)
data2, _ := afero.ReadFile(cachedFs, "data.json")
```

### Example 3: Security Jails (chroot)

Restrict an application component's access to a specific subdirectory.

```go
osFs := afero.NewOsFs()

// Create a filesystem rooted at /home/user/public
// The application cannot access anything above this directory.
jailedFs := afero.NewBasePathFs(osFs, "/home/user/public")

// To the application, this is reading "/"
// In reality, it's reading "/home/user/public/"
dirInfo, err := afero.ReadDir(jailedFs, "/")

// Attempts to access parent directories fail
_, err = jailedFs.Open("../secrets.txt") // Returns an error
```

## Real-World Use Cases

### Build Cloud-Agnostic Applications

Write applications that seamlessly work with different storage backends:

```go
type DocumentProcessor struct {
    fs afero.Fs
}

func NewDocumentProcessor(fs afero.Fs) *DocumentProcessor {
    return &DocumentProcessor{fs: fs}
}

func (p *DocumentProcessor) Process(inputPath, outputPath string) error {
    // This code works whether fs is local disk, cloud storage, or memory
    content, err := afero.ReadFile(p.fs, inputPath)
    if err != nil {
        return err
    }
    
    processed := processContent(content)
    return afero.WriteFile(p.fs, outputPath, processed, 0644)
}

// Use with local filesystem
processor := NewDocumentProcessor(afero.NewOsFs())

// Use with Google Cloud Storage
processor := NewDocumentProcessor(gcsFS)

// Use with in-memory filesystem for testing
processor := NewDocumentProcessor(afero.NewMemMapFs())
```

### Treating Archives as Filesystems

Read files directly from `.zip` or `.tar` archives without unpacking them to disk first.

```go
import (
    "archive/zip"
    "github.com/spf13/afero/zipfs"
)

// Assume 'zipReader' is a *zip.Reader initialized from a file or memory
var zipReader *zip.Reader 

// Create a read-only ZipFs
archiveFS := zipfs.New(zipReader)

// Read a file from within the archive using the standard Afero API
content, err := afero.ReadFile(archiveFS, "/docs/readme.md")
```

### Serving Any Filesystem over HTTP

Use `HttpFs` to expose any Afero filesystem—even one created dynamically in memory—through a standard Go web server.

```go
import (
    "net/http"
    "github.com/spf13/afero"
)

func main() {
    memFS := afero.NewMemMapFs()
    afero.WriteFile(memFS, "index.html", []byte("<h1>Hello from Memory!</h1>"), 0644)

    // Wrap the memory filesystem to make it compatible with http.FileServer.
    httpFS := afero.NewHttpFs(memFS)

    http.Handle("/", http.FileServer(httpFS.Dir("/")))
    http.ListenAndServe(":8080", nil)
}
```

### Layering Embedded Assets with Copy-on-Write

One of the most powerful patterns unlocked by `NewBridgeIOFS`: use Go's native `//go:embed` assets as the read-only base of a `CopyOnWriteFs`. Your application gets the embedded defaults at startup, while any runtime writes are safely isolated in memory.

```go
import (
    "embed"
    "github.com/spf13/afero"
)

//go:embed configs/*
var embeddedConfigs embed.FS

func NewAppFs() afero.Fs {
    // Import the read-only embedded assets into Afero.
    base := afero.NewBridgeIOFS(embeddedConfigs)

    // Layer a writable in-memory filesystem on top.
    // Reads fall through to the embedded assets; writes stay in memory.
    overlay := afero.NewMemMapFs()
    return afero.NewCopyOnWriteFs(base, overlay)
}

func main() {
    fs := NewAppFs()

    // Reads the embedded default — no disk access.
    cfg, _ := afero.ReadFile(fs, "configs/default.yaml")

    // Writes go to the in-memory overlay; embedded assets are untouched.
    afero.WriteFile(fs, "configs/default.yaml", []byte("override: true"), 0644)
}
```

### Testing Made Simple

One of Afero's greatest strengths is making filesystem-dependent code easily testable:

```go
func SaveUserData(fs afero.Fs, userID string, data []byte) error {
    filename := fmt.Sprintf("users/%s.json", userID)
    return afero.WriteFile(fs, filename, data, 0644)
}

func TestSaveUserData(t *testing.T) {
    // Create a clean, fast, in-memory filesystem for testing
    testFS := afero.NewMemMapFs()
    
    userData := []byte(`{"name": "John", "email": "john@example.com"}`)
    err := SaveUserData(testFS, "123", userData)
    
    if err != nil {
        t.Fatalf("SaveUserData failed: %v", err)
    }
    
    // Verify the file was saved correctly
    saved, err := afero.ReadFile(testFS, "users/123.json")
    if err != nil {
        t.Fatalf("Failed to read saved file: %v", err)
    }
    
    if string(saved) != string(userData) {
        t.Errorf("Data mismatch: got %s, want %s", saved, userData)
    }
}
```

**Benefits of testing with Afero:**
- ⚡ **Fast** - No disk I/O, tests run in memory
- 🔄 **Reliable** - Each test starts with a clean slate
- 🧹 **No cleanup** - Memory is automatically freed
- 🔒 **Safe** - Can't accidentally modify real files
- 🏃 **Parallel** - Tests can run concurrently without conflicts

## Backend Reference

| Type | Backend | Constructor | Description | Status |
| :--- | :--- | :--- | :--- | :--- |
| **Core** | **OsFs** | `afero.NewOsFs()` | Interacts with the real operating system filesystem. Use in production. | ✅ Official |
| | **MemMapFs** | `afero.NewMemMapFs()` | A fast, atomic, concurrent-safe, in-memory filesystem. Ideal for testing. | ✅ Official |
| **Composition** | **CopyOnWriteFs**| `afero.NewCopyOnWriteFs(base, overlay)` | A read-only base with a writable overlay. Ideal for sandboxing. | ✅ Official |
| | **CacheOnReadFs**| `afero.NewCacheOnReadFs(base, cache, ttl)` | Lazily caches files from a slow base into a fast layer on first read. | ✅ Official |
| | **BasePathFs** | `afero.NewBasePathFs(source, path)` | Restricts operations to a subdirectory (chroot/jail). | ✅ Official |
| | **ReadOnlyFs** | `afero.NewReadOnlyFs(source)` | Provides a read-only view, preventing any modifications. | ✅ Official |
| | **RegexpFs** | `afero.NewRegexpFs(source, regexp)` | Filters a filesystem, only showing files that match a regex. | ✅ Official |
| **Utility** | **HttpFs** | `afero.NewHttpFs(source)` | Wraps any Afero filesystem to be served via `http.FileServer`. | ✅ Official |
| **`io/fs` Bridge** | **IOFS** | `afero.NewIOFS(fs)` | Exports any Afero `Fs` as a standard `io/fs.FS` (read-only view). | ✅ Official |
| | **BridgeIOFS** | `afero.NewBridgeIOFS(fsys)` | Imports any `io/fs.FS` (e.g. `embed.FS`) as a read-only Afero `Fs`. | ✅ Official |
| **Archives** | **ZipFs** | `zipfs.New(zipReader)` | Read-only access to files within a ZIP archive. | ✅ Official |
| | **TarFs** | `tarfs.New(tarReader)` | Read-only access to files within a TAR archive. | ✅ Official |
| **Network** | **GcsFs** | `gcsfs.NewGcsFs(...)` | Google Cloud Storage backend. | ⚡ Experimental |
| | **SftpFs** | `sftpfs.New(...)` | SFTP backend. | ⚡ Experimental |
| **3rd Party Cloud** | **S3Fs** | [`fclairamb/afero-s3`](https://github.com/fclairamb/afero-s3) | Production-ready S3 backend built on official AWS SDK. | 🔹 3rd Party |
| | **MinioFs** | [`cpyun/afero-minio`](https://github.com/cpyun/afero-minio) | MinIO object storage backend with S3 compatibility. | 🔹 3rd Party |
| | **DriveFs** | [`fclairamb/afero-gdrive`](https://github.com/fclairamb/afero-gdrive) | Google Drive backend with streaming support. | 🔹 3rd Party |
| | **DropboxFs** | [`fclairamb/afero-dropbox`](https://github.com/fclairamb/afero-dropbox) | Dropbox backend with streaming support. | 🔹 3rd Party |
| **3rd Party Specialized** | **GitFs** | [`tobiash/go-gitfs`](https://github.com/tobiash/go-gitfs) | Git repository filesystem (read-only, Afero compatible). | 🔹 3rd Party |
| | **DockerFs** | [`unmango/aferox`](https://github.com/unmango/aferox) | Docker container filesystem access. | 🔹 3rd Party |
| | **GitHubFs** | [`unmango/aferox`](https://github.com/unmango/aferox) | GitHub repository and releases filesystem. | 🔹 3rd Party |
| | **FilterFs** | [`unmango/aferox`](https://github.com/unmango/aferox) | Filesystem filtering with predicates. | 🔹 3rd Party |
| | **IgnoreFs** | [`unmango/aferox`](https://github.com/unmango/aferox) | .gitignore-aware filtering filesystem. | 🔹 3rd Party |
| | **FUSEFs** | [`JakWai01/sile-fystem`](https://github.com/JakWai01/sile-fystem) | Generic FUSE implementation using any Afero backend. | 🔹 3rd Party |

## Afero & the Go Standard Library (`io/fs`)

Go 1.16 introduced `io/fs`, a standard abstraction for **read-only** filesystems. Afero bridges this gap in **both directions**, so you can freely move between the Afero and standard library worlds.

### When to use which

| Situation | Recommendation |
| :--- | :--- |
| You only need to **read** files and want strict stdlib compatibility | Use `io/fs` directly |
| You need to **create, write, modify, or delete** files | Use Afero |
| You want to test complex read/write interactions | Use Afero (`MemMapFs`) |
| You need Copy-on-Write, Caching, or other composition | Use Afero |
| You have an `io/fs.FS` source and want Afero's full API | Use `NewBridgeIOFS` |

### Exporting Afero → `io/fs.FS` with `NewIOFS`

Wrap any Afero filesystem to satisfy the `fs.FS` interface. This lets you pass a writable Afero filesystem to any standard library function that accepts `fs.FS`.

```go
import (
    "io/fs"
    "github.com/spf13/afero"
)

// A writable Afero filesystem
myAferoFs := afero.NewMemMapFs()
afero.WriteFile(myAferoFs, "hello.txt", []byte("hello"), 0644)

// Export as a read-only standard library fs.FS
var myIoFs fs.FS = afero.NewIOFS(myAferoFs)

// Now compatible with fs.WalkDir, http.FS, template.ParseFS, etc.
fs.WalkDir(myIoFs, ".", func(path string, d fs.DirEntry, err error) error {
    // ...
    return nil
})
```

### Importing `io/fs.FS` → Afero with `NewBridgeIOFS`

Bring any `io/fs.FS` source—such as Go's built-in `embed.FS`—into Afero as a read-only `Fs`. This makes it composable with all of Afero's layer and filter types.

```go
import (
    "embed"
    "github.com/spf13/afero"
)

//go:embed assets/*
var embeddedAssets embed.FS

// Import the embedded assets into Afero (read-only).
// Write operations return os.ErrPermission.
embeddedFs := afero.NewBridgeIOFS(embeddedAssets)

// Now use it anywhere an afero.Fs is expected—including composition.
// For example, layer a writable overlay on top for testing:
overlay := afero.NewMemMapFs()
testFs := afero.NewCopyOnWriteFs(embeddedFs, overlay)
```

## Third-Party Backends & Ecosystem

The Afero community has developed numerous backends and tools that extend the library's capabilities. Below are curated, well-maintained options organized by maturity and reliability.

### Featured Community Backends

These are mature, reliable backends that we can confidently recommend for production use:

#### **Amazon S3** - [`fclairamb/afero-s3`](https://github.com/fclairamb/afero-s3)
Production-ready S3 backend built on the official AWS SDK for Go.

```go
import "github.com/fclairamb/afero-s3"

s3fs := s3.NewFs(bucket, session)
```

#### **MinIO** - [`cpyun/afero-minio`](https://github.com/cpyun/afero-minio)
MinIO object storage backend providing S3-compatible object storage with deduplication and optimization features.

```go
import "github.com/cpyun/afero-minio"

minioFs := miniofs.NewMinioFs(ctx, "minio://endpoint/bucket")
```

### Community & Specialized Backends

#### Cloud Storage

- **Google Drive** - [`fclairamb/afero-gdrive`](https://github.com/fclairamb/afero-gdrive)  
  Streaming support; no write-seeking or POSIX permissions; no files listing cache

- **Dropbox** - [`fclairamb/afero-dropbox`](https://github.com/fclairamb/afero-dropbox)  
  Streaming support; no write-seeking or POSIX permissions

#### Version Control Systems

- **Git Repositories** - [`tobiash/go-gitfs`](https://github.com/tobiash/go-gitfs)  
  Read-only filesystem abstraction for Git repositories. Works with bare repositories and provides filesystem view of any git reference. Uses go-git for repository access.

#### Container and Remote Systems

- **Docker Containers** - [`unmango/aferox`](https://github.com/unmango/aferox)  
  Access Docker container filesystems as if they were local filesystems

- **GitHub API** - [`unmango/aferox`](https://github.com/unmango/aferox)  
  Turn GitHub repositories, releases, and assets into browsable filesystems

#### FUSE Integration

- **Generic FUSE** - [`JakWai01/sile-fystem`](https://github.com/JakWai01/sile-fystem)  
  Mount any Afero filesystem as a FUSE filesystem, allowing any Afero backend to be used as a real mounted filesystem

#### Specialized Filesystems

- **FAT32 Support** - [`aligator/GoFAT`](https://github.com/aligator/GoFAT)  
  Pure Go FAT filesystem implementation (currently read-only)

### Interface Adapters & Utilities

**Cross-Interface Compatibility:**
- [`jfontan/go-billy-desfacer`](https://github.com/jfontan/go-billy-desfacer) - Adapter between Afero and go-billy interfaces (for go-git compatibility)
- [`Maldris/go-billy-afero`](https://github.com/Maldris/go-billy-afero) - Alternative wrapper for using Afero with go-billy
- [`c4milo/afero2billy`](https://github.com/c4milo/afero2billy) - Another Afero to billy filesystem adapter

**Working Directory Management:**
- [`carolynvs/aferox`](https://github.com/carolynvs/aferox) - Working directory-aware filesystem wrapper

**Advanced Filtering:**
- [`unmango/aferox`](https://github.com/unmango/aferox) includes multiple specialized filesystems:
  - **FilterFs** - Predicate-based file filtering
  - **IgnoreFs** - .gitignore-aware filtering
  - **WriterFs** - Dump writes to io.Writer for debugging

#### Developer Tools & Utilities

**nhatthm Utility Suite** - Essential tools for Afero development:
- [`nhatthm/aferocopy`](https://github.com/nhatthm/aferocopy) - Copy files between any Afero filesystems
- [`nhatthm/aferomock`](https://github.com/nhatthm/aferomock) - Mocking toolkit for testing
- [`nhatthm/aferoassert`](https://github.com/nhatthm/aferoassert) - Assertion helpers for filesystem testing

### Ecosystem Showcase

**Windows Virtual Drives** - [`balazsgrill/potatodrive`](https://github.com/balazsgrill/potatodrive)  
Mount any Afero filesystem as a Windows drive letter. Brilliant demonstration of Afero's power!

### Modern Asset Embedding (Go 1.16+)

Instead of third-party tools, use Go's native `//go:embed` with Afero via `NewBridgeIOFS`:

```go
import (
    "embed"
    "github.com/spf13/afero"
)

//go:embed assets/*
var assetsFS embed.FS

func main() {
    // Import the embedded assets into Afero as a read-only filesystem.
    fs := afero.NewBridgeIOFS(assetsFS)

    // Use the full Afero API against the embedded files.
    content, _ := afero.ReadFile(fs, "assets/config.json")
}
```

## Contributing

We welcome contributions! The project is mature, but we are actively looking for contributors to help implement and stabilize network/cloud backends.

* 🔥 **Microsoft Azure Blob Storage**  
* 🔒 **Modern Encryption Backend** - Built on secure, contemporary crypto (not legacy EncFS)  
* 🐙 **Canonical go-git Adapter** - Unified solution for Git integration  
* 📡 **SSH/SCP Backend** - Secure remote file operations  
*  Stabilization of existing experimental backends (GCS, SFTP)

To contribute:
1. Fork the repository
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Commit your changes (`git commit -am 'Add some feature'`)
4. Push to the branch (`git push origin my-new-feature`)
5. Create a new Pull Request

## 📄 License

Afero is released under the Apache 2.0 license. See [LICENSE.txt](https://github.com/spf13/afero/blob/master/LICENSE.txt) for details.

## 🔗 Additional Resources

- [📖 Full API Documentation](https://pkg.go.dev/github.com/spf13/afero)
- [🎯 Examples Repository](https://github.com/spf13/afero/tree/master/examples)
- [📋 Release Notes](https://github.com/spf13/afero/releases)
- [❓ GitHub Discussions](https://github.com/spf13/afero/discussions)

---

*Afero comes from the Latin roots Ad-Facere, meaning "to make" or "to do" - fitting for a library that empowers you to make and do amazing things with filesystems.*
