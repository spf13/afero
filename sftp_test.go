// Copyright © 2015 Jerry Jacobs <jerry.jacobs@xor-gate.org>.
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
	"os"
	"io/ioutil"
	"testing"

	"github.com/xor-gate/afero"

	"golang.org/x/crypto/ssh"
	"github.com/pkg/sftp"
)

type SftpFsContext struct {
	sshc   *ssh.Client
	sshcfg *ssh.ClientConfig
	sftpc  *sftp.Client
}

func SftpConnect(user string, host string) (*SftpFsContext, error) {
	pemBytes, err := ioutil.ReadFile(os.Getenv("HOME") + "/.ssh/id_rsa")
	if err != nil {
		return nil,err
	}

	signer, err := ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		return nil,err
	}

	sshcfg := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
	}

	sshc, err := ssh.Dial("tcp", host, sshcfg)
	if err != nil {
		return nil,err
	}

	sftpc, err := sftp.NewClient(sshc)
	if err != nil {
		return nil,err
	}

	ctx := &SftpFsContext{
		sshc: sshc,
		sshcfg: sshcfg,
		sftpc: sftpc,
	}

	return ctx,nil
}

func (ctx *SftpFsContext) Disconnect() error {
	ctx.sftpc.Close()
	ctx.sshc.Close()
	return nil
}

func TestSftpCreate(t *testing.T) {
	ctx, err := SftpConnect("user", "host:port")
	if err != nil {
		t.Fatal(err)
	}
	defer ctx.Disconnect()

	var AppFs afero.Fs = afero.SftpFs{
		SftpClient: ctx.sftpc,
	}

	file, err := AppFs.Create("aferoSftpFsTestFile")
	if err != nil {
		t.Error(err)
	}
	defer file.Close()
}
