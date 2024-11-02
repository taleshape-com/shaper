package core

import (
	"context"
	"os"
	"strings"
)

type Result struct {
	Title   string  `json:"title"`
	Queries []Query `json:"queries"`
}

type Query struct {
	Columns []string        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
}

func Sample(app *App, ctx context.Context) (Result, error) {
	fileName := "Birth Data.sql"
	title, _ := strings.CutSuffix(fileName, ".sql")
	result := Result{
		Title:   title,
		Queries: []Query{},
	}
	// read sql file
	sqlFile, err := os.ReadFile(fileName)
	if err != nil {
		return result, err
	}
	sqls := strings.Split(string(sqlFile), ";")
	for i, sql := range sqls {
		if i == len(sqls)-1 {
			continue
		}
		query := Query{}
		// run query
		rows, err := app.db.QueryxContext(ctx, string(sql)+";")
		if err != nil {
			return result, err
		}
		columns, err := rows.Columns()
		query.Columns = columns
		if err != nil {
			return result, err
		}
		for rows.Next() {
			row, err := rows.SliceScan()
			if err != nil {
				return result, err
			}
			query.Rows = append(query.Rows, row)
		}
		result.Queries = append(result.Queries, query)
	}
	return result, err
}
