package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/skill-home/server/internal/config"
	"github.com/skill-home/server/internal/storage"
)

type backfillOptions struct {
	DryRun bool
	Stdout io.Writer
}

type backfillStats struct {
	Succeeded int
	Skipped   int
	Failed    int
}

type publicSkillVersionRecord struct {
	Namespace   string `gorm:"column:namespace"`
	Name        string `gorm:"column:name"`
	Version     string `gorm:"column:version"`
	StoragePath string `gorm:"column:storage_path"`
}

type sourceStorageFlags struct {
	Type      string
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool
	LocalPath string
}

func main() {
	if err := run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "public oss backfill failed: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	opts, sourceCfg, err := parseFlags(args)
	if err != nil {
		return err
	}
	opts.Stdout = stdout

	if err := config.Load(); err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	cfg := config.Get()
	if cfg == nil {
		return fmt.Errorf("config not loaded")
	}

	if !hasDatabaseConfig(cfg.Database) {
		if opts.DryRun {
			fmt.Fprintln(stdout, "dry-run: 未检测到完整数据库配置，跳过历史公共包回填。")
			return nil
		}
		return fmt.Errorf("database config is incomplete")
	}

	db, err := storage.NewDatabase(cfg.Database)
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}

	targetStorage, err := storage.NewObjectStorage(cfg.Storage)
	if err != nil {
		return fmt.Errorf("connect target object storage: %w", err)
	}

	sourceStorage, err := newSourceObjectStorage(sourceCfg)
	if err != nil {
		return fmt.Errorf("connect source object storage: %w", err)
	}

	stats, err := runBackfill(ctx, db, sourceStorage, targetStorage, opts)
	if err != nil {
		return err
	}

	if stats.Failed > 0 {
		return fmt.Errorf("backfill finished with %d failures", stats.Failed)
	}
	return nil
}

func parseFlags(args []string) (backfillOptions, sourceStorageFlags, error) {
	var opts backfillOptions
	sourceCfg := sourceStorageFlags{
		Type:      strings.TrimSpace(os.Getenv("SKILL_HOME_BACKFILL_SOURCE_TYPE")),
		Endpoint:  strings.TrimSpace(os.Getenv("SKILL_HOME_BACKFILL_SOURCE_ENDPOINT")),
		AccessKey: strings.TrimSpace(os.Getenv("SKILL_HOME_BACKFILL_SOURCE_ACCESS_KEY")),
		SecretKey: strings.TrimSpace(os.Getenv("SKILL_HOME_BACKFILL_SOURCE_SECRET_KEY")),
		Bucket:    strings.TrimSpace(os.Getenv("SKILL_HOME_BACKFILL_SOURCE_BUCKET")),
		Region:    strings.TrimSpace(os.Getenv("SKILL_HOME_BACKFILL_SOURCE_REGION")),
		UseSSL:    strings.EqualFold(strings.TrimSpace(os.Getenv("SKILL_HOME_BACKFILL_SOURCE_USE_SSL")), "true"),
		LocalPath: strings.TrimSpace(os.Getenv("SKILL_HOME_BACKFILL_SOURCE_LOCAL_PATH")),
	}

	fs := flag.NewFlagSet("public-oss-backfill", flag.ContinueOnError)
	fs.BoolVar(&opts.DryRun, "dry-run", false, "仅检查并输出计划，不写入目标对象存储")
	fs.StringVar(&sourceCfg.Type, "source-type", sourceCfg.Type, "源对象存储类型: local/minio/s3")
	fs.StringVar(&sourceCfg.Endpoint, "source-endpoint", sourceCfg.Endpoint, "源对象存储 endpoint")
	fs.StringVar(&sourceCfg.AccessKey, "source-access-key", sourceCfg.AccessKey, "源对象存储 access key")
	fs.StringVar(&sourceCfg.SecretKey, "source-secret-key", sourceCfg.SecretKey, "源对象存储 secret key")
	fs.StringVar(&sourceCfg.Bucket, "source-bucket", sourceCfg.Bucket, "源对象存储 bucket")
	fs.StringVar(&sourceCfg.Region, "source-region", sourceCfg.Region, "源对象存储 region")
	fs.BoolVar(&sourceCfg.UseSSL, "source-use-ssl", sourceCfg.UseSSL, "源对象存储是否启用 SSL")
	fs.StringVar(&sourceCfg.LocalPath, "source-local-path", sourceCfg.LocalPath, "源本地存储目录")
	if err := fs.Parse(args); err != nil {
		return backfillOptions{}, sourceStorageFlags{}, err
	}

	return opts, sourceCfg, nil
}

func hasDatabaseConfig(cfg config.DatabaseConfig) bool {
	return strings.TrimSpace(cfg.User) != "" && strings.TrimSpace(cfg.Name) != ""
}

func newSourceObjectStorage(flags sourceStorageFlags) (*storage.ObjectStorage, error) {
	if flags.Type == "" && flags.LocalPath != "" {
		flags.Type = "local"
	}
	if flags.Type == "" {
		return nil, nil
	}

	return storage.NewObjectStorage(config.StorageConfig{
		Type:      flags.Type,
		Endpoint:  flags.Endpoint,
		AccessKey: flags.AccessKey,
		SecretKey: flags.SecretKey,
		Bucket:    flags.Bucket,
		Region:    flags.Region,
		UseSSL:    flags.UseSSL,
		LocalPath: flags.LocalPath,
	})
}

func runBackfill(ctx context.Context, db *storage.Database, sourceStorage, targetStorage *storage.ObjectStorage, opts backfillOptions) (backfillStats, error) {
	if opts.Stdout == nil {
		opts.Stdout = io.Discard
	}
	if db == nil {
		return backfillStats{}, fmt.Errorf("database is nil")
	}
	if targetStorage == nil {
		return backfillStats{}, fmt.Errorf("target object storage is nil")
	}

	records, err := loadPublicSkillVersions(db)
	if err != nil {
		return backfillStats{}, fmt.Errorf("load public skill versions: %w", err)
	}

	stats := backfillStats{}
	for _, record := range records {
		label := fmt.Sprintf("%s/%s@%s", record.Namespace, record.Name, record.Version)

		exists, err := targetStorage.Exists(ctx, record.StoragePath)
		if err != nil {
			stats.Failed++
			fmt.Fprintf(opts.Stdout, "failed %s: 检查目标对象失败: %v\n", label, err)
			continue
		}
		if exists {
			stats.Skipped++
			fmt.Fprintf(opts.Stdout, "skip %s: 目标对象已存在\n", label)
			continue
		}

		if sourceStorage == nil {
			stats.Failed++
			fmt.Fprintf(opts.Stdout, "failed %s: 未配置源对象存储\n", label)
			continue
		}

		sourceExists, err := sourceStorage.Exists(ctx, record.StoragePath)
		if err != nil {
			stats.Failed++
			fmt.Fprintf(opts.Stdout, "failed %s: 检查源对象失败: %v\n", label, err)
			continue
		}
		if !sourceExists {
			stats.Failed++
			fmt.Fprintf(opts.Stdout, "failed %s: 源对象不存在\n", label)
			continue
		}

		if opts.DryRun {
			stats.Succeeded++
			fmt.Fprintf(opts.Stdout, "dry-run %s: 将从源存储回填到目标 OSS\n", label)
			continue
		}

		if err := targetStorage.CopyFrom(ctx, record.StoragePath, sourceStorage, record.StoragePath); err != nil {
			stats.Failed++
			fmt.Fprintf(opts.Stdout, "failed %s: 回填复制失败: %v\n", label, err)
			continue
		}

		stats.Succeeded++
		fmt.Fprintf(opts.Stdout, "success %s: 已回填到目标 OSS\n", label)
	}

	fmt.Fprintf(opts.Stdout, "summary: success=%d skipped=%d failed=%d\n", stats.Succeeded, stats.Skipped, stats.Failed)
	return stats, nil
}

func loadPublicSkillVersions(db *storage.Database) ([]publicSkillVersionRecord, error) {
	rows := []publicSkillVersionRecord{}
	err := db.Table("skill_versions").
		Select("skills.namespace, skills.name, skill_versions.version, skill_versions.storage_path").
		Joins("JOIN skills ON skills.id = skill_versions.skill_id").
		Where("skills.is_public = ?", true).
		Where("COALESCE(skills.is_owner_only, FALSE) = FALSE").
		Where("skills.deleted_at IS NULL").
		Where("skill_versions.deleted_at IS NULL").
		Where("COALESCE(skill_versions.storage_path, '') <> ''").
		Order("skills.namespace ASC, skills.name ASC, skill_versions.version ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}
