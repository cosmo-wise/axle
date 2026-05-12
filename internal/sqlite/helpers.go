package sqlite

import "github.com/cosmo-wise/axle/internal/schema"

func (s Store) fieldNames() []string {
	names := make([]string, 0, len(s.resource.Fields))
	for _, field := range s.resource.Fields {
		names = append(names, field.Name)
	}
	return names
}

func (s Store) quotedFieldNames() []string {
	names := make([]string, 0, len(s.resource.Fields))
	for _, field := range s.resource.Fields {
		names = append(names, schema.MustQuoteIdent(field.Name))
	}
	return names
}
