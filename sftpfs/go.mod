module github.com/spf13/afero/sftpfs

go 1.23.0

replace github.com/spf13/afero => ../

require (
	github.com/pkg/sftp v1.13.8
	github.com/spf13/afero v1.15.0
	golang.org/x/crypto v0.36.0
)

require (
	github.com/kr/fs v0.1.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.28.0 // indirect
)
