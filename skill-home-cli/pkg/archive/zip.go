package archive

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractZip 解压 zip 文件到指定目录。
func ExtractZip(src, dst string) error {
	dstAbs, err := filepath.Abs(dst)
	if err != nil {
		return fmt.Errorf("解析目标目录失败: %w", err)
	}

	reader, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("打开 zip 文件失败: %w", err)
	}
	defer reader.Close()

	for _, file := range reader.File {
		isDir := strings.HasSuffix(file.Name, "/")
		cleanName := filepath.Clean(file.Name)
		if cleanName == "." {
			continue
		}
		if filepath.IsAbs(cleanName) || cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(os.PathSeparator)) {
			return fmt.Errorf("不安全的文件路径: %s", file.Name)
		}

		targetPath := filepath.Join(dstAbs, cleanName)
		if !isSubPath(targetPath, dstAbs) {
			return fmt.Errorf("不安全的文件路径: %s", file.Name)
		}

		info := file.FileInfo()
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("检测到符号链接条目，已拒绝: %s", file.Name)
		}

		if info.IsDir() || isDir {
			if err := os.MkdirAll(targetPath, normalizeArchiveMode(info.Mode(), true)); err != nil {
				return fmt.Errorf("创建目录失败: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
			return fmt.Errorf("创建目录失败: %w", err)
		}
		if err := ensureNoSymlinkParent(dstAbs, targetPath); err != nil {
			return err
		}

		srcFile, err := file.Open()
		if err != nil {
			return fmt.Errorf("打开压缩文件条目失败: %w", err)
		}

		targetMode := normalizeArchiveMode(info.Mode(), false)
		outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, targetMode)
		if err != nil {
			srcFile.Close()
			return fmt.Errorf("创建文件失败: %w", err)
		}

		if _, err := io.Copy(outFile, srcFile); err != nil {
			outFile.Close()
			srcFile.Close()
			return fmt.Errorf("写入文件失败: %w", err)
		}

		outFile.Close()
		srcFile.Close()

		if err := os.Chmod(targetPath, targetMode); err != nil {
			return fmt.Errorf("设置权限失败: %w", err)
		}
	}

	return nil
}

func normalizeArchiveMode(mode os.FileMode, isDir bool) os.FileMode {
	if isDir {
		perm := mode.Perm()
		if perm == 0 {
			return 0o755
		}
		if perm&0o700 != 0o700 {
			perm |= 0o700
		}
		return perm
	}
	if mode.Perm() == 0 {
		return 0o644
	}
	return mode
}

// CreateZip 创建 zip 归档。
func CreateZip(srcDir, dstPath string, shouldSkip func(name string) bool) error {
	file, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	return filepath.Walk(srcDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if shouldSkip != nil && shouldSkip(info.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(srcDir, filePath)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}

		relPath = filepath.ToSlash(relPath)
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		if info.IsDir() {
			header.Name += "/"
			header.Method = zip.Store
		} else {
			header.Method = zip.Deflate
		}

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		data, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer data.Close()

		if _, err := io.Copy(writer, data); err != nil {
			return err
		}
		return nil
	})
}
