package sqlite

import (
	"fmt"
	"strings"

	"github.com/cosmo-wise/axle/pkg/axle"
)

func (s Store) createValues(values map[string]any) (map[string]any, error) {
	fields := s.fieldsByName()
	filtered := map[string]any{}
	for key, value := range values {
		if _, ok := fields[key]; !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnknownField, key)
		}
		filtered[key] = value
	}
	for _, field := range s.resource.Fields {
		if field.Auto == "uuid" && isEmpty(filtered[field.Name]) {
			id, err := newUUID()
			if err != nil {
				return nil, err
			}
			filtered[field.Name] = id
			values[field.Name] = id
		}
	}
	return filtered, nil
}

func (s Store) updateValues(values map[string]any) (map[string]any, error) {
	fields := s.fieldsByName()
	filtered := map[string]any{}
	for key, value := range values {
		field, ok := fields[key]
		if !ok {
			return nil, fmt.Errorf("%w: %s", ErrUnknownField, key)
		}
		if !field.Mutable {
			return nil, fmt.Errorf("%w: %s", ErrImmutableField, key)
		}
		filtered[key] = value
	}
	return filtered, nil
}

func isEmpty(value any) bool {
	if value == nil {
		return true
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text) == ""
	}
	return false
}

func (s Store) fieldsByName() map[string]axle.FieldDescriptor {
	fields := map[string]axle.FieldDescriptor{}
	for _, field := range s.resource.Fields {
		fields[field.Name] = field
	}
	return fields
}
