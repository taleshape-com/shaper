package core

import (
	"context"
	"fmt"
	"os"
	"path"
	"regexp"
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
	Type          string  `json:"type"`
	Label         *string `json:"label"`
	XAxis         *string `json:"xAxis"`
	YAxis         *string `json:"yAxis"`
	CategoryIndex *int    `json:"categoryIndex"`
}

//	type Desc struct {
//		ColumnName string  `db:"column_name"`
//		ColumnType string  `db:"column_type"`
//		Null       string  `db:"null"`
//		Key        *string `db:"key"`
//		Default    *string `db:"default"`
//		Extra      *string `db:"extra"`
//	}
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
	nextLabel := ""
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
		// prefix with DESCRIBE and run query to get the details of the UNION type
		// desc := []Desc{}
		// err = app.db.SelectContext(ctx, &desc, "DESCRIBE "+string(sql)+";")
		// if err != nil {
		// 	return result, err
		// }
		// for i, d := range desc {
		// 	col := Column{
		// 		Name:     d.ColumnName,
		// 		Type:     mapDBType(d.ColumnType, i, query.Rows),
		// 		Nullable: d.Null == "YES",
		// 	}
		// 	query.Columns = append(query.Columns, col)
		// }
		if isLabel(sql, query.Rows) {
			nextLabel = query.Rows[0][0].(string)
			continue
		}
		for i, c := range colTypes {
			nullable, ok := c.Nullable()
			col := Column{Name: c.Name(), Type: mapDBType(c.DatabaseTypeName(), i, query.Rows), Nullable: ok && nullable}
			query.Columns = append(query.Columns, col)
		}
		query.Render = getRender(query.Columns, query.Rows, sql, nextLabel)
		result.Queries = append(result.Queries, query)
		nextLabel = ""
	}
	return result, err
}

// TODO: map all types
func mapDBType(dbType string, index int, rows [][]interface{}) string {
	// t := getTypeByDefinition(dbType)
	// if t == "" {
	// t = dbType
	// }
	t := dbType
	switch t {
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
	case "TIMESTAMP", "TIMESTAMP_NS", "TIMESTAMP_MS", "TIMESTAMP_S", "TIMESTAMPZ":
		return "timestamp"
		// case "LINECHART_CATEGORY":
		// 	return "string"
		// case "LINECHART_YAXIS":
		// 	return "number"
		// case "XAXIS":
		// 	return "string"
	}
	panic(fmt.Sprintf("unsupported type: %s", t))
}

// TODO: Once UNION types work, we need a more solid way to get tags
func getTagName(sql string, tag string) string {
	s := "::" + tag + " AS \"(.+)\""
	r := regexp.MustCompile(s)
	m := r.FindStringSubmatch(sql)
	if len(m) != 2 {
		return ""
	}
	return m[1]
}

// TODO: Once UNION types work, we need a more solid way to detect labels
func isLabel(sql string, rows [][]interface{}) bool {
	return strings.Contains(sql, "::LABEL") && len(rows) == 1 && len(rows[0]) == 1
}

// TODO: Line charts should assert that only one XAXIS/LINECHART_YAXIS/LINECHART_CATEGORY is present and no columns without a tag
func getRender(columns []Column, rows [][]interface{}, sql string, label string) Render {
	var labelValue *string
	if label != "" {
		labelValue = &label
	}
	xaxis := getTagName(sql, "XAXIS")

	lineY := getTagName(sql, "LINECHART_YAXIS")
	if lineY != "" && xaxis != "" {
		lineCat := getTagName(sql, "LINECHART_CATEGORY")
		r := Render{
			Label: labelValue,
			Type:  "linechart",
			XAxis: &xaxis,
			YAxis: &lineY,
		}
		if lineCat != "" {
			for i, c := range columns {
				if c.Name == lineCat {
					r.CategoryIndex = &i
					break
				}
			}
		}
		return r
	}

	barY := getTagName(sql, "BARCHART_YAXIS")
	if barY != "" && xaxis != "" {
		barCat := getTagName(sql, "BARCHART_CATEGORY")
		r := Render{
			Label: labelValue,
			Type:  "barchart",
			XAxis: &xaxis,
			YAxis: &barY,
		}
		if barCat != "" {
			for i, c := range columns {
				if c.Name == barCat {
					r.CategoryIndex = &i
					break
				}
			}
		}
		return r
	}

	// dateTimeColumn := -1
	// numberColumns := 0
	// for i, col := range columns {
	// 	if col.Type == "year" || col.Type == "date" || col.Type == "timestamp" {
	// 		dateTimeColumn = i
	// 	} else if col.Type == "number" {
	// 		numberColumns++
	// 	}
	// }
	// if dateTimeColumn != -1 && numberColumns == len(columns)-1 {
	// 	if hasOnlyUniqueValues(rows, dateTimeColumn) {
	// 		return Render{Type: "line", XAxis: &columns[dateTimeColumn].Name}
	// 	}
	// }

	return Render{
		Label: labelValue,
		Type:  "table",
	}
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
