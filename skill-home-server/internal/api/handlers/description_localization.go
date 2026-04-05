package handlers

import (
	"strings"

	"github.com/skill-home/server/internal/models"
)

func manifestStringValue(manifest models.JSON, key string) string {
	if len(manifest) == 0 {
		return ""
	}

	value, ok := manifest[key].(string)
	if !ok {
		return ""
	}

	return strings.TrimSpace(value)
}

func resolveSkillDescriptions(formDescription, formDescriptionZh string, manifest models.JSON) (string, string) {
	description := strings.TrimSpace(formDescription)
	if description == "" {
		description = manifestStringValue(manifest, "description")
	}

	descriptionZh := strings.TrimSpace(formDescriptionZh)
	if descriptionZh == "" {
		descriptionZh = manifestStringValue(manifest, "description_zh")
	}

	return description, descriptionZh
}
