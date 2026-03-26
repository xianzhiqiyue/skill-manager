package handlers

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type archiveFormat string

const (
	archiveFormatTarGz archiveFormat = "tar.gz"
	archiveFormatZip   archiveFormat = "zip"
)

func detectArchiveFormat(name string, content []byte) (archiveFormat, error) {
	if len(content) >= 2 && content[0] == 0x1f && content[1] == 0x8b {
		return archiveFormatTarGz, nil
	}
	if len(content) >= 4 && content[0] == 'P' && content[1] == 'K' {
		return archiveFormatZip, nil
	}

	lowerName := strings.ToLower(strings.TrimSpace(name))
	switch {
	case strings.HasSuffix(lowerName, ".tar.gz"), strings.HasSuffix(lowerName, ".tgz"):
		return archiveFormatTarGz, nil
	case strings.HasSuffix(lowerName, ".zip"):
		return archiveFormatZip, nil
	default:
		return "", fmt.Errorf("unsupported archive format: %s", name)
	}
}

func resolveDownloadFormat(raw string, stored archiveFormat) (archiveFormat, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return archiveFormatZip, nil
	case "tgz", "tar.gz":
		return archiveFormatTarGz, nil
	case "zip":
		return archiveFormatZip, nil
	case "original":
		return stored, nil
	default:
		return "", fmt.Errorf("unsupported format: %s", raw)
	}
}

func archiveExtension(format archiveFormat) string {
	switch format {
	case archiveFormatZip:
		return "zip"
	default:
		return "tar.gz"
	}
}

func archiveContentType(format archiveFormat) string {
	switch format {
	case archiveFormatZip:
		return "application/zip"
	default:
		return "application/gzip"
	}
}

func convertArchive(content []byte, source, target archiveFormat) ([]byte, error) {
	if source == target {
		return content, nil
	}

	switch {
	case source == archiveFormatTarGz && target == archiveFormatZip:
		return convertTarGzToZip(content)
	case source == archiveFormatZip && target == archiveFormatTarGz:
		return convertZipToTarGz(content)
	default:
		return nil, fmt.Errorf("unsupported archive conversion: %s -> %s", source, target)
	}
}

func convertTarGzToZip(content []byte) ([]byte, error) {
	gzipReader, err := gzip.NewReader(bytes.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)

	var out bytes.Buffer
	zipWriter := zip.NewWriter(&out)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			zipWriter.Close()
			return nil, fmt.Errorf("read tar entry: %w", err)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := zipWriter.Create(strings.TrimSuffix(header.Name, "/") + "/"); err != nil {
				zipWriter.Close()
				return nil, fmt.Errorf("create zip dir entry: %w", err)
			}
		case tar.TypeReg, tar.TypeRegA:
			entryHeader := &zip.FileHeader{
				Name:     filepath.ToSlash(header.Name),
				Method:   zip.Deflate,
				Modified: header.ModTime,
			}
			entryHeader.SetMode(os.FileMode(header.Mode))

			writer, err := zipWriter.CreateHeader(entryHeader)
			if err != nil {
				zipWriter.Close()
				return nil, fmt.Errorf("create zip file entry: %w", err)
			}
			if _, err := io.Copy(writer, tarReader); err != nil {
				zipWriter.Close()
				return nil, fmt.Errorf("write zip file entry: %w", err)
			}
		case tar.TypeSymlink:
			zipWriter.Close()
			return nil, fmt.Errorf("symbolic links are not supported in zip conversion: %s", header.Name)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("close zip writer: %w", err)
	}
	return out.Bytes(), nil
}

func convertZipToTarGz(content []byte) ([]byte, error) {
	zipReader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, fmt.Errorf("open zip reader: %w", err)
	}

	var out bytes.Buffer
	gzipWriter := gzip.NewWriter(&out)
	tarWriter := tar.NewWriter(gzipWriter)

	for _, file := range zipReader.File {
		info := file.FileInfo()
		if info.Mode()&os.ModeSymlink != 0 {
			tarWriter.Close()
			gzipWriter.Close()
			return nil, fmt.Errorf("symbolic links are not supported in tar.gz conversion: %s", file.Name)
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			tarWriter.Close()
			gzipWriter.Close()
			return nil, fmt.Errorf("create tar header: %w", err)
		}
		header.Name = file.Name

		if err := tarWriter.WriteHeader(header); err != nil {
			tarWriter.Close()
			gzipWriter.Close()
			return nil, fmt.Errorf("write tar header: %w", err)
		}

		if info.IsDir() {
			continue
		}

		reader, err := file.Open()
		if err != nil {
			tarWriter.Close()
			gzipWriter.Close()
			return nil, fmt.Errorf("open zip entry: %w", err)
		}

		if _, err := io.Copy(tarWriter, reader); err != nil {
			reader.Close()
			tarWriter.Close()
			gzipWriter.Close()
			return nil, fmt.Errorf("write tar body: %w", err)
		}
		reader.Close()
	}

	if err := tarWriter.Close(); err != nil {
		gzipWriter.Close()
		return nil, fmt.Errorf("close tar writer: %w", err)
	}
	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("close gzip writer: %w", err)
	}
	return out.Bytes(), nil
}
