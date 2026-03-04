package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExtractTarGz 解压 tar.gz 文件到指定目录
func ExtractTarGz(src, dst string) error {
	dstAbs, err := filepath.Abs(dst)
	if err != nil {
		return fmt.Errorf("解析目标目录失败: %w", err)
	}

	// 打开文件
	file, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	// 创建 gzip reader
	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("创建 gzip reader 失败: %w", err)
	}
	defer gzReader.Close()

	// 创建 tar reader
	tarReader := tar.NewReader(gzReader)

	// 解压文件
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("读取 tar 失败: %w", err)
		}

			// 构建目标路径（清理并拒绝绝对路径/上级目录）
			cleanName := filepath.Clean(header.Name)
			if cleanName == "." {
				continue
			}
			if filepath.IsAbs(cleanName) || cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(os.PathSeparator)) {
				return fmt.Errorf("不安全的文件路径: %s", header.Name)
			}
			targetPath := filepath.Join(dstAbs, cleanName)

			// 安全检查：防止路径遍历攻击
			if !isSubPath(targetPath, dstAbs) {
				return fmt.Errorf("不安全的文件路径: %s", header.Name)
			}

			switch header.Typeflag {
			case tar.TypeDir:
				if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
					return fmt.Errorf("创建目录失败: %w", err)
				}

			case tar.TypeReg, tar.TypeRegA:
				// 确保父目录存在
				if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
					return fmt.Errorf("创建目录失败: %w", err)
				}
				if err := ensureNoSymlinkParent(dstAbs, targetPath); err != nil {
					return err
				}

				// 创建文件
				outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, os.FileMode(header.Mode))
				if err != nil {
					return fmt.Errorf("创建文件失败: %w", err)
				}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("写入文件失败: %w", err)
			}
			outFile.Close()

			// 设置权限
			if err := os.Chmod(targetPath, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("设置权限失败: %w", err)
			}

			case tar.TypeSymlink:
				// 为避免链接逃逸，当前拒绝提取归档中的符号链接。
				return fmt.Errorf("检测到符号链接条目，已拒绝: %s", header.Name)

			default:
				// 跳过其他类型
				continue
		}
	}

	return nil
}

// isSubPath 检查 child 是否是 parent 的子路径
func isSubPath(child, parent string) bool {
	rel, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return !filepath.IsAbs(rel) && rel != ".." && !filepath.HasPrefix(rel, "../")
}

// ensureNoSymlinkParent 确保目标路径的父目录链路中没有符号链接。
func ensureNoSymlinkParent(root, target string) error {
	parentDir := filepath.Dir(target)
	rel, err := filepath.Rel(root, parentDir)
	if err != nil {
		return fmt.Errorf("解析路径失败: %w", err)
	}
	if rel == "." {
		return nil
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return fmt.Errorf("目标路径超出解压目录")
	}

	current := root
	for _, part := range strings.Split(rel, string(os.PathSeparator)) {
		if part == "" || part == "." {
			continue
		}
		current = filepath.Join(current, part)
		info, err := os.Lstat(current)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("检查路径失败: %w", err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("检测到不安全符号链接路径: %s", current)
		}
	}
	return nil
}

// CreateTarGz 创建 tar.gz 归档
func CreateTarGz(srcDir, dstPath string) error {
	// 创建输出文件
	file, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer file.Close()

	// 创建 gzip writer
	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	// 创建 tar writer
	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// 遍历目录
	return filepath.Walk(srcDir, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 获取相对路径
		relPath, err := filepath.Rel(srcDir, filePath)
		if err != nil {
			return err
		}

		// 跳过根目录
		if relPath == "." {
			return nil
		}

		// 创建 header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		// 写入 header
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// 如果是普通文件，写入内容
		if !info.IsDir() {
			data, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer data.Close()

			if _, err := io.Copy(tarWriter, data); err != nil {
				return err
			}
		}

		return nil
	})
}
