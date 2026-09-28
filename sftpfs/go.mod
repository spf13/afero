module github.com/spf13/afero/sftpfs

go 1.26.0

replace github.com/spf13/afero => ../

require (
	github.com/pkg/sftp v1.13.11
	github.com/spf13/afero v1.15.0
	golang.org/x/crypto v0.57.0
)

require (
	github.com/kr/fs v0.1.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
)
