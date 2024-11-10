package core

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
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
	Type  string  `json:"type"`
	Label *string `json:"label"`
}

type renderInfo struct {
	Type          string
	Label         *string
	XAxisIndex    *int
	YAxisIndex    *int
	CategoryIndex *int
	ValueIndex    *int
	LabelIndex    *int
	HintIndex     *int
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
	Tag      string `json:"tag"`
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

func GetDashboard(app *App, ctx context.Context, name string, queryParams url.Values) (GetResult, error) {
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
	// TODO: support multiple values
	// TODO: currently variables have to be defined in the order they are used. create a dependency graph instead
	singleVars := map[string]string{}
	multiVars := map[string][]string{}
	for i, sql := range sqls {
		if i == len(sqls)-1 {
			continue
		}
		// TODO: assert that variable names are alphanumeric
		// TODO: escape variable values
		// TODO: assert that variables in query are set. otherwise it silently falls back to empty string
		varPrefix := ""
		for k, v := range singleVars {
			varPrefix += fmt.Sprintf("SET VARIABLE %s = '%s';\n", k, v)
		}
		for k, v := range multiVars {
			l := ""
			for i, p := range v {
				prefix := ", "
				if i == 0 {
					prefix = ""
				}
				l += fmt.Sprintf("%s'%s'", prefix, p)
			}
			varPrefix += fmt.Sprintf("SET VARIABLE %s = [%s];\n", k, l)
		}
		query := Query{Columns: []Column{}}
		// run query
		rows, err := app.db.QueryxContext(ctx, varPrefix+string(sql)+";")
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
		rInfo := getRenderInfo(colTypes, query.Rows, sql, nextLabel)
		query.Render = Render{
			Type:  rInfo.Type,
			Label: rInfo.Label,
		}
		for i, c := range colTypes {
			nullable, ok := c.Nullable()
			col := Column{
				Name:     c.Name(),
				Type:     mapDBType(c.DatabaseTypeName(), i, query.Rows),
				Nullable: ok && nullable,
				Tag:      mapTag(i, rInfo),
			}
			query.Columns = append(query.Columns, col)
			// Fetch vars from dropdown
			if query.Render.Type == "dropdown" && col.Tag == "value" {
				param := queryParams.Get(col.Name)
				if param == "" {
					// Set default value to first row
					param = query.Rows[0][i].(string)
				} else {
					isValidVar := false
					for _, row := range query.Rows {
						if row[i].(string) == param {
							isValidVar = true
							break
						}
					}
					if !isValidVar {
						return result, fmt.Errorf("invalid value for query param '%s': %s", col.Name, param)
					}
				}
				singleVars[col.Name] = param
			}
			// Fetch vars from dropdownMulti
			if query.Render.Type == "dropdownMulti" && col.Tag == "value" {
				params := queryParams[col.Name]
				if len(params) == 0 {
					// Set default value to all rows
					for _, row := range query.Rows {
						params = append(params, row[i].(string))
					}
				} else {
					isValidVar := false
					paramsToCheck := map[string]bool{}
					for _, param := range params {
						paramsToCheck[param] = true
					}
					for _, row := range query.Rows {
						val := row[i].(string)
						if paramsToCheck[val] {
							delete(paramsToCheck, val)
							if len(paramsToCheck) == 0 {
								isValidVar = true
								break
							}
							continue
						}
					}
					if !isValidVar {
						return result, fmt.Errorf("invalid value for query param '%s': %s", col.Name, params)
					}
				}
				multiVars[col.Name] = params
			}
		}
		result.Queries = append(result.Queries, query)
		nextLabel = ""
	}
	return result, err
}

func mapTag(index int, rInfo renderInfo) string {
	if rInfo.Type == "linechart" || rInfo.Type == "barchart" {
		if rInfo.XAxisIndex != nil && index == *rInfo.XAxisIndex {
			return "xAxis"
		}
		if rInfo.YAxisIndex != nil && index == *rInfo.YAxisIndex {
			return "yAxis"
		}
		if rInfo.CategoryIndex != nil && index == *rInfo.CategoryIndex {
			return "category"
		}
	}
	if rInfo.Type == "dropdown" || rInfo.Type == "dropdownMulti" {
		if rInfo.ValueIndex != nil && index == *rInfo.ValueIndex {
			return "value"
		}
		if rInfo.LabelIndex != nil && index == *rInfo.LabelIndex {
			return "label"
		}
		if rInfo.HintIndex != nil && index == *rInfo.HintIndex {
			return "hint"
		}
	}
	return ""
}

// TODO: map all types
func mapDBType(dbType string, index int, rows [][]interface{}) string {
	// t := getTypeByDefinition(dbType)
	// if t == "" {
	// t = dbType
	// }
	t := dbType
	switch t {
	case "BOOLEAN":
		return "boolean"
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
func getRenderInfo(columns []*sql.ColumnType, rows [][]interface{}, sql string, label string) renderInfo {
	var labelValue *string
	if label != "" {
		labelValue = &label
	}
	xaxis := getTagName(sql, "XAXIS")

	lineY := getTagName(sql, "LINECHART_YAXIS")
	if lineY != "" && xaxis != "" {
		lineCat := getTagName(sql, "LINECHART_CATEGORY")
		r := renderInfo{
			Label: labelValue,
			Type:  "linechart",
		}
		for i, c := range columns {
			if lineCat != "" {
				if c.Name() == lineCat {
					r.CategoryIndex = &i
				}
			}
			if c.Name() == xaxis {
				r.XAxisIndex = &i
			}
			if c.Name() == lineY {
				r.YAxisIndex = &i
			}
		}
		return r
	}

	barY := getTagName(sql, "BARCHART_YAXIS")
	if barY != "" && xaxis != "" {
		barCat := getTagName(sql, "BARCHART_CATEGORY")
		r := renderInfo{
			Label: labelValue,
			Type:  "barchart",
		}
		for i, c := range columns {
			if barCat != "" {
				if c.Name() == barCat {
					r.CategoryIndex = &i
				}
			}
			if c.Name() == xaxis {
				r.XAxisIndex = &i
			}
			if c.Name() == barY {
				r.YAxisIndex = &i
			}
		}
		return r
	}

	dropdown := getTagName(sql, "DROPDOWN")
	if dropdown != "" {
		label := getTagName(sql, "LABEL")
		valueIndex := -1
		var labelIndex *int
		for i, c := range columns {
			if c.Name() == dropdown {
				valueIndex = i
			}
			if label != "" && c.Name() == label {
				labelIndex = &i
			}
		}
		if valueIndex == -1 {
			panic(fmt.Sprintf("column %s not found", dropdown))
		}
		return renderInfo{
			Label:      labelValue,
			Type:       "dropdown",
			ValueIndex: &valueIndex,
			LabelIndex: labelIndex,
		}
	}

	dropdownMulti := getTagName(sql, "DROPDOWN_MULTI")
	if dropdownMulti != "" {
		label := getTagName(sql, "LABEL")
		hint := getTagName(sql, "HINT")
		valueIndex := -1
		var labelIndex *int
		var hintIndex *int
		for i, c := range columns {
			if c.Name() == dropdownMulti {
				valueIndex = i
			}
			if label != "" && c.Name() == label {
				labelIndex = &i
			}
			if hint != "" && c.Name() == hint {
				hintIndex = &i
			}
		}
		if valueIndex == -1 {
			panic(fmt.Sprintf("column %s not found", dropdownMulti))
		}
		return renderInfo{
			Label:      labelValue,
			Type:       "dropdownMulti",
			ValueIndex: &valueIndex,
			LabelIndex: labelIndex,
			HintIndex:  hintIndex,
		}
	}

	return renderInfo{
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
