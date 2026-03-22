package handlers

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/skill-home/server/internal/models"
	"github.com/skill-home/server/internal/storage"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

func applySkillFilters(query *gorm.DB, namespace, tag string) *gorm.DB {
	namespace = normalizeNamespace(namespace)
	if namespace != "" {
		query = query.Where("namespace IN ?", namespaceVariants(namespace))
	}
	if tag != "" {
		query = query.Where("? = ANY(tags)", tag)
	}
	return query
}

func applySearchFilter(query *gorm.DB, q string) *gorm.DB {
	q = strings.TrimSpace(q)
	if q == "" {
		return query
	}

	dialect := ""
	if query != nil && query.Dialector != nil {
		dialect = query.Dialector.Name()
	}
	if dialect != "postgres" {
		return query.
			Where("name LIKE ? OR description LIKE ?", "%"+q+"%", "%"+q+"%").
			Order(clause.Expr{
				SQL:  "CASE WHEN lower(name) = lower(?) THEN 0 WHEN lower(name) LIKE lower(?) THEN 1 ELSE 2 END",
				Vars: []interface{}{q, q + "%"},
			})
	}

	vectorExpr := "to_tsvector('simple', coalesce(name, '') || ' ' || coalesce(description, ''))"
	return query.
		Where(vectorExpr+" @@ plainto_tsquery('simple', ?) OR name ILIKE ? OR description ILIKE ?", q, "%"+q+"%", "%"+q+"%").
		Order(clause.Expr{
			SQL:  "CASE WHEN lower(name) = lower(?) THEN 0 WHEN name ILIKE ? THEN 1 ELSE 2 END",
			Vars: []interface{}{q, q + "%"},
		}).
		Order(clause.Expr{
			SQL:  "ts_rank_cd(" + vectorExpr + ", plainto_tsquery('simple', ?)) DESC",
			Vars: []interface{}{q},
		})
}

func populateSkillComputedFields(skill *models.Skill) {
	if skill == nil {
		return
	}
	skill.Rating = skill.GetRating()
}

func populateSkillsComputedFields(skills []models.Skill) {
	for i := range skills {
		populateSkillComputedFields(&skills[i])
	}
}

func loadUserRating(db *storage.Database, skill *models.Skill, userID *uuid.UUID) error {
	if db == nil || skill == nil || userID == nil || *userID == uuid.Nil {
		return nil
	}

	var rating models.SkillRating
	if err := db.Where("skill_id = ? AND user_id = ?", skill.ID, *userID).First(&rating).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil
		}
		return err
	}

	skill.UserRating = &rating
	return nil
}
