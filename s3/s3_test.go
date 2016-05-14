package s3

import (
	"flag"
	"log"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/spf13/afero/test"
)

var s3bucket string

func init() {
	flag.StringVar(&s3bucket, "s3bucket", "", "s3 bucket to use for testing")
	flag.Parse()

	if s3bucket == "" {
		log.Fatal("must provide -s3bucket for testing this package")
	}
	for _, requiredEnv := range []string{"AWS_REGION", "AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"} {
		if os.Getenv(requiredEnv) == "" {
			log.Fatalf("must set %s for testing this package", requiredEnv)
		}
	}
}

func TestRead0(t *testing.T) {
	t.Skip("TODO: S3 FS doesn't support nonzero offset read/writes")
	test.Read0(t, NewS3Fs(s3bucket, s3.New(session.New())))
}

func TestOpenFile(t *testing.T) {
	t.Skip("S3 doesn't support appending, which this test performs")
	test.OpenFile(t, NewS3Fs(s3bucket, s3.New(session.New())))
}

func TestCreate(t *testing.T) {
	test.Create(t, NewS3Fs(s3bucket, s3.New(session.New())))
}

func TestRename(t *testing.T) {
	test.Rename(t, NewS3Fs(s3bucket, s3.New(session.New())))
}

func TestRemove(t *testing.T) {
	test.Remove(t, NewS3Fs(s3bucket, s3.New(session.New())))
}

func TestTruncate(t *testing.T) {
	t.Skip("TODO: implement Truncate")
	test.Truncate(t, NewS3Fs(s3bucket, s3.New(session.New())))
}

func TestReadWriteSeek(t *testing.T) {
	t.Skip("TODO: S3 FS doesn't support nonzero offset read/writes")
	test.ReadWriteSeek(t, NewS3Fs(s3bucket, s3.New(session.New())))
}

func TestSeek(t *testing.T) {
	t.Skip("TODO: S3 FS doesn't support whence==2 seeks")
	test.Seek(t, NewS3Fs(s3bucket, s3.New(session.New())))
}

func TestReadAt(t *testing.T) {
	t.Skip("TODO: S3 FS doesn't support nonzero offset read/writes")
	test.ReadAt(t, NewS3Fs(s3bucket, s3.New(session.New())))
}

func TestWriteAt(t *testing.T) {
	t.Skip("TODO: S3 FS doesn't support nonzero offset read/writes")
	test.WriteAt(t, NewS3Fs(s3bucket, s3.New(session.New())))
}

func TestReaddirnames(t *testing.T) {
	test.Readdirnames(t, NewS3Fs(s3bucket, s3.New(session.New())))
}

func TestReaddirSimple(t *testing.T) {
	test.ReaddirSimple(t, NewS3Fs(s3bucket, s3.New(session.New())))
}

func TestReaddirAll(t *testing.T) {
	test.ReaddirAll(t, NewS3Fs(s3bucket, s3.New(session.New())))
}

func TestStatDirectory(t *testing.T) {
	test.StatDirectory(t, NewS3Fs(s3bucket, s3.New(session.New())))
}

func TestStatFile(t *testing.T) {
	test.StatFile(t, NewS3Fs(s3bucket, s3.New(session.New())))
}
