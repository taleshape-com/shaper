package core

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/marcboeker/go-duckdb"
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
			tag := mapTag(colIndex, rInfo)
			colType, err := mapDBType(c.DatabaseTypeName(), colIndex, query.Rows, tag)
			if err != nil {
				return result, err
			}
			col := Column{
				Name:     c.Name(),
				Type:     colType,
				Nullable: ok && nullable,
				Tag:      tag,
			}
			query.Columns = append(query.Columns, col)
			if rInfo.Download == "csv" {
				filename := query.Rows[0][colIndex].(string)
				query.Rows[0][colIndex] = fmt.Sprintf("/api/dashboards/%s/query/%d/%s.csv", dashboardName, queryIndex+1, url.QueryEscape(filename))
			}
		}

		// TODO: Once UNION types work, we can return floats from the DB directly and don't have to guess
		for _, row := range query.Rows {
			for i, cell := range row {
				if colTypes[i].DatabaseTypeName() == "VARCHAR" && query.Columns[i].Type == "number" {
					if n, err := strconv.ParseFloat(cell.(string), 64); err == nil {
						row[i] = n
					}
					break
				}
				if query.Columns[i].Type == "duration" {
					// in milliseconds
					row[i] = (row[i].(duckdb.Interval)).Micros / 1000
				}
			}
		}

		err = collectVars(singleVars, multiVars, rInfo.Type, queryParams, query.Columns, query.Rows)
		if err != nil {
			return result, err
		}

		wantedSectionType := "content"
		if query.Render.Type == "dropdown" || query.Render.Type == "dropdownMulti" || query.Render.Type == "button" || query.Render.Type == "datepicker" || query.Render.Type == "daterangePicker" {
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
	if rInfo.Type == "linechart" || rInfo.Type == "barchartHorizontal" || rInfo.Type == "barchartHorizontalStacked" || rInfo.Type == "barchartVertical" || rInfo.Type == "barchartVerticalStacked" {
		if rInfo.IndexAxisIndex != nil && index == *rInfo.IndexAxisIndex {
			return "index"
		}
		if rInfo.ValueAxisIndex != nil && index == *rInfo.ValueAxisIndex {
			return "value"
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
	if rInfo.Type == "datepicker" {
		if rInfo.ValueIndex != nil && index == *rInfo.ValueIndex {
			return "default"
		}
	}
	if rInfo.Type == "daterangePicker" {
		if rInfo.FromIndex != nil && index == *rInfo.FromIndex {
			return "defaultFrom"
		}
		if rInfo.ToIndex != nil && index == *rInfo.ToIndex {
			return "defaultTo"
		}
	}
	if rInfo.Download != "" {
		return "download"
	}
	if rInfo.Type == "value" {
		if rInfo.CompareIndex != nil && index == *rInfo.CompareIndex {
			return "compare"
		}
		return "value"
	}
	return ""
}

// TODO: map all types
func mapDBType(dbType string, index int, rows Rows, tag string) (string, error) {
	// t := getTypeByDefinition(dbType)
	// if t == "" {
	// t = dbType
	// }
	t := dbType
	switch t {
	case "BOOLEAN":
		return "boolean", nil
	case "VARCHAR":
		// TODO: Once we have union types, VARCHAR cannot be date anymore. We can use actual TIMESTAMP AND DATE types
		if onlyYears(rows, index) {
			return "year", nil
		}
		if onlyMonths(rows, index) {
			return "month", nil
		}
		if onlyDates(rows, index) {
			return "date", nil
		}
		if onlyHours(rows, index) {
			return "hour", nil
		}
		if onlyTimestamps(rows, index) {
			return "timestamp", nil
		}
		if tag == "index" && onlyNumbers(rows, index) {
			return "number", nil
		}
		return "string", nil
	case "DOUBLE":
		return "number", nil
	case "FLOAT":
		return "number", nil
	case "BIGINT":
		return "number", nil
	case "DATE":
		return "date", nil
	case "TIMESTAMP", "TIMESTAMP_NS", "TIMESTAMP_MS", "TIMESTAMP_S", "TIMESTAMPZ":
		return "timestamp", nil
	case "INTERVAL":
		return "duration", nil
	}
	return "", fmt.Errorf("unsupported type: %s", t)
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

// TODO: Charts should assert that only the required columns are present.
// TODO: BARCHART_STACKED must have CATEGORY column
func getRenderInfo(columns []*sql.ColumnType, rows Rows, sqlString string, label string) renderInfo {
	var labelValue *string
	if label != "" {
		labelValue = &label
	}
	xaxis := getTagName(sqlString, "XAXIS")

	linechart := getTagName(sqlString, "LINECHART")
	if linechart != "" && xaxis != "" {
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
				r.IndexAxisIndex = &i
			}
			if c.Name() == linechart {
				r.ValueAxisIndex = &i
			}
		}
		return r
	}

	barchart := getTagName(sqlString, "BARCHART")
	barCat := getTagName(sqlString, "BARCHART_CATEGORY")
	if barchart != "" && xaxis != "" {
		r := renderInfo{
			Label: labelValue,
			Type:  "barchartHorizontal",
		}
		for i, c := range columns {
			if barCat != "" {
				if c.Name() == barCat {
					r.CategoryIndex = &i
				}
			}
			if c.Name() == xaxis {
				r.IndexAxisIndex = &i
			}
			if c.Name() == barchart {
				r.ValueAxisIndex = &i
			}
		}
		return r
	}
	barchartStacked := getTagName(sqlString, "BARCHART_STACKED")
	if barchartStacked != "" && xaxis != "" && barCat != "" {
		r := renderInfo{
			Label: labelValue,
			Type:  "barchartHorizontalStacked",
		}
		for i, c := range columns {
			if c.Name() == barCat {
				r.CategoryIndex = &i
			}
			if c.Name() == xaxis {
				r.IndexAxisIndex = &i
			}
			if c.Name() == barchartStacked {
				r.ValueAxisIndex = &i
			}
		}
		return r
	}

	yaxis := getTagName(sqlString, "YAXIS")
	if barchart != "" && yaxis != "" {
		r := renderInfo{
			Label: labelValue,
			Type:  "barchartVertical",
		}
		for i, c := range columns {
			if barCat != "" {
				if c.Name() == barCat {
					r.CategoryIndex = &i
				}
			}
			if c.Name() == yaxis {
				r.IndexAxisIndex = &i
			}
			if c.Name() == barchart {
				r.ValueAxisIndex = &i
			}
		}
		return r
	}
	if barchartStacked != "" && yaxis != "" && barCat != "" {
		r := renderInfo{
			Label: labelValue,
			Type:  "barchartVerticalStacked",
		}
		for i, c := range columns {
			if c.Name() == barCat {
				r.CategoryIndex = &i
			}
			if c.Name() == yaxis {
				r.IndexAxisIndex = &i
			}
			if c.Name() == barchartStacked {
				r.ValueAxisIndex = &i
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

	datepicker := getTagName(sqlString, "DATEPICKER")
	if datepicker != "" {
		defaultValueIndex := -1
		for i, c := range columns {
			if c.Name() == datepicker {
				defaultValueIndex = i
			}
		}
		if defaultValueIndex == -1 {
			panic(fmt.Sprintf("column %s not found", datepicker))
		}
		return renderInfo{
			Label:      labelValue,
			Type:       "datepicker",
			ValueIndex: &defaultValueIndex,
		}
	}

	daterangeFrom := getTagName(sqlString, "DATEPICKER_FROM")
	daterangeTo := getTagName(sqlString, "DATEPICKER_TO")
	if daterangeFrom != "" && daterangeTo != "" {
		fromIndex := -1
		toIndex := -1
		for i, c := range columns {
			if c.Name() == daterangeFrom {
				fromIndex = i
			}
			if c.Name() == daterangeTo {
				toIndex = i
			}
		}
		if fromIndex == -1 {
			panic(fmt.Sprintf("column %s not found", daterangeFrom))
		}
		if toIndex == -1 {
			panic(fmt.Sprintf("column %s not found", daterangeFrom))
		}
		return renderInfo{
			Label:     labelValue,
			Type:      "daterangePicker",
			FromIndex: &fromIndex,
			ToIndex:   &toIndex,
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

	// TODO: assert that COMPARE can only be used if both values are the same type
	if len(rows) == 1 {
		firstRow := rows[0]
		if len(firstRow) == 1 {
			return renderInfo{
				Label: labelValue,
				Type:  "value",
			}
		}
		compareTag := getTagName(sqlString, "COMPARE")
		if compareTag != "" && len(firstRow) == 2 {
			valueIndex := -1
			compareIndex := -1
			for i, c := range columns {
				if c.Name() == compareTag {
					compareIndex = i
				} else {
					valueIndex = i
				}
			}
			if valueIndex == -1 {
				panic("value index not found")
			}
			if compareIndex == -1 {
				panic(fmt.Sprintf("column %s not found", compareTag))
			}
			return renderInfo{
				Label:        labelValue,
				CompareIndex: &compareIndex,
				Type:         "value",
			}
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

func onlyNumbers(rows Rows, index int) bool {
	for _, row := range rows {
		s, ok := row[index].(string)
		if !ok {
			return false
		}
		if _, err := strconv.ParseFloat(s, 64); err != nil {
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
		varPrefix.WriteString(fmt.Sprintf("SET VARIABLE %s = %s;\n", escapeSQLIdentifier(k), v))
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
		varPrefix.WriteString(fmt.Sprintf("SET VARIABLE %s = [%s];\n", escapeSQLIdentifier(k), l))
	}
	return varPrefix.String()
}

func collectVars(singleVars map[string]string, multiVars map[string][]string, renderType string, queryParams url.Values, columns []Column, data Rows) error {
	// Fetch vars from dropdown
	if renderType == "dropdown" {
		columnName := ""
		columnIndex := -1
		labelName := ""
		labelIndex := -1
		for i, col := range columns {
			if col.Tag == "value" {
				columnName = col.Name
				columnIndex = i
			}
			if col.Tag == "label" {
				labelName = col.Name
				labelIndex = i
			}
		}
		if columnName == "" {
			return fmt.Errorf("missing value column for dropdown")
		}
		param := queryParams.Get(columnName)
		if param == "" {
			if len(data) == 0 {
				// No vars for dropdown without options
				return nil
			}
			// Set default value to first row
			param = data[0][columnIndex].(string)
		} else {
			// Check if param actually exists in the dropdown
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
		singleVars[columnName] = "'" + escapeSQLString(param) + "'"
		if labelIndex != -1 {
			for _, row := range data {
				val := row[columnIndex].(string)
				// Checking len of row to avoid out of bounds error in case of label being NULL
				if val == param && len(row) > labelIndex {
					label := row[labelIndex].(string)
					singleVars[labelName] = "'" + escapeSQLString(label) + "'"
					break
				}
			}
		}
	}

	// Fetch vars from dropdownMulti
	if renderType == "dropdownMulti" {
		columnName := ""
		columnIndex := -1
		labelName := ""
		labelIndex := -1
		for i, col := range columns {
			if col.Tag == "value" {
				columnName = col.Name
				columnIndex = i
			}
			if col.Tag == "label" {
				labelName = col.Name
				labelIndex = i
			}
		}
		if columnName == "" {
			return fmt.Errorf("missing value column for dropdownMulti")
		}
		params := queryParams[columnName]
		if len(params) == 0 {
			// Set default value to all rows
			for _, row := range data {
				params = append(params, row[columnIndex].(string))
			}
		} else {
			// Check if all params actually exist in the dropdown
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
				}
			}
			if !isValidVar {
				return fmt.Errorf("invalid value for query param '%s': %s", columnName, params)
			}
		}
		multiVars[columnName] = params
		labels := []string{}
		if labelIndex != -1 {
			for _, param := range params {
				for _, row := range data {
					val := row[columnIndex].(string)
					// Checking len of row to avoid out of bounds error in case of label being NULL
					if val == param && len(row) > labelIndex {
						label := row[labelIndex].(string)
						labels = append(labels, label)
					}
				}
			}
			multiVars[labelName] = labels
		}
	}

	// Fetch vars from datepicker
	if renderType == "datepicker" {
		if len(data) == 0 {
			return nil
		}
		columnName := ""
		defaultValueIndex := -1
		for i, col := range columns {
			if col.Tag == "default" {
				columnName = col.Name
				defaultValueIndex = i
				break
			}
		}
		if columnName == "" {
			return fmt.Errorf("missing datepicker column")
		}
		param := queryParams.Get(columnName)
		if param == "" {
			// Set default value
			if defaultValueIndex != -1 {
				val := data[0][defaultValueIndex]
				if val != nil {
					date := val.(time.Time)
					param = date.Format(time.DateOnly)
				}
			}
		} else {
			// Check if param is a valid date
			if !isDateValue(param) {
				return fmt.Errorf("invalid date for datepicker query param '%s': %s", columnName, param)
			}
		}
		if param != "" {
			singleVars[columnName] = "DATE '" + escapeSQLString(param) + "'"
		}
	}

	// Fetch vars from daterangePicker
	if renderType == "daterangePicker" {
		if len(data) == 0 {
			return nil
		}
		fromColumnName := ""
		toColumnName := ""
		fromDefaultValueIndex := -1
		toDefaultValueIndex := -1
		for i, col := range columns {
			if col.Tag == "defaultFrom" {
				fromColumnName = col.Name
				fromDefaultValueIndex = i
			}
			if col.Tag == "defaultTo" {
				toColumnName = col.Name
				toDefaultValueIndex = i
			}
		}
		if fromColumnName == "" {
			return fmt.Errorf("missing DATEPICKER_FROM column")
		}
		if toColumnName == "" {
			return fmt.Errorf("missing DATEPICKER_TO column")
		}
		fromParam := queryParams.Get(fromColumnName)
		if fromParam == "" {
			// Set default value
			if fromDefaultValueIndex != -1 {
				val := data[0][fromDefaultValueIndex]
				if val != nil {
					date := val.(time.Time)
					fromParam = date.Format(time.DateOnly)
				}
			}
		} else {
			// Check if fromParam is a valid date
			if !isDateValue(fromParam) {
				return fmt.Errorf("invalid date for datepicker query fromParam '%s': %s", fromColumnName, fromParam)
			}
		}
		if fromParam != "" {
			singleVars[fromColumnName] = "TIMESTAMP '" + escapeSQLString(fromParam) + "'"
		}
		toParam := queryParams.Get(toColumnName)
		if toParam == "" {
			// Set default value
			if toDefaultValueIndex != -1 {
				val := data[0][toDefaultValueIndex]
				if val != nil {
					date := val.(time.Time)
					toParam = date.Format(time.DateOnly)
				}
			}
		} else {
			// Check if toParam is a valid date
			if !isDateValue(toParam) {
				return fmt.Errorf("invalid date for datepicker query toParam '%s': %s", toColumnName, toParam)
			}
		}
		if toParam != "" {
			singleVars[toColumnName] = "TIMESTAMP '" + escapeSQLString(toParam) + " 23:59:59.999999'"
		}
	}
	return nil
}

func isDateValue(stringDate string) bool {
	_, err := time.Parse(time.DateOnly, stringDate)
	return err == nil
}
