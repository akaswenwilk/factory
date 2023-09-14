package factory

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

func NewQueryFunc(db *sql.DB) QueryFunc {
	return func(ctx context.Context, sqlStatement string, args ...any) (string, error) {
		rows, err := db.QueryContext(ctx, sqlStatement, args...)
		if err != nil {
			return "", err
		}
		defer rows.Close()

		type TableRow map[string]interface{}

		// Get column names
		columns, err := rows.Columns()
		if err != nil {
			return "", err
		}

		// Create a slice to hold the results as TableRow maps
		var results []TableRow

		// Iterate through the result rows and build TableRow maps
		for rows.Next() {
			// Create a slice of interface{} to store column values
			values := make([]interface{}, len(columns))
			valuePointers := make([]interface{}, len(columns))

			for i := range columns {
				valuePointers[i] = &values[i]
			}

			// Scan the row into the value pointers
			if err := rows.Scan(valuePointers...); err != nil {
				return "", err
			}

			// Build a TableRow map from column names and values
			rowData := make(TableRow)
			for i, col := range columns {
				switch v := values[i].(type) {
				case []byte:
					rowData[col] = string(v)
				case nil:
					rowData[col] = nil
				case bool, string, int, int64, float64, []interface{}:
					rowData[col] = v
				default:
					rowData[col] = fmt.Sprintf("%v", v)
				}
			}

			results = append(results, rowData)
		}

		// Check for errors from iterating over rows
		if err := rows.Err(); err != nil {
			return "", err
		}

		// Marshal the result slice into a JSON string
		jsonData, err := json.Marshal(results)
		if err != nil {
			return "", err
		}

		return string(jsonData), nil
	}
}
