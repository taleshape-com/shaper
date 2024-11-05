package core

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"
)

type ListResult struct {
	Dashboards []string `json:"dashboards"`
}

type GetResult struct {
	Title   string  `json:"title"`
	Queries []Query `json:"queries"`
}

type Query struct {
	Render  Render          `json:"render"`
	Columns []Column        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
}

type Render struct {
	Type  string  `json:"type"`
	XAxis *string `json:"xAxis"`
}

type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

func ListDashboards(app *App, ctx context.Context) (ListResult, error) {
	result := ListResult{Dashboards: []string{}}
	files, err := os.ReadDir("dashboards")
	if err != nil {
		return result, fmt.Errorf("failed to read dashboards directory: %w", err)
	}
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			dashboardName := strings.TrimSuffix(file.Name(), ".sql")
			result.Dashboards = append(result.Dashboards, dashboardName)
		}
	}
	return result, nil
}

func GetDashboard(app *App, ctx context.Context, name string) (GetResult, error) {
	fileName := path.Join(app.DashboardDir, name+".sql")
	result := GetResult{
		Title:   name,
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
		query := Query{Columns: []Column{}}
		// run query
		rows, err := app.db.QueryxContext(ctx, string(sql)+";")
		if err != nil {
			return result, err
		}
		colTypes, err := rows.ColumnTypes()
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
		for i, c := range colTypes {
			nullable, ok := c.Nullable()
			col := Column{Name: c.Name(), Type: mapDBType(c.DatabaseTypeName(), i, query.Rows), Nullable: ok && nullable}
			query.Columns = append(query.Columns, col)
		}
		query.Render = getRender(query.Columns, query.Rows)
		result.Queries = append(result.Queries, query)
	}
	return result, err
}

// TODO: map all types
func mapDBType(dbType string, index int, rows [][]interface{}) string {
	switch dbType {
	case "VARCHAR":
		return "string"
	case "DOUBLE":
		return "number"
	case "FLOAT":
		return "number"
	case "BIGINT":
		return "number"
	case "DATE":
		if onlyYears(rows, index) {
			return "year"
		}
		return "date"
	case "TIMESTAMP":
	case "TIMESTAMP_NS":
	case "TIMESTAMP_MS":
	case "TIMESTAMP_S":
	case "TIMESTAMPZ":
		return "timestamp"
	}
	panic(fmt.Sprintf("unsupported type: %s", dbType))
}

func getRender(columns []Column, rows [][]interface{}) Render {
	// Check for title case: single string column with one row
	if len(columns) == 1 && len(rows) == 1 && columns[0].Type == "string" {
		return Render{Type: "title"}
	}

	dateTimeColumn := -1
	numberColumns := 0

	for i, col := range columns {
		if col.Type == "year" || col.Type == "date" || col.Type == "timestamp" {
			dateTimeColumn = i
		} else if col.Type == "number" {
			numberColumns++
		}
	}

	if dateTimeColumn != -1 && numberColumns == len(columns)-1 {
		if hasOnlyUniqueValues(rows, dateTimeColumn) {
			return Render{Type: "line", XAxis: &columns[dateTimeColumn].Name}
		}
	}

	return Render{Type: "table"}
}

func onlyYears(rows [][]interface{}, index int) bool {
	for _, row := range rows {
		t := row[index].(time.Time)
		if t.Month() != 1 || t.Day() != 1 {
			return false
		}
	}
	return true
}

func hasOnlyUniqueValues(rows [][]interface{}, columnIndex int) bool {
	seen := make(map[interface{}]bool)
	for _, row := range rows {
		value := row[columnIndex]
		if seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}
