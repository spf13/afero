package gcsfs

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestTestMainExitStatus(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	credentials, err := filepath.Abs("gcs-fake-service-account.json")
	if err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name        string
		credentials string
		wantExit    int
	}{
		{"pass", credentials, 0},
		{"fail", credentials, 1},
		{"setup-panic", filepath.Join(t.TempDir(), "missing-credentials.json"), 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("AFERO_GCS_TEST_MAIN_HELPER", tc.name)
			t.Setenv("GOOGLE_APPLICATION_CREDENTIALS", tc.credentials)
			cmd := exec.Command(executable, "-test.run=^TestTestMainExitStatusHelper$")
			output, err := cmd.CombinedOutput()
			exitCode := 0
			if err != nil {
				var exitErr *exec.ExitError
				if !errors.As(err, &exitErr) {
					t.Fatalf("could not run test process: %v\n%s", err, output)
				}
				exitCode = exitErr.ExitCode()
			}
			if exitCode != tc.wantExit {
				t.Fatalf("test process exited with %d, want %d\n%s", exitCode, tc.wantExit, output)
			}
		})
	}
}

func TestTestMainExitStatusHelper(t *testing.T) {
	switch os.Getenv("AFERO_GCS_TEST_MAIN_HELPER") {
	case "pass":
	case "fail":
		t.Fatal("intentional test failure")
	default:
		t.Skip("subprocess helper")
	}
}
