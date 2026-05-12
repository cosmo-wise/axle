package cli

import (
	"strings"

	"github.com/cosmo-wise/axle/pkg/axle"
)

func resourceSlug(resource axle.ResourceDescriptor) string {
	value := strings.Trim(strings.TrimSpace(resource.Path), "/")
	if value == "" {
		value = resource.Name
	}
	value = strings.ReplaceAll(value, "-", "_")
	value = strings.ReplaceAll(value, "/", "_")
	return strings.ToLower(value)
}

func importAlias(slug string) string {
	parts := strings.FieldsFunc(slug, func(r rune) bool { return r == '_' || r == '-' || r == '/' })
	if len(parts) == 0 {
		return "resource"
	}
	return strings.Join(parts, "")
}
