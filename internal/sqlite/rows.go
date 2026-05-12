package sqlite

import (
	"database/sql"
	"sort"
)

func scanRows(rows *sql.Rows, columns []string) ([]map[string]any, error) {
	var items []map[string]any
	for rows.Next() {
		values := make([]any, len(columns))
		ptrs := make([]any, len(columns))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			return nil, err
		}
		item := map[string]any{}
		for i, column := range columns {
			item[column] = values[i]
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func sortedKeys(values map[string]any) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
