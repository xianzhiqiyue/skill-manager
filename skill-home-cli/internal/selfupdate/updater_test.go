package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdaterUpdateLatestInstallsBinaryAndCreatesBackup(t *testing.T) {
	t.Parallel()

	currentExec := filepath.Join(t.TempDir(), "skill-home")
	if err := os.WriteFile(currentExec, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("WriteFile(currentExec) failed: %v", err)
	}

	assetName := "skill-home-linux-amd64.tar.gz"
	archiveData := createTarGzArchive(t, "skill-home", []byte("new-binary"))
	checksums := fmt.Sprintf("%s  %s\n", sha256Hex(archiveData), assetName)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/test/repo/releases/latest":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"tag_name":"v1.2.3"}`)
		case "/test/repo/releases/download/v1.2.3/" + assetName:
			w.Write(archiveData)
		case "/test/repo/releases/download/v1.2.3/checksums.txt":
			fmt.Fprint(w, checksums)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	updater := Updater{
		Repo:            "test/repo",
		CurrentVersion:  "v1.0.0",
		ExecutablePath:  currentExec,
		GOOS:            "linux",
		GOARCH:          "amd64",
		APIBaseURL:      server.URL,
		DownloadBaseURL: server.URL,
		Client:          server.Client(),
	}

	version, err := updater.Update("")
	if err != nil {
		t.Fatalf("Update returned error: %v", err)
	}
	if version != "v1.2.3" {
		t.Fatalf("version = %q, want %q", version, "v1.2.3")
	}

	assertFileContent(t, currentExec, "new-binary")
	assertFileContent(t, currentExec+".bak", "old-binary")

	info, err := os.Stat(currentExec)
	if err != nil {
		t.Fatalf("Stat(currentExec) failed: %v", err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Fatalf("expected executable mode, got %v", info.Mode())
	}
}

func TestInstallUpdatedBinaryRollsBackOnCopyFailure(t *testing.T) {
	t.Parallel()

	currentExec := filepath.Join(t.TempDir(), "skill-home")
	if err := os.WriteFile(currentExec, []byte("old-binary"), 0o755); err != nil {
		t.Fatalf("WriteFile(currentExec) failed: %v", err)
	}

	err := installUpdatedBinary(filepath.Join(t.TempDir(), "missing-binary"), currentExec)
	if err == nil {
		t.Fatal("installUpdatedBinary expected error, got nil")
	}

	assertFileContent(t, currentExec, "old-binary")
	if _, statErr := os.Stat(currentExec + ".bak"); !os.IsNotExist(statErr) {
		t.Fatalf("backup should be cleaned up after rollback, stat err = %v", statErr)
	}
}

func createTarGzArchive(t *testing.T, fileName string, content []byte) []byte {
	t.Helper()

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	tarWriter := tar.NewWriter(gzipWriter)

	header := &tar.Header{
		Name: fileName,
		Mode: 0o755,
		Size: int64(len(content)),
	}
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("WriteHeader failed: %v", err)
	}
	if _, err := tarWriter.Write(content); err != nil {
		t.Fatalf("Write(content) failed: %v", err)
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("tarWriter.Close failed: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("gzipWriter.Close failed: %v", err)
	}

	return buffer.Bytes()
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) failed: %v", path, err)
	}
	if strings.TrimSpace(string(got)) != want {
		t.Fatalf("content = %q, want %q", string(got), want)
	}
}
