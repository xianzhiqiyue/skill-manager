package handlers

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

const (
	maxSkillArchiveBytes = 20 * 1024 * 1024
	defaultPage          = 1
	defaultPerPage       = 20
	maxPerPage           = 100
)

var (
	versionPattern = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
)

func normalizeNamespace(namespace string) string {
	namespace = strings.TrimSpace(namespace)
	return strings.TrimPrefix(namespace, "@")
}

func namespaceVariants(namespace string) []string {
	namespace = strings.TrimSpace(namespace)
	if namespace == "" {
		return []string{""}
	}

	normalized := normalizeNamespace(namespace)
	withAt := "@" + normalized

	if namespace == normalized {
		return []string{normalized, withAt}
	}
	if namespace == withAt {
		return []string{normalized, withAt}
	}
	return []string{namespace, normalized, withAt}
}

func scopeNamespaceName(db *gorm.DB, namespace, name string) *gorm.DB {
	variants := namespaceVariants(namespace)
	if len(variants) == 1 {
		return db.Where("namespace = ? AND name = ?", variants[0], name)
	}
	return db.Where("name = ? AND namespace IN ?", name, variants)
}

func validateNamespace(namespace string) error {
	return validatePathSegment(namespace, "namespace")
}

func validateSkillName(name string) error {
	return validatePathSegment(name, "name")
}

func validateVersion(version string) error {
	if !versionPattern.MatchString(version) {
		return fmt.Errorf("version must be valid semver, e.g. 1.0.0")
	}
	return nil
}

func parsePagination(pageRaw, perPageRaw string) (int, int) {
	page := defaultPage
	perPage := defaultPerPage

	if n, err := strconv.Atoi(pageRaw); err == nil && n > 0 {
		page = n
	}
	if n, err := strconv.Atoi(perPageRaw); err == nil && n > 0 {
		perPage = n
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	return page, perPage
}

func readUploadedArchive(file *multipart.FileHeader, maxBytes int64) ([]byte, error) {
	if file == nil {
		return nil, fmt.Errorf("missing file")
	}
	if file.Size > 0 && file.Size > maxBytes {
		return nil, fmt.Errorf("skill file is too large (max %d bytes)", maxBytes)
	}

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	content, err := io.ReadAll(io.LimitReader(src, maxBytes+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	if int64(len(content)) > maxBytes {
		return nil, fmt.Errorf("skill file is too large (max %d bytes)", maxBytes)
	}
	return content, nil
}

func storageSegment(value string) string {
	return url.PathEscape(strings.TrimSpace(value))
}

func validatePathSegment(value string, field string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s is required", field)
	}
	if len(value) > 64 {
		return fmt.Errorf("%s is too long", field)
	}
	if value == "." || value == ".." {
		return fmt.Errorf("%s format is invalid", field)
	}
	if strings.ContainsAny(value, `/\`) {
		return fmt.Errorf("%s format is invalid", field)
	}
	if strings.ContainsRune(value, 0) {
		return fmt.Errorf("%s format is invalid", field)
	}
	return nil
}
