module github.com/spf13/afero/sftpfs

go 1.21.0

replace github.com/spf13/afero => ../

require (
	github.com/pkg/sftp v1.13.7
	github.com/spf13/afero v1.12.0
	golang.org/x/crypto v0.33.0
)

require (
	github.com/kr/fs v0.1.0 // indirect
	golang.org/x/sys v0.30.0 // indirect
	golang.org/x/text v0.22.0 // indirect
)
