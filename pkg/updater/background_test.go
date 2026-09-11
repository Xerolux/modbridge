package updater

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestUpdateSurvivesRequestCancellationAndPreservesBackup(t *testing.T) {
	for _, validChecksum := range []bool{true, false} {
		t.Run(fmt.Sprint(validChecksum), func(t *testing.T) {
			target := filepath.Join(t.TempDir(), "modbridge")
			if err := os.WriteFile(target, []byte("previous"), 0755); err != nil {
				t.Fatal(err)
			}
			payload := []byte("replacement-test-binary")
			started, proceed := make(chan struct{}), make(chan struct{})
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/binary" {
					close(started)
					<-proceed
					w.Write(payload)
					return
				}
				sum := sha256.Sum256(payload)
				if !validChecksum {
					sum = sha256.Sum256([]byte("wrong"))
				}
				fmt.Fprintf(w, "%x  modbridge-linux-amd64\n", sum)
			}))
			defer srv.Close()
			u := New("test/repo", BuildInfo{Version: "1.0.0", OS: "linux", Arch: "amd64"})
			u.executablePath = func() (string, error) { return target, nil }
			u.cachedRelease = &ReleaseInfo{TagName: "v3.0.0", Assets: []Asset{{Name: "modbridge-linux-amd64", URL: srv.URL + "/binary", Size: int64(len(payload))}, {Name: "checksums.txt", URL: srv.URL + "/checksums"}}}
			u.cacheExpiry = time.Now().Add(time.Minute)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if err := u.PerformUpdate(ctx); err != nil {
				t.Fatal(err)
			}
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				close(proceed)
				t.Fatal("download did not start")
			}
			cancel()
			if err := u.PerformUpdate(context.Background()); err != ErrUpdateInProgress {
				close(proceed)
				t.Fatalf("concurrent update: %v", err)
			}
			close(proceed)
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				state := u.GetStatus().State
				if state == StateDone || state == StateError {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			got, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if validChecksum {
				if u.GetStatus().State != StateDone || string(got) != string(payload) {
					t.Fatalf("update failed: %+v, %s", u.GetStatus(), got)
				}
				backups, _ := filepath.Glob(target + ".bak.*")
				if len(backups) != 1 {
					t.Fatal("missing backup")
				}
				old, _ := os.ReadFile(backups[0])
				if string(old) != "previous" {
					t.Fatal("backup differs")
				}
			} else if u.GetStatus().State != StateError || string(got) != "previous" {
				t.Fatal("checksum failure replaced original")
			}
		})
	}
}
