package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/skill-home/cli/pkg/archive"
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
		Long:  "将技能目录打包为 .zip 文件",
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
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("解析技能路径失败: %w", err)
	}
	skillName := filepath.Base(absPath)

	// 确定输出文件名
	output := opts.output
	if output == "" {
		output = fmt.Sprintf("%s-%s.zip", skillName, time.Now().Format("20060102"))
	}

	if !opts.compress {
		fmt.Println(color.YellowString("!"), "--compress=false 对 zip 输出无影响，已忽略")
	}

	filesPacked := 0
	err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldSkipFile(info.Name()) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Name() == "." {
			return nil
		}
		filesPacked++
		return nil
	})
	if err != nil {
		return fmt.Errorf("统计文件失败: %w", err)
	}

	if err := archive.CreateZip(path, output, shouldSkipFile); err != nil {
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
