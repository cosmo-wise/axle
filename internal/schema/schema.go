package schema

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/cosmo-wise/axle/pkg/axle"
)

var identRE = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

// SQLiteType is the single source of truth for descriptor field -> SQLite type mapping.
func SQLiteType(kind string) string {
	switch kind {
	case "integer":
		return "INTEGER"
	case "boolean":
		return "INTEGER"
	case "real":
		return "REAL"
	default:
		return "TEXT"
	}
}

// ValidIdentifier reports whether a descriptor table/column can be quoted safely.
func ValidIdentifier(value string) bool { return identRE.MatchString(strings.TrimSpace(value)) }

// QuoteIdent quotes a validated SQLite identifier.
func QuoteIdent(value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if !ValidIdentifier(trimmed) {
		return "", fmt.Errorf("invalid SQLite identifier %q", value)
	}
	return `"` + strings.ReplaceAll(trimmed, `"`, `""`) + `"`, nil
}

// MustQuoteIdent quotes descriptor identifiers after validation has already run.
func MustQuoteIdent(value string) string {
	quoted, err := QuoteIdent(value)
	if err != nil {
		panic(err)
	}
	return quoted
}

// ColumnDefinition renders a deterministic SQLite column definition.
func ColumnDefinition(resource axle.ResourceDescriptor, field axle.FieldDescriptor, includePrimaryKey bool) (string, error) {
	name, err := QuoteIdent(field.Name)
	if err != nil {
		return "", err
	}
	parts := []string{name, SQLiteType(field.Type)}
	if includePrimaryKey && field.Name == resource.ID {
		parts = append(parts, "PRIMARY KEY")
	}
	if field.Nullable != nil && !*field.Nullable && field.Name != resource.ID {
		parts = append(parts, "NOT NULL")
	}
	if field.Unique && field.Name != resource.ID {
		parts = append(parts, "UNIQUE")
	}
	if strings.TrimSpace(field.Default) != "" {
		parts = append(parts, "DEFAULT", field.Default)
	}
	if field.References != nil {
		table, err := QuoteIdent(field.References.Table)
		if err != nil {
			return "", err
		}
		refField := field.References.Field
		if strings.TrimSpace(refField) == "" {
			refField = "id"
		}
		quotedField, err := QuoteIdent(refField)
		if err != nil {
			return "", err
		}
		parts = append(parts, "REFERENCES", table+"("+quotedField+")")
		if strings.TrimSpace(field.References.OnDelete) != "" {
			parts = append(parts, "ON DELETE", strings.ToUpper(strings.TrimSpace(field.References.OnDelete)))
		}
	}
	return strings.Join(parts, " "), nil
}

// IndexStatements renders deterministic index DDL for fields that request indexes.
func IndexStatements(resource axle.ResourceDescriptor) ([]string, error) {
	table, err := QuoteIdent(resource.Table)
	if err != nil {
		return nil, err
	}
	var statements []string
	for _, field := range resource.Fields {
		if !field.Index {
			continue
		}
		column, err := QuoteIdent(field.Name)
		if err != nil {
			return nil, err
		}
		indexName, err := QuoteIdent("idx_" + resource.Table + "_" + field.Name)
		if err != nil {
			return nil, err
		}
		statements = append(statements, fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s)", indexName, table, column))
	}
	return statements, nil
}
