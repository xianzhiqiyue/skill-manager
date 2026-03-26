package archive

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

type Format string

const (
	FormatTarGz Format = "tar.gz"
	FormatZip   Format = "zip"
)

// DetectFormat 检测归档格式，优先使用文件头，扩展名作为兜底。
func DetectFormat(path string) (Format, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	header := make([]byte, 4)
	n, err := file.Read(header)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("读取文件头失败: %w", err)
	}
	header = header[:n]

	if len(header) >= 2 && header[0] == 0x1f && header[1] == 0x8b {
		return FormatTarGz, nil
	}
	if len(header) >= 4 && header[0] == 'P' && header[1] == 'K' {
		return FormatZip, nil
	}

	lowerPath := strings.ToLower(path)
	switch {
	case strings.HasSuffix(lowerPath, ".tar.gz"), strings.HasSuffix(lowerPath, ".tgz"):
		return FormatTarGz, nil
	case strings.HasSuffix(lowerPath, ".zip"):
		return FormatZip, nil
	default:
		return "", fmt.Errorf("不支持的归档格式: %s", path)
	}
}

// ExtractAuto 根据文件格式自动选择解压器。
func ExtractAuto(src, dst string) error {
	format, err := DetectFormat(src)
	if err != nil {
		return err
	}

	switch format {
	case FormatTarGz:
		return ExtractTarGz(src, dst)
	case FormatZip:
		return ExtractZip(src, dst)
	default:
		return fmt.Errorf("不支持的归档格式: %s", format)
	}
}
