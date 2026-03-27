package handlers

import (
	"errors"
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
	versionPattern        = regexp.MustCompile(`^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
	errEmailExists        = errors.New("email already exists")
	errSkillAlreadyExists = errors.New("skill already exists")
	errSkillVersionExists = errors.New("skill version already exists")
	errUsernameExists     = errors.New("username already exists")
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

func isDuplicatedKeyError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate key value violates unique constraint") ||
		strings.Contains(message, "unique constraint failed") ||
		strings.Contains(message, "duplicated key not allowed")
}

func isSkillVersionConflictError(err error) bool {
	if !isDuplicatedKeyError(err) {
		return false
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "idx_skill_version") ||
		strings.Contains(message, "skill_versions.skill_id, skill_versions.version")
}

func isSkillNamespaceNameConflictError(err error) bool {
	if !isDuplicatedKeyError(err) {
		return false
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "idx_namespace_name") ||
		strings.Contains(message, "skills.namespace, skills.name")
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
	tag = normalizeTagValue(tag)
	if tag != "" {
		dialect := ""
		if query != nil && query.Dialector != nil {
			dialect = query.Dialector.Name()
		}
		if dialect == "postgres" {
			query = query.Where("? = ANY(tags)", tag)
		} else {
			query = query.Where(
				"tags = ? OR tags LIKE ? OR tags LIKE ? OR tags LIKE ?",
				"{"+tag+"}",
				"{"+tag+",%",
				"%,"+tag+",%",
				"%,"+tag+"}",
			)
		}
	}
	return query
}

func applyExtendedSkillFilters(query *gorm.DB, namespace, tag, license string) *gorm.DB {
	query = applySkillFilters(query, namespace, tag)
	license = strings.TrimSpace(license)
	if license != "" {
		query = query.Where("license = ?", license)
	}
	return query
}

func applySkillOrdering(query *gorm.DB, sort string) *gorm.DB {
	if query == nil {
		return query
	}

	switch strings.ToLower(strings.TrimSpace(sort)) {
	case "updated", "recent":
		return query.Order("updated_at DESC").Order("download_count DESC")
	case "newest", "created":
		return query.Order("created_at DESC").Order("download_count DESC")
	case "rating":
		dialect := ""
		if query.Dialector != nil {
			dialect = query.Dialector.Name()
		}
		if dialect == "postgres" {
			return query.
				Order(clause.Expr{
					SQL: "CASE WHEN rating_count = 0 THEN 0 ELSE rating_sum::float / rating_count END DESC",
				}).
				Order("rating_count DESC").
				Order("download_count DESC")
		}
		return query.
			Order(clause.Expr{
				SQL: "CASE WHEN rating_count = 0 THEN 0 ELSE CAST(rating_sum AS REAL) / rating_count END DESC",
			}).
			Order("rating_count DESC").
			Order("download_count DESC")
	case "name", "alpha":
		return query.Order("name ASC").Order("namespace ASC")
	default:
		return query.Order("download_count DESC").Order("updated_at DESC")
	}
}

func applySearchOrdering(query *gorm.DB, q, sort string) *gorm.DB {
	if query == nil {
		return query
	}

	sort = strings.ToLower(strings.TrimSpace(sort))
	if sort != "" {
		return applySkillOrdering(query, sort)
	}

	q = strings.TrimSpace(q)
	if q == "" {
		return applySkillOrdering(query, sort)
	}

	dialect := ""
	if query.Dialector != nil {
		dialect = query.Dialector.Name()
	}

	if dialect != "postgres" {
		return query.Order(clause.Expr{
			SQL: "CASE " +
				"WHEN lower(name) = lower(?) THEN 0 " +
				"WHEN lower(name) LIKE lower(?) THEN 1 " +
				"WHEN lower(description) LIKE lower(?) THEN 2 " +
				"ELSE 3 END",
			Vars: []interface{}{q, q + "%", "%" + q + "%"},
		}).Order("updated_at DESC").Order("created_at DESC").Order("download_count DESC")
	}

	vectorExpr := "to_tsvector('simple', coalesce(name, '') || ' ' || coalesce(description, ''))"
	return query.
		Order(clause.Expr{
			SQL: "CASE " +
				"WHEN lower(name) = lower(?) THEN 0 " +
				"WHEN name ILIKE ? THEN 1 " +
				"WHEN description ILIKE ? THEN 2 " +
				"ELSE 3 END",
			Vars: []interface{}{q, q + "%", "%" + q + "%"},
		}).
		Order(clause.Expr{
			SQL:  "ts_rank_cd(" + vectorExpr + ", plainto_tsquery('simple', ?)) DESC",
			Vars: []interface{}{q},
		}).
		Order("updated_at DESC").
		Order("created_at DESC").
		Order("download_count DESC")
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
		return query.Where("name LIKE ? OR description LIKE ?", "%"+q+"%", "%"+q+"%")
	}

	vectorExpr := "to_tsvector('simple', coalesce(name, '') || ' ' || coalesce(description, ''))"
	return query.
		Where(vectorExpr+" @@ plainto_tsquery('simple', ?) OR name ILIKE ? OR description ILIKE ?", q, "%"+q+"%", "%"+q+"%")
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

func normalizeTagValue(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}

	return strings.Join(strings.Fields(value), "-")
}

func normalizeTags(values []string) []string {
	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))

	for _, value := range values {
		tag := normalizeTagValue(value)
		if tag == "" || len(tag) > 64 {
			continue
		}
		if _, exists := seen[tag]; exists {
			continue
		}
		seen[tag] = struct{}{}
		normalized = append(normalized, tag)
	}

	return normalized
}

func parseTagList(raw string) []string {
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '，' || r == ';' || r == '；' || r == '\n' || r == '\r'
	})
	return normalizeTags(parts)
}

func loadCommunityTags(db *storage.Database, skill *models.Skill, userID *uuid.UUID) error {
	if db == nil || skill == nil || skill.ID == uuid.Nil {
		return nil
	}

	var communityTags []models.SkillCommunityTagSummary
	if err := db.Model(&models.SkillCommunityTag{}).
		Select("tag, COUNT(*) as count").
		Where("skill_id = ?", skill.ID).
		Group("tag").
		Order("count DESC").
		Order("tag ASC").
		Scan(&communityTags).Error; err != nil {
		return err
	}
	skill.CommunityTags = communityTags
	skill.ViewerTags = nil

	if userID == nil || *userID == uuid.Nil {
		return nil
	}

	var viewerTags []string
	if err := db.Model(&models.SkillCommunityTag{}).
		Where("skill_id = ? AND user_id = ?", skill.ID, *userID).
		Order("tag ASC").
		Pluck("tag", &viewerTags).Error; err != nil {
		return err
	}

	skill.ViewerTags = viewerTags
	return nil
}

func populateSkillDetailResponse(db *storage.Database, skill *models.Skill, viewer *models.User) error {
	if skill == nil {
		return nil
	}

	populateSkillComputedFields(skill)

	var viewerID *uuid.UUID
	if viewer != nil {
		viewerID = &viewer.ID
		if err := loadUserRating(db, skill, viewerID); err != nil {
			return err
		}
	}

	return loadCommunityTags(db, skill, viewerID)
}
