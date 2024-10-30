package core

import (
	"context"
	"os"
)

func Sample(app *App, ctx context.Context) ([]map[string]interface{}, error) {
	results := []map[string]interface{}{}
	// read sql file
	fileName := "test.sql"
	sqlFile, err := os.ReadFile(fileName)
	if err != nil {
		return results, err
	}
	// run query
	rows, err := app.db.QueryxContext(ctx, string(sqlFile))
	if err != nil {
		return results, err
	}
	for rows.Next() {
		row := make(map[string]interface{})
		err := rows.MapScan(row)
		if err != nil {
			return results, err
		}
		results = append(results, row)
	}
	return results, err
}
