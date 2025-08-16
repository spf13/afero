module github.com/spf13/afero/sftpfs

go 1.24

toolchain go1.24.2

replace github.com/spf13/afero => ../

require (
	github.com/pkg/sftp v1.13.8
	github.com/spf13/afero v1.14.0
	github.com/stretchr/testify v1.10.0
	golang.org/x/crypto v0.36.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/sys v0.31.0 // indirect
	golang.org/x/text v0.25.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
