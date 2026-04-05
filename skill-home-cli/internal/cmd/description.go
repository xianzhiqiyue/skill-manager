package cmd

import "strings"

func preferredSkillDescription(descriptionZh, description string) string {
	if value := strings.TrimSpace(descriptionZh); value != "" {
		return value
	}
	return strings.TrimSpace(description)
}

