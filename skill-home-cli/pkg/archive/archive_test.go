package archive

import (
	"archive/zip"
	"os"
	"path/filepath"
	"testing"
)

func TestCreateZipAndExtractAuto(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	writeFile(t, filepath.Join(srcDir, "SKILL.md"), "name: zip-skill")
	writeFile(t, filepath.Join(srcDir, "docs", "guide.md"), "hello")
	writeFile(t, filepath.Join(srcDir, ".gitignore"), "ignored")

	archivePath := filepath.Join(t.TempDir(), "skill.zip")
	if err := CreateZip(srcDir, archivePath, func(name string) bool {
		return name == ".gitignore"
	}); err != nil {
		t.Fatalf("CreateZip returned error: %v", err)
	}

	format, err := DetectFormat(archivePath)
	if err != nil {
		t.Fatalf("DetectFormat returned error: %v", err)
	}
	if format != FormatZip {
		t.Fatalf("DetectFormat() = %q, want %q", format, FormatZip)
	}

	outDir := filepath.Join(t.TempDir(), "out")
	if err := ExtractAuto(archivePath, outDir); err != nil {
		t.Fatalf("ExtractAuto returned error: %v", err)
	}

	assertFileContent(t, filepath.Join(outDir, "SKILL.md"), "name: zip-skill")
	assertFileContent(t, filepath.Join(outDir, "docs", "guide.md"), "hello")
	if _, err := os.Stat(filepath.Join(outDir, ".gitignore")); !os.IsNotExist(err) {
		t.Fatalf(".gitignore should not be archived")
	}
}

func TestDetectFormatRecognizesTarGz(t *testing.T) {
	t.Parallel()

	srcDir := t.TempDir()
	writeFile(t, filepath.Join(srcDir, "SKILL.md"), "name: tgz-skill")

	archivePath := filepath.Join(t.TempDir(), "skill.tar.gz")
	if err := CreateTarGz(srcDir, archivePath); err != nil {
		t.Fatalf("CreateTarGz returned error: %v", err)
	}

	format, err := DetectFormat(archivePath)
	if err != nil {
		t.Fatalf("DetectFormat returned error: %v", err)
	}
	if format != FormatTarGz {
		t.Fatalf("DetectFormat() = %q, want %q", format, FormatTarGz)
	}
}

func TestExtractAutoHandlesLegacyZipEntriesWithoutModes(t *testing.T) {
	t.Parallel()

	archivePath := filepath.Join(t.TempDir(), "legacy.zip")
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	writer := zip.NewWriter(file)

	dirHeader := &zip.FileHeader{Name: "references/"}
	dirHeader.Method = zip.Store
	if _, err := writer.CreateHeader(dirHeader); err != nil {
		t.Fatalf("CreateHeader(dir) failed: %v", err)
	}

	fileHeader := &zip.FileHeader{Name: "references/cli-workflows.md", Method: zip.Deflate}
	entry, err := writer.CreateHeader(fileHeader)
	if err != nil {
		t.Fatalf("CreateHeader(file) failed: %v", err)
	}
	if _, err := entry.Write([]byte("legacy")); err != nil {
		t.Fatalf("Write(entry) failed: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("zip.Close failed: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("file.Close failed: %v", err)
	}

	outDir := filepath.Join(t.TempDir(), "out")
	if err := ExtractAuto(archivePath, outDir); err != nil {
		t.Fatalf("ExtractAuto returned error: %v", err)
	}

	assertFileContent(t, filepath.Join(outDir, "references", "cli-workflows.md"), "legacy")

	info, err := os.Stat(filepath.Join(outDir, "references"))
	if err != nil {
		t.Fatalf("Stat(dir) failed: %v", err)
	}
	if info.Mode().Perm() == 0 {
		t.Fatalf("directory permissions should be normalized, got %v", info.Mode())
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll failed: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q) failed: %v", path, err)
	}
	if string(got) != want {
		t.Fatalf("content = %q, want %q", string(got), want)
	}
}
