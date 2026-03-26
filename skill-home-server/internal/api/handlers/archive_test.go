package handlers

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"testing"
)

func TestDetectArchiveFormat(t *testing.T) {
	t.Parallel()

	zipContent := mustZipArchive(t, map[string]string{"SKILL.md": "zip"})
	tgzContent := mustTarGzArchive(t, map[string]string{"SKILL.md": "tgz"})

	tests := []struct {
		name    string
		file    string
		content []byte
		want    archiveFormat
	}{
		{name: "zip by content", file: "skill.bin", content: zipContent, want: archiveFormatZip},
		{name: "tgz by content", file: "skill.bin", content: tgzContent, want: archiveFormatTarGz},
		{name: "zip by extension", file: "skill.zip", content: []byte("plain"), want: archiveFormatZip},
		{name: "tgz by extension", file: "skill.tar.gz", content: []byte("plain"), want: archiveFormatTarGz},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := detectArchiveFormat(tt.file, tt.content)
			if err != nil {
				t.Fatalf("detectArchiveFormat returned error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("detectArchiveFormat() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveDownloadFormat(t *testing.T) {
	t.Parallel()

	got, err := resolveDownloadFormat("", archiveFormatZip)
	if err != nil {
		t.Fatalf("resolveDownloadFormat returned error: %v", err)
	}
	if got != archiveFormatZip {
		t.Fatalf("default format = %q, want %q", got, archiveFormatZip)
	}

	got, err = resolveDownloadFormat("zip", archiveFormatTarGz)
	if err != nil {
		t.Fatalf("resolveDownloadFormat returned error: %v", err)
	}
	if got != archiveFormatZip {
		t.Fatalf("zip format = %q, want %q", got, archiveFormatZip)
	}

	got, err = resolveDownloadFormat("original", archiveFormatZip)
	if err != nil {
		t.Fatalf("resolveDownloadFormat returned error: %v", err)
	}
	if got != archiveFormatZip {
		t.Fatalf("original format = %q, want %q", got, archiveFormatZip)
	}
}

func TestConvertArchiveRoundTrip(t *testing.T) {
	t.Parallel()

	zipContent := mustZipArchive(t, map[string]string{
		"SKILL.md":      "name: review",
		"docs/guide.md": "hello",
	})

	tgzContent, err := convertArchive(zipContent, archiveFormatZip, archiveFormatTarGz)
	if err != nil {
		t.Fatalf("convertArchive zip->tgz returned error: %v", err)
	}
	gotTar := readTarGzArchive(t, tgzContent)
	if gotTar["SKILL.md"] != "name: review" || gotTar["docs/guide.md"] != "hello" {
		t.Fatalf("unexpected tar contents: %#v", gotTar)
	}

	zipAgain, err := convertArchive(tgzContent, archiveFormatTarGz, archiveFormatZip)
	if err != nil {
		t.Fatalf("convertArchive tgz->zip returned error: %v", err)
	}
	gotZip := readZipArchive(t, zipAgain)
	if gotZip["SKILL.md"] != "name: review" || gotZip["docs/guide.md"] != "hello" {
		t.Fatalf("unexpected zip contents: %#v", gotZip)
	}
}

func mustZipArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	writer := zip.NewWriter(&buf)
	for name, content := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("Create zip entry failed: %v", err)
		}
		if _, err := io.WriteString(entry, content); err != nil {
			t.Fatalf("WriteString failed: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("Close zip writer failed: %v", err)
	}
	return buf.Bytes()
}

func mustTarGzArchive(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	gzipWriter := gzip.NewWriter(&buf)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatalf("WriteHeader failed: %v", err)
		}
		if _, err := io.WriteString(tarWriter, content); err != nil {
			t.Fatalf("WriteString failed: %v", err)
		}
	}
	if err := tarWriter.Close(); err != nil {
		t.Fatalf("Close tar writer failed: %v", err)
	}
	if err := gzipWriter.Close(); err != nil {
		t.Fatalf("Close gzip writer failed: %v", err)
	}
	return buf.Bytes()
}

func readZipArchive(t *testing.T, content []byte) map[string]string {
	t.Helper()

	reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}

	result := make(map[string]string, len(reader.File))
	for _, file := range reader.File {
		rc, err := file.Open()
		if err != nil {
			t.Fatalf("Open failed: %v", err)
		}
		body, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			t.Fatalf("ReadAll failed: %v", err)
		}
		result[file.Name] = string(body)
	}
	return result
}

func readTarGzArchive(t *testing.T, content []byte) map[string]string {
	t.Helper()

	gzipReader, err := gzip.NewReader(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("NewReader failed: %v", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	result := map[string]string{}
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Next failed: %v", err)
		}
		body, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatalf("ReadAll failed: %v", err)
		}
		result[header.Name] = string(body)
	}
	return result
}
