package cmd

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type packOptions struct {
	output   string
	compress bool
}

func newPackCmd() *cobra.Command {
	opts := &packOptions{}

	cmd := &cobra.Command{
		Use:   "pack [path]",
		Short: "打包技能",
		Long:  "将技能目录打包为 .tar.gz 文件",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			return runPack(path, opts)
		},
	}

	cmd.Flags().StringVarP(&opts.output, "output", "o", "", "输出文件名")
	cmd.Flags().BoolVar(&opts.compress, "compress", true, "启用压缩")

	return cmd
}

func runPack(path string, opts *packOptions) error {
	// 获取技能名称
	skillName := filepath.Base(path)

	// 确定输出文件名
	output := opts.output
	if output == "" {
		output = fmt.Sprintf("%s-%s.tar.gz", skillName, time.Now().Format("20060102"))
	}

	// 创建输出文件
	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer file.Close()

	// 创建 gzip writer
	var writer io.WriteCloser = file
	if opts.compress {
		gw := gzip.NewWriter(file)
		gw.Name = output
		gw.ModTime = time.Now()
		writer = gw
		defer writer.Close()
	}

	// 创建 tar writer
	tw := tar.NewWriter(writer)
	defer tw.Close()

	// 打包目录
	filesPacked := 0
	err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过隐藏文件和目录
		if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// 跳过不需要的文件
		if shouldSkipFile(info.Name()) {
			return nil
		}

		// 获取相对路径
		relPath, err := filepath.Rel(path, filePath)
		if err != nil {
			return err
		}

		// 创建 tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		// 写入 header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// 如果是普通文件，写入内容
		if !info.IsDir() {
			data, err := os.Open(filePath)
			if err != nil {
				return err
			}
			defer data.Close()

			if _, err := io.Copy(tw, data); err != nil {
				return err
			}
		}

		filesPacked++
		return nil
	})

	if err != nil {
		return fmt.Errorf("打包失败: %w", err)
	}

	fmt.Println(color.GreenString("✓"), "打包成功!")
	fmt.Printf("  技能: %s\n", color.CyanString(skillName))
	fmt.Printf("  输出: %s\n", color.CyanString(output))
	fmt.Printf("  文件数: %d\n", filesPacked)

	return nil
}

// shouldSkipFile 判断是否应该跳过该文件
func shouldSkipFile(name string) bool {
	skipFiles := []string{
		".git", ".gitignore",
		"node_modules",
		".DS_Store", "Thumbs.db",
		"*.log", "*.tmp",
	}

	for _, skip := range skipFiles {
		if matched, _ := filepath.Match(skip, name); matched {
			return true
		}
	}
	return false
}
