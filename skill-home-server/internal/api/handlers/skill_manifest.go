package handlers

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"path"
	"strings"

	"github.com/skill-home/server/internal/models"
	"gopkg.in/yaml.v3"
)

func parseSkillArchiveManifest(content []byte, format archiveFormat) models.JSON {
	skillMarkdown, err := readSkillMarkdownFromArchive(content, format)
	if err != nil || strings.TrimSpace(skillMarkdown) == "" {
		return nil
	}

	frontmatter, ok := extractSkillFrontmatter(skillMarkdown)
	if !ok {
		return nil
	}

	var manifest map[string]any
	if err := yaml.Unmarshal([]byte(frontmatter), &manifest); err != nil || len(manifest) == 0 {
		return nil
	}

	normalizeOpenClawCredentials(manifest)
	deriveRequiresFromOpenClawCredentialsManifest(manifest)
	return models.JSON(manifest)
}

func readSkillMarkdownFromArchive(content []byte, format archiveFormat) (string, error) {
	switch format {
	case archiveFormatZip:
		reader, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
		if err != nil {
			return "", err
		}
		for _, file := range reader.File {
			if file.FileInfo().IsDir() || !strings.EqualFold(path.Base(file.Name), "SKILL.md") {
				continue
			}
			rc, err := file.Open()
			if err != nil {
				return "", err
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
	case archiveFormatTarGz:
		gzipReader, err := gzip.NewReader(bytes.NewReader(content))
		if err != nil {
			return "", err
		}
		defer gzipReader.Close()

		tarReader := tar.NewReader(gzipReader)
		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", err
			}
			if header.FileInfo().IsDir() || !strings.EqualFold(path.Base(header.Name), "SKILL.md") {
				continue
			}
			data, err := io.ReadAll(tarReader)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
	}

	return "", nil
}

func extractSkillFrontmatter(content string) (string, bool) {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	normalized = strings.TrimSpace(normalized)
	if normalized == "" {
		return "", false
	}

	lines := strings.Split(normalized, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", false
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "---" {
			continue
		}
		return strings.TrimSpace(strings.Join(lines[1:i], "\n")), true
	}

	return "", false
}

func normalizeOpenClawCredentials(manifest map[string]any) {
	metadata, ok := manifest["metadata"].(map[string]any)
	if !ok {
		return
	}
	openclaw, ok := metadata["openclaw"].(map[string]any)
	if !ok {
		return
	}
	credentials := normalizeCredentialDescriptors(openclaw["credentials"])
	if len(credentials) == 0 {
		return
	}
	openclaw["credentials"] = credentials
}

func deriveRequiresFromOpenClawCredentialsManifest(manifest map[string]any) {
	if len(manifest) == 0 {
		return
	}
	if existing, ok := manifest["requires"].([]any); ok && len(existing) > 0 {
		return
	}
	if existing, ok := manifest["requires"].([]string); ok && len(existing) > 0 {
		return
	}

	credentials := extractSkillCredentialsFromManifest(models.JSON(manifest))
	if len(credentials) == 0 {
		return
	}

	requires := make([]string, 0, len(credentials))
	seen := make(map[string]struct{}, len(credentials))
	for _, credential := range credentials {
		env := strings.TrimSpace(credential.Env)
		if env == "" {
			continue
		}
		if _, exists := seen[env]; exists {
			continue
		}
		seen[env] = struct{}{}
		requires = append(requires, env)
	}
	if len(requires) == 0 {
		return
	}
	manifest["requires"] = requires
}

func extractSkillCredentialsFromManifest(manifest models.JSON) []models.SkillCredentialDescriptor {
	if len(manifest) == 0 {
		return nil
	}
	metadata, ok := manifest["metadata"].(map[string]any)
	if !ok {
		return nil
	}
	openclaw, ok := metadata["openclaw"].(map[string]any)
	if !ok {
		return nil
	}
	return normalizeCredentialDescriptors(openclaw["credentials"])
}

func normalizeCredentialDescriptors(raw any) []models.SkillCredentialDescriptor {
	switch values := raw.(type) {
	case []models.SkillCredentialDescriptor:
		if len(values) == 0 {
			return nil
		}
		return values
	case []map[string]any:
		descriptors := make([]models.SkillCredentialDescriptor, 0, len(values))
		for _, value := range values {
			descriptor, ok := decodeCredentialDescriptor(value)
			if !ok {
				continue
			}
			descriptors = append(descriptors, descriptor)
		}
		if len(descriptors) == 0 {
			return nil
		}
		return descriptors
	case []any:
		descriptors := make([]models.SkillCredentialDescriptor, 0, len(values))
		for _, value := range values {
			descriptor, ok := decodeCredentialDescriptor(value)
			if !ok {
				continue
			}
			descriptors = append(descriptors, descriptor)
		}
		if len(descriptors) == 0 {
			return nil
		}
		return descriptors
	default:
		return nil
	}
}

func decodeCredentialDescriptor(raw any) (models.SkillCredentialDescriptor, bool) {
	switch value := raw.(type) {
	case models.SkillCredentialDescriptor:
		if strings.TrimSpace(value.ID) == "" && strings.TrimSpace(value.Env) == "" {
			return models.SkillCredentialDescriptor{}, false
		}
		return value, true
	case map[string]any:
		descriptor := models.SkillCredentialDescriptor{
			ID:          readStringValue(value["id"]),
			Env:         readStringValue(value["env"]),
			Label:       readStringValue(value["label"]),
			Description: readStringValue(value["description"]),
			Secret:      readBoolValue(value["secret"]),
			Required:    readBoolValue(value["required"]),
			Input:       readStringValue(value["input"]),
			HelpURL:     readStringValue(value["help_url"]),
			Group:       readStringValue(value["group"]),
		}
		if descriptor.ID == "" && descriptor.Env == "" {
			return models.SkillCredentialDescriptor{}, false
		}
		return descriptor, true
	default:
		return models.SkillCredentialDescriptor{}, false
	}
}

func readStringValue(raw any) string {
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func readBoolValue(raw any) bool {
	value, ok := raw.(bool)
	return ok && value
}
