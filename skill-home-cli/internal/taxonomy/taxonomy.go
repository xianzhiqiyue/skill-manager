package taxonomy

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"
)

//go:embed taxonomy.json
var rawDefinition []byte

type Category struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Description string `json:"description"`
}

type OfficialTag struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type Definition struct {
	Categories      []Category        `json:"categories"`
	CategoryAliases map[string]string `json:"category_aliases"`
	OfficialTags    []OfficialTag     `json:"official_tags"`
	Aliases         map[string]string `json:"aliases"`
	categorySet     map[string]struct{}
	categoryValues  map[string]string
	tagSet          map[string]struct{}
}

var (
	loadOnce sync.Once
	cached   *Definition
	loadErr  error
)

func Load() (*Definition, error) {
	loadOnce.Do(func() {
		var definition Definition
		if err := json.Unmarshal(rawDefinition, &definition); err != nil {
			loadErr = err
			return
		}
		definition.buildIndexes()
		cached = &definition
	})
	return cached, loadErr
}

func (d *Definition) HasCategory(value string) bool {
	if d == nil {
		return false
	}
	_, ok := d.categorySet[normalizeValue(d.NormalizeCategory(value))]
	return ok
}

func (d *Definition) NormalizeCategory(value string) string {
	normalized := normalizeValue(value)
	if normalized == "" {
		return ""
	}
	if target, ok := d.CategoryAliases[normalized]; ok {
		normalized = normalizeValue(target)
	}
	if canonical, ok := d.categoryValues[normalized]; ok {
		return canonical
	}
	return normalized
}

func (d *Definition) HasOfficialTag(value string) bool {
	if d == nil {
		return false
	}
	_, ok := d.tagSet[d.NormalizeTag(value)]
	return ok
}

func (d *Definition) NormalizeTag(value string) string {
	normalized := normalizeValue(value)
	if normalized == "" {
		return ""
	}
	if target, ok := d.Aliases[normalized]; ok {
		return normalizeValue(target)
	}
	return normalized
}

func (d *Definition) buildIndexes() {
	d.categorySet = make(map[string]struct{}, len(d.Categories))
	d.categoryValues = make(map[string]string, len(d.Categories))
	for _, category := range d.Categories {
		normalized := normalizeValue(category.ID)
		d.categorySet[normalized] = struct{}{}
		d.categoryValues[normalized] = category.ID
	}
	d.tagSet = make(map[string]struct{}, len(d.OfficialTags))
	for _, tag := range d.OfficialTags {
		d.tagSet[normalizeValue(tag.ID)] = struct{}{}
	}
}

func normalizeValue(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return ""
	}
	return strings.Join(strings.Fields(value), "-")
}
