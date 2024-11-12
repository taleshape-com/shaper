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

func GetDashboard(app *App, ctx context.Context, dashboardName string, queryParams url.Values) (GetResult, error) {
	fileName := path.Join(app.DashboardDir, dashboardName+".sql")
	result := GetResult{
		Title:    dashboardName,
		Sections: []Section{},
	}
	// read sql file
	sqlFile, err := os.ReadFile(fileName)
	if err != nil {
		return result, err
	}
	nextLabel := ""
	nextIsDownload := false
	sqls := strings.Split(string(sqlFile), ";")
	// TODO: currently variables have to be defined in the order they are used. create a dependency graph for queryies instead
	singleVars := map[string]string{}
	multiVars := map[string][]string{}

	for queryIndex, sqlString := range sqls {
		if queryIndex == len(sqls)-1 {
			// Ignore text after last semicolon
			break
		}
		if nextIsDownload {
			nextIsDownload = false
			continue
		}
		varPrefix := buildVarPrefix(singleVars, multiVars)
		query := Query{Columns: []Column{}, Rows: Rows{}}
		// run query
		rows, err := app.db.QueryxContext(ctx, varPrefix+string(sqlString)+";")
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

		if isLabel(sqlString, query.Rows) {
			nextLabel = query.Rows[0][0].(string)
			continue
		}

		if isSectionTitle(sqlString, query.Rows) {
			sectionTitle := query.Rows[0][0].(string)
			if len(result.Sections) == 0 || result.Sections[len(result.Sections)-1].Type != "header" || result.Sections[len(result.Sections)-1].Title != nil {
				result.Sections = append(result.Sections, Section{
					Type:    "header",
					Queries: []Query{},
				})
			}
			lastSection := &result.Sections[len(result.Sections)-1]
			if sectionTitle == "" {
				lastSection.Title = nil
			} else {
				lastSection.Title = &sectionTitle
			}
			continue
		}

		rInfo := getRenderInfo(colTypes, query.Rows, sqlString, nextLabel)
		query.Render = Render{
			Type:  rInfo.Type,
			Label: rInfo.Label,
		}

		if rInfo.Download != "" {
			nextIsDownload = true
		}

		for colIndex, c := range colTypes {
			nullable, ok := c.Nullable()
			col := Column{
				Name:     c.Name(),
				Type:     mapDBType(c.DatabaseTypeName(), colIndex, query.Rows),
				Nullable: ok && nullable,
				Tag:      mapTag(colIndex, rInfo),
			}
			query.Columns = append(query.Columns, col)
			err := collectVars(singleVars, multiVars, rInfo.Type, colIndex, queryParams, col.Tag, query.Rows, col.Name)
			if err != nil {
				return result, err
			}
			if rInfo.Download == "csv" {
				filename := query.Rows[0][colIndex].(string)
				query.Rows[0][colIndex] = fmt.Sprintf("/api/dashboards/%s/query/%d/%s.csv", dashboardName, queryIndex+1, url.QueryEscape(filename))
			}
		}

		wantedSectionType := "content"
		if query.Render.Type == "dropdown" || query.Render.Type == "dropdownMulti" || query.Render.Type == "button" {
			wantedSectionType = "header"
		}
		if len(result.Sections) != 0 && result.Sections[len(result.Sections)-1].Type == wantedSectionType {
			lastSection := &result.Sections[len(result.Sections)-1]
			lastSection.Queries = append(lastSection.Queries, query)
		} else {
			result.Sections = append(result.Sections, Section{
				Type:    wantedSectionType,
				Queries: []Query{query},
			})
		}

		nextLabel = ""
	}
	return result, err
}

func escapeSQLString(str string) string {
	// Replace single quotes with doubled single quotes
	escaped := strings.Replace(str, "'", "''", -1)

	// Optional: Replace other potentially dangerous characters
	escaped = strings.Replace(escaped, "\x00", "", -1) // Remove null bytes
	escaped = strings.Replace(escaped, "\n", " ", -1)  // Replace newlines
	escaped = strings.Replace(escaped, "\r", " ", -1)  // Replace carriage returns
	escaped = strings.Replace(escaped, "\x1a", "", -1) // Remove ctrl+Z

	return escaped
}

func escapeSQLIdentifier(str string) string {
	// Replace single quotes with doubled single quotes
	escaped := strings.Replace(str, "\"", "\"\"", -1)

	// Optional: Replace other potentially dangerous characters
	escaped = strings.Replace(escaped, "\x00", "", -1) // Remove null bytes
	escaped = strings.Replace(escaped, "\n", " ", -1)  // Replace newlines
	escaped = strings.Replace(escaped, "\r", " ", -1)  // Replace carriage returns
	escaped = strings.Replace(escaped, "\x1a", "", -1) // Remove ctrl+Z

	return escaped
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
	if rInfo.Download != "" {
		return "download"
	}
	return ""
}

// TODO: map all types
func mapDBType(dbType string, index int, rows Rows) string {
	// t := getTypeByDefinition(dbType)
	// if t == "" {
	// t = dbType
	// }
	t := dbType
	switch t {
	case "BOOLEAN":
		return "boolean"
	case "VARCHAR":
		// TODO: Once we have union types, VARCHAR cannot be date anymore. We can use actual TIMESTAMP AND DATE types
		if onlyYears(rows, index) {
			return "year"
		}
		if onlyMonths(rows, index) {
			return "month"
		}
		if onlyDates(rows, index) {
			return "date"
		}
		if onlyHours(rows, index) {
			return "hour"
		}
		if onlyTimestamps(rows, index) {
			return "timestamp"
		}
		return "string"
	case "DOUBLE":
		return "number"
	case "FLOAT":
		return "number"
	case "BIGINT":
		return "number"
	case "DATE":
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
func isLabel(sqlString string, rows Rows) bool {
	return strings.Contains(sqlString, "::LABEL") && len(rows) == 1 && len(rows[0]) == 1
}

// TODO: Once UNION types work, we need a more solid way to detect labels
func isSectionTitle(sqlString string, rows Rows) bool {
	return strings.Contains(sqlString, "::SECTION") && len(rows) == 1 && len(rows[0]) == 1
}

// TODO: Line charts should assert that only one XAXIS/LINECHART_YAXIS/LINECHART_CATEGORY is present and no columns without a tag
func getRenderInfo(columns []*sql.ColumnType, rows Rows, sqlString string, label string) renderInfo {
	var labelValue *string
	if label != "" {
		labelValue = &label
	}
	xaxis := getTagName(sqlString, "XAXIS")

	lineY := getTagName(sqlString, "LINECHART_YAXIS")
	if lineY != "" && xaxis != "" {
		lineCat := getTagName(sqlString, "LINECHART_CATEGORY")
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

	barY := getTagName(sqlString, "BARCHART_YAXIS")
	if barY != "" && xaxis != "" {
		barCat := getTagName(sqlString, "BARCHART_CATEGORY")
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

	dropdown := getTagName(sqlString, "DROPDOWN")
	if dropdown != "" {
		label := getTagName(sqlString, "LABEL")
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

	dropdownMulti := getTagName(sqlString, "DROPDOWN_MULTI")
	if dropdownMulti != "" {
		label := getTagName(sqlString, "LABEL")
		hint := getTagName(sqlString, "HINT")
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

	downloadCSV := getTagName(sqlString, "DOWNLOAD_CSV")
	if downloadCSV != "" {
		return renderInfo{
			Label:    labelValue,
			Type:     "button",
			Download: "csv",
		}
	}

	return renderInfo{
		Label: labelValue,
		Type:  "table",
	}
}

// TODO: Once we use union types, we should not have to use strings for dates and we should note attempt to parse dates
func onlyDates(rows Rows, index int) bool {
	for _, row := range rows {
		s, ok := row[index].(string)
		if !ok {
			return false
		}
		_, err := time.Parse(time.DateOnly, s)
		if err != nil {
			return false
		}
	}
	return true
}

// TODO: Once we use union types, we should not have to use strings for dates and we should note attempt to parse dates
func onlyTimestamps(rows Rows, index int) bool {
	for _, row := range rows {
		s, ok := row[index].(string)
		if !ok {
			return false
		}
		_, err := time.Parse(time.DateTime, s)
		if err != nil {
			return false
		}
	}
	return true
}

func onlyYears(rows Rows, index int) bool {
	for _, row := range rows {
		// TODO: Once we use union types, we should not have to use strings for dates and we should note attempt to parse dates
		s, ok := row[index].(string)
		if !ok {
			return false
		}
		t, err := time.Parse(time.DateOnly, s)
		if err != nil {
			t, err = time.Parse(time.DateTime, s)
			if err != nil {
				return false
			}
		}
		if t.Month() != 1 || t.Day() != 1 || t.Minute() != 0 || t.Second() != 0 || t.Nanosecond() != 0 {
			return false
		}
	}
	return true
}

func onlyMonths(rows Rows, index int) bool {
	for _, row := range rows {
		// TODO: Once we use union types, we should not have to use strings for dates and we should note attempt to parse dates
		s, ok := row[index].(string)
		if !ok {
			return false
		}
		t, err := time.Parse(time.DateOnly, s)
		if err != nil {
			t, err = time.Parse(time.DateTime, s)
			if err != nil {
				return false
			}
		}
		if t.Day() != 1 || t.Minute() != 0 || t.Second() != 0 || t.Nanosecond() != 0 {
			return false
		}
	}
	return true
}

func onlyHours(rows Rows, index int) bool {
	for _, row := range rows {
		// TODO: Once we use union types, we should not have to use strings for dates and we should note attempt to parse dates
		s, ok := row[index].(string)
		if !ok {
			return false
		}
		t, err := time.Parse(time.DateOnly, s)
		if err != nil {
			t, err = time.Parse(time.DateTime, s)
			if err != nil {
				return false
			}
		}
		if t.Minute() != 0 || t.Second() != 0 || t.Nanosecond() != 0 {
			return false
		}
	}
	return true
}

func hasOnlyUniqueValues(rows Rows, columnIndex int) bool {
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

// TODO: assert that variable names are alphanumeric
// TODO: test and harden variable escaping
// TODO: assert that variables in query are set. otherwise it silently falls back to empty string
func buildVarPrefix(singleVars map[string]string, multiVars map[string][]string) string {
	varPrefix := strings.Builder{}
	for k, v := range singleVars {
		varPrefix.WriteString(fmt.Sprintf("SET VARIABLE %s = '%s';\n", escapeSQLIdentifier(k), escapeSQLString(v)))
	}
	for k, v := range multiVars {
		l := ""
		for i, p := range v {
			prefix := ", "
			if i == 0 {
				prefix = ""
			}
			l += fmt.Sprintf("%s'%s'", prefix, escapeSQLString(p))
		}
		varPrefix.WriteString(fmt.Sprintf("SET VARIABLE %s = [%s]::VARCHAR;\n", escapeSQLIdentifier(k), l))
	}
	return varPrefix.String()
}

func collectVars(singleVars map[string]string, multiVars map[string][]string, renderType string, columnIndex int, queryParams url.Values, columnTag string, data Rows, columnName string) error {
	// Fetch vars from dropdown
	if renderType == "dropdown" && columnTag == "value" {
		param := queryParams.Get(columnName)
		if param == "" {
			// Set default value to first row
			if len(data) == 0 {
				// Hide dropdown if now rows to select from
				return nil
			}
			param = data[0][columnIndex].(string)
		} else {
			isValidVar := false
			for _, row := range data {
				if row[columnIndex].(string) == param {
					isValidVar = true
					break
				}
			}
			if !isValidVar {
				return fmt.Errorf("invalid value for query param '%s': %s", columnName, param)
			}
		}
		singleVars[columnName] = param
	}
	// Fetch vars from dropdownMulti
	if renderType == "dropdownMulti" && columnTag == "value" {
		params := queryParams[columnName]
		if len(params) == 0 {
			// Set default value to all rows
			for _, row := range data {
				params = append(params, row[columnIndex].(string))
			}
		} else {
			isValidVar := false
			paramsToCheck := map[string]bool{}
			for _, param := range params {
				paramsToCheck[param] = true
			}
			for _, row := range data {
				val := row[columnIndex].(string)
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
				return fmt.Errorf("invalid value for query param '%s': %s", columnName, params)
			}
		}
		multiVars[columnName] = params
	}
	return nil
}
