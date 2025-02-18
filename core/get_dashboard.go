package core

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/marcboeker/go-duckdb"
)

const QUERY_MAX_ROWS = 2000

type DashboardQuery struct {
	Content string
	ID      string
	Name    string
}

func QueryDashboard(app *App, ctx context.Context, dashboardQuery DashboardQuery, queryParams url.Values, variables map[string]interface{}) (GetResult, error) {
	result := GetResult{
		Name:     dashboardQuery.Name,
		Sections: []Section{},
	}
	nextLabel := ""
	hideNextContentSection := false
	nextIsDownload := false
	cleanContent := stripSQLComments(dashboardQuery.Content)
	sqls, err := splitSQLQueries(cleanContent)
	if err != nil {
		return result, err
	}

	// TODO: currently variables have to be defined in the order they are used. create a dependency graph for queryies instead
	singleVars, multiVars, err := getTokenVars(variables)
	if err != nil {
		return result, err
	}

	var minTimeValue int64 = math.MaxInt64
	var maxTimeValue int64

	conn, err := app.db.Connx(ctx)
	if err != nil {
		return result, fmt.Errorf("Error getting conn: %v", err)
	}

	for queryIndex, sqlString := range sqls {
		sqlString = strings.TrimSpace(sqlString)
		if sqlString == "" {
			break
		}
		if nextIsDownload {
			nextIsDownload = false
			continue
		}
		varPrefix, varCleanup := buildVarPrefix(singleVars, multiVars)
		query := Query{Columns: []Column{}, Rows: Rows{}}
		// run query
		rows, err := conn.QueryxContext(ctx, varPrefix+string(sqlString)+";")
		// TODO: Harden DB cleanup logic. We must not leak vars to other queries. Also consider parallel queries
		if varCleanup != "" {
			if _, cleanupErr := conn.ExecContext(ctx, varCleanup); cleanupErr != nil {
				return result, fmt.Errorf("Error cleaning up vars in query %d: %v", queryIndex, cleanupErr)
			}
		}
		if err != nil {
			return result, fmt.Errorf("Error querying DB in query %d: %v", queryIndex, err)
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
			if len(query.Rows) > QUERY_MAX_ROWS {
				// TODO: add a warning to the result and show to user
				app.Logger.InfoContext(ctx, "Query result too large, truncating", "dashboard", dashboardQuery.ID, "query", sqlString, "maxRows", QUERY_MAX_ROWS)
				if err := rows.Close(); err != nil {
					return result, fmt.Errorf("Error closing rows while truncating (dashboard '%v'): %v", dashboardQuery.ID, err)
				}
				break
			}
		}

		if isLabel(colTypes, query.Rows) {
			nextLabel = query.Rows[0][0].(string)
			continue
		}

		if isSectionTitle(colTypes, query.Rows) {
			result.Sections = append(result.Sections, Section{
				Type:    "header",
				Queries: []Query{},
			})
			hideNextContentSection = false
			lastSection := &result.Sections[len(result.Sections)-1]
			if len(query.Rows) == 0 {
				hideNextContentSection = true
				continue
			}
			sectionTitle := query.Rows[0][0].(string)
			if sectionTitle == "" {
				lastSection.Title = nil
			} else {
				lastSection.Title = &sectionTitle
			}
			continue
		}

		rInfo := getRenderInfo(colTypes, query.Rows, nextLabel)
		query.Render = Render{
			Type:  rInfo.Type,
			Label: rInfo.Label,
		}

		if rInfo.Download != "" {
			nextIsDownload = true
		}

		timeColumnIndices := map[int]bool{}

		for colIndex, c := range colTypes {
			nullable, ok := c.Nullable()
			tag := mapTag(colIndex, rInfo)
			colType, err := mapDBType(c.DatabaseTypeName(), colIndex, query.Rows)
			if err != nil {
				return result, err
			}
			if isTimeType(colType) {
				timeColumnIndices[colIndex] = true
			}
			col := Column{
				Name:     c.Name(),
				Type:     colType,
				Nullable: ok && nullable,
				Tag:      tag,
			}
			query.Columns = append(query.Columns, col)
			if rInfo.Download == "csv" || rInfo.Download == "xlsx" {
				filename := query.Rows[0][colIndex].(string)
				queryString := ""
				if len(queryParams) > 0 {
					queryString = "?" + queryParams.Encode()
				}
				query.Rows[0][colIndex] = fmt.Sprintf("/api/dashboards/%s/query/%d/%s.%s%s", dashboardQuery.ID, queryIndex+1, url.QueryEscape(filename), rInfo.Download, queryString)
			}
		}

		err = collectVars(singleVars, multiVars, rInfo.Type, queryParams, query.Columns, query.Rows)
		if err != nil {
			return result, err
		}

		for _, row := range query.Rows {
			for i, cell := range row {
				if t, ok := cell.(time.Time); ok {
					if query.Columns[i].Type == "time" {
						// Convert time to ms since midnight
						seconds := t.Hour()*3600 + t.Minute()*60 + t.Second()
						ms := int64(seconds*1000) + int64(t.Nanosecond()/1000000)
						row[i] = ms
						continue
					}
					ms := t.UnixMilli()
					// Find min/max time for index axis
					if query.Columns[i].Tag == "index" {
						if ms > maxTimeValue {
							maxTimeValue = ms
						} else if ms < minTimeValue {
							minTimeValue = ms
						}
					}
					if query.Columns[i].Type == "string" {
						row[i] = strconv.FormatInt(ms, 10)
					} else {
						row[i] = ms
					}
					continue
				}
				if n, ok := cell.(float64); ok {
					if math.IsNaN(n) {
						row[i] = nil
					} else if query.Columns[i].Type == "string" {
						row[i] = strconv.FormatFloat(n, 'f', -1, 64)
					}
					continue
				}
				if colTypes[i].DatabaseTypeName() == "UUID" {
					if byteSlice, ok := cell.([]uint8); ok {
						row[i] = formatUUID(byteSlice)
					}
					continue
				}
				if query.Columns[i].Type == "duration" {
					v := row[i]
					if v != nil {
						row[i] = formatInterval(v)
					}
					continue
				}
				if query.Columns[i].Type == "stringArray" {
					if arr, ok := cell.([]interface{}); ok {
						s := make([]string, len(arr))
						for i, v := range arr {
							s[i] = fmt.Sprintf("%v", v)
						}
						row[i] = strings.Join(s, ", ")
						continue
					}
				}
				if query.Columns[i].Type == "number" {
					if d, ok := cell.(duckdb.Decimal); ok {
						row[i] = d.Float64()
					}
				}
			}
		}

		wantedSectionType := "content"
		if query.Render.Type == "dropdown" || query.Render.Type == "dropdownMulti" || query.Render.Type == "button" || query.Render.Type == "datepicker" || query.Render.Type == "daterangePicker" {
			wantedSectionType = "header"
		}
		if len(result.Sections) != 0 && result.Sections[len(result.Sections)-1].Type == wantedSectionType {
			lastSection := &result.Sections[len(result.Sections)-1]
			lastSection.Queries = append(lastSection.Queries, query)
		} else {
			if !hideNextContentSection || wantedSectionType != "content" {
				result.Sections = append(result.Sections, Section{
					Type:    wantedSectionType,
					Queries: []Query{query},
				})
			}
			if wantedSectionType == "header" {
				hideNextContentSection = false
			}
		}

		nextLabel = ""
	}
	if err := conn.Close(); err != nil {
		return result, fmt.Errorf("Error closing conn: %v", err)
	}
	result.MinTimeValue = minTimeValue
	result.MaxTimeValue = maxTimeValue
	return result, err
}

func GetDashboard(app *App, ctx context.Context, dashboardId string, queryParams url.Values, variables map[string]interface{}) (GetResult, error) {
	dashboard, err := GetDashboardQuery(app, ctx, dashboardId)
	if err != nil {
		return GetResult{}, err
	}

	return QueryDashboard(app, ctx, DashboardQuery{
		Content: dashboard.Content,
		ID:      dashboardId,
		Name:    dashboard.Name,
	}, queryParams, variables)
}

func stripSQLComments(sql string) string {
	var result strings.Builder
	lines := strings.Split(sql, "\n")

	for _, line := range lines {
		if idx := strings.Index(line, "--"); idx >= 0 {
			// Only take the part before the comment
			line = line[:idx]
		}
		if strings.TrimSpace(line) != "" {
			result.WriteString(line)
			result.WriteString("\n")
		}
	}

	return result.String()
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
	if rInfo.TrendIndex != nil && index == *rInfo.TrendIndex {
		return "trend"
	}
	return ""
}

// regex match for DECIMAL(X,Y)
var matchDecimal = regexp.MustCompile(`DECIMAL\(\d+,\d+\)`)

// TODO: BIT type is not supported yet by Go duckdb lib
// TODO: Support DECIMAL, ARRAY, STRUCT, MAP and generic UNION types
func mapDBType(dbType string, index int, rows Rows) (string, error) {
	t := dbType
	for _, dbType := range dbTypes {
		if dbType.Definition == t {
			if dbType.ResultType == "axis" {
				return getAxisType(rows, index)
			}
			return dbType.ResultType, nil
		}
	}
	switch t {
	case "BOOLEAN":
		return "boolean", nil
	case "VARCHAR":
		// Check if it's a JSON object or array. Unfortunately the database doesn't tell us if it's JSON.
		cell := getFirstNonEmtpyCell(rows, index)
		if cell != nil {
			if _, ok := cell.(map[string]interface{}); ok {
				return "object", nil
			}
			if _, ok := cell.([]interface{}); ok {
				return "array", nil
			}
		}
		return "string", nil
	case "DOUBLE":
		return "number", nil
	case "FLOAT":
		return "number", nil
	case "INTEGER":
		return "number", nil
	case "DATE":
		return "date", nil
	case "TIMESTAMP", "TIMESTAMP_NS", "TIMESTAMP_MS", "TIMESTAMP_S", "TIMESTAMPTZ":
		return getTimestampType(rows, index)
	case "INTERVAL":
		return "duration", nil
	case "TIME":
		return "time", nil
	case "UUID":
		return "string", nil
	case "UINTEGER":
		return "number", nil
	case "BIGINT":
		return "number", nil
	case "SMALLINT":
		return "number", nil
	case "TINYINT":
		return "number", nil
	case "HUGEINT":
		return "number", nil
	case "UBIGINT":
		return "number", nil
	case "UHUGEINT":
		return "number", nil
	case "USMALLINT":
		return "number", nil
	case "UTINYINT":
		return "number", nil
	case "BLOB":
		return "string", nil
	case "VARCHAR[]":
		return "stringArray", nil
	}
	if matchDecimal.MatchString(t) {
		return "number", nil
	}
	return "", fmt.Errorf("unsupported type: %s", t)
}

func getFirstNonEmtpyCell(rows Rows, index int) interface{} {
	for _, row := range rows {
		if row[index] != nil {
			return row[index]
		}
	}
	return nil
}

func isTimeType(columnType string) bool {
	return columnType == "year" || columnType == "month" || columnType == "date" || columnType == "hour" || columnType == "timestamp"
}

func findColumnByTag(columns []*sql.ColumnType, tag string) (*sql.ColumnType, int) {
	unionDefinition := ""
	for _, dbType := range dbTypes {
		if dbType.Name == tag {
			unionDefinition = dbType.Definition
			break
		}
	}
	if unionDefinition == "" {
		return nil, 0
	}
	for i, c := range columns {
		if c.DatabaseTypeName() == unionDefinition {
			return c, i
		}
	}
	return nil, 0
}

func isLabel(columns []*sql.ColumnType, rows Rows) bool {
	col, _ := findColumnByTag(columns, "LABEL")
	if col == nil {
		return false
	}
	return len(rows) == 1 && len(rows[0]) == 1
}

func isSectionTitle(columns []*sql.ColumnType, rows Rows) bool {
	col, _ := findColumnByTag(columns, "SECTION")
	if col == nil {
		return false
	}
	return (len(rows) == 0 || (len(rows) == 1 && len(rows[0]) == 1))
}

func isPlaceholder(columns []*sql.ColumnType, rows Rows) bool {
	col, _ := findColumnByTag(columns, "PLACEHOLDER")
	if col == nil {
		return false
	}
	return (len(rows) == 1 && len(rows[0]) == 1)
}

func getDownloadType(columns []*sql.ColumnType) string {
	csvColumn, _ := findColumnByTag(columns, "DOWNLOAD_CSV")
	if csvColumn != nil {
		return "csv"
	}
	xlsxColumn, _ := findColumnByTag(columns, "DOWNLOAD_XLSX")
	if xlsxColumn != nil {
		return "xlsx"
	}
	return ""
}

// TODO: Charts should assert that only the required columns are present.
// TODO: BARCHART_STACKED must have CATEGORY column
func getRenderInfo(columns []*sql.ColumnType, rows Rows, label string) renderInfo {
	var labelValue *string
	if label != "" {
		labelValue = &label
	}
	xaxis, xaxisIndex := findColumnByTag(columns, "XAXIS")

	linechart, linechartIndex := findColumnByTag(columns, "LINECHART")
	if linechart != nil && xaxis != nil {
		lineCat, lineCatIndex := findColumnByTag(columns, "LINECHART_CATEGORY")
		r := renderInfo{
			Label:          labelValue,
			Type:           "linechart",
			IndexAxisIndex: &xaxisIndex,
			ValueAxisIndex: &linechartIndex,
		}
		if lineCat != nil {
			r.CategoryIndex = &lineCatIndex
		}
		return r
	}

	barchart, barchartIndex := findColumnByTag(columns, "BARCHART")
	barCat, barCatIndex := findColumnByTag(columns, "BARCHART_CATEGORY")
	if barchart != nil && xaxis != nil {
		r := renderInfo{
			Label:          labelValue,
			Type:           "barchartHorizontal",
			IndexAxisIndex: &xaxisIndex,
			ValueAxisIndex: &barchartIndex,
		}
		if barCat != nil {
			r.CategoryIndex = &barCatIndex
		}
		return r
	}
	barchartStacked, barchartStackedIndex := findColumnByTag(columns, "BARCHART_STACKED")
	if barchartStacked != nil && xaxis != nil && barCat != nil {
		r := renderInfo{
			Label:          labelValue,
			Type:           "barchartHorizontalStacked",
			CategoryIndex:  &barCatIndex,
			IndexAxisIndex: &xaxisIndex,
			ValueAxisIndex: &barchartStackedIndex,
		}
		return r
	}

	yaxis, yaxisIndex := findColumnByTag(columns, "YAXIS")
	if barchart != nil && yaxis != nil {
		r := renderInfo{
			Label:          labelValue,
			Type:           "barchartVertical",
			IndexAxisIndex: &yaxisIndex,
			ValueAxisIndex: &barchartIndex,
		}
		if barCat != nil {
			r.CategoryIndex = &barCatIndex
		}
		return r
	}
	if barchartStacked != nil && yaxis != nil && barCat != nil {
		r := renderInfo{
			Label:          labelValue,
			Type:           "barchartVerticalStacked",
			CategoryIndex:  &barCatIndex,
			IndexAxisIndex: &yaxisIndex,
			ValueAxisIndex: &barchartStackedIndex,
		}
		return r
	}

	dropdown, dropdownIndex := findColumnByTag(columns, "DROPDOWN")
	if dropdown != nil {
		label, labelIndex := findColumnByTag(columns, "LABEL")
		r := renderInfo{
			Label:      labelValue,
			Type:       "dropdown",
			ValueIndex: &dropdownIndex,
		}
		if label != nil {
			r.LabelIndex = &labelIndex
		}
		return r
	}

	dropdownMulti, dropdownMultiIndex := findColumnByTag(columns, "DROPDOWN_MULTI")
	if dropdownMulti != nil {
		label, labelIndex := findColumnByTag(columns, "LABEL")
		hint, hintIndex := findColumnByTag(columns, "HINT")
		r := renderInfo{
			Label:      labelValue,
			Type:       "dropdownMulti",
			ValueIndex: &dropdownMultiIndex,
		}
		if label != nil {
			r.LabelIndex = &labelIndex
		}
		if hint != nil {
			r.HintIndex = &hintIndex
		}
		return r
	}

	datepicker, datepickerIndex := findColumnByTag(columns, "DATEPICKER")
	if datepicker != nil {
		return renderInfo{
			Label:      labelValue,
			Type:       "datepicker",
			ValueIndex: &datepickerIndex,
		}
	}

	daterangeFrom, daterangeFromIndex := findColumnByTag(columns, "DATEPICKER_FROM")
	daterangeTo, daterangeToIndex := findColumnByTag(columns, "DATEPICKER_TO")
	if daterangeFrom != nil && daterangeTo != nil {
		return renderInfo{
			Label:     labelValue,
			Type:      "daterangePicker",
			FromIndex: &daterangeFromIndex,
			ToIndex:   &daterangeToIndex,
		}
	}

	downloadType := getDownloadType(columns)
	if downloadType != "" {
		return renderInfo{
			Label:    labelValue,
			Type:     "button",
			Download: downloadType,
		}
	}

	if isPlaceholder(columns, rows) {
		return renderInfo{
			Label: labelValue,
			Type:  "placeholder",
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
		compareTag, compareTagIndex := findColumnByTag(columns, "COMPARE")
		if compareTag != nil && len(firstRow) == 2 {
			return renderInfo{
				Label:        labelValue,
				CompareIndex: &compareTagIndex,
				Type:         "value",
			}
		}
	}

	trendTag, trendTagIndex := findColumnByTag(columns, "TREND")
	r := renderInfo{
		Label: labelValue,
		Type:  "table",
	}
	if trendTag != nil {
		r.TrendIndex = &trendTagIndex
	}
	return r
}

func getTimestampType(rows Rows, index int) (string, error) {
	s := "timestamp"
	if len(rows) < 2 {
		return s, nil
	}
	for _, row := range rows {
		r := row[index]
		if r == nil {
			continue
		}
		t, ok := r.(time.Time)
		if !ok {
			return "", fmt.Errorf("invalid timestamp value: %v", row[index])
		}
		if s == "timestamp" {
			s = "year"
		}
		if s != "date" && s != "hour" && s != "timestamp" && t.Month() != 1 {
			s = "month"
		}
		if s != "hour" && s != "timestamp" && t.Day() != 1 {
			s = "date"
		}
		if s != "timestamp" && t.Hour() != 0 {
			s = "hour"
		}
		if s == "hour" && t.Minute() != 0 || t.Second() != 0 || t.Nanosecond() != 0 {
			return "timestamp", nil
		}
	}
	return s, nil
}

func getAxisType(rows Rows, index int) (string, error) {
	if len(rows) == 0 {
		return "string", nil
	}
	// Try timestamp first
	if s, err := getTimestampType(rows, index); err == nil {
		return s, nil
	}
	// Then try number and fallback to string
	for _, row := range rows {
		if _, ok := row[index].(float64); !ok {
			return "string", nil
		}
	}
	return "number", nil
}

// TODO: assert that variable names are alphanumeric
// TODO: test and harden variable escaping
// TODO: assert that variables in query are set. otherwise it silently falls back to empty string
func buildVarPrefix(singleVars map[string]string, multiVars map[string][]string) (string, string) {
	varPrefix := strings.Builder{}
	varCleanup := strings.Builder{}
	for k, v := range singleVars {
		varPrefix.WriteString(fmt.Sprintf("SET VARIABLE \"%s\" = %s;\n", escapeSQLIdentifier(k), v))
		varCleanup.WriteString(fmt.Sprintf("RESET VARIABLE \"%s\";\n", escapeSQLIdentifier(k)))
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
		varPrefix.WriteString(fmt.Sprintf("SET VARIABLE \"%s\" = [%s]::VARCHAR[];\n", escapeSQLIdentifier(k), l))
		varCleanup.WriteString(fmt.Sprintf("RESET VARIABLE \"%s\";\n", escapeSQLIdentifier(k)))
	}
	return varPrefix.String(), varCleanup.String()
}

func collectVars(singleVars map[string]string, multiVars map[string][]string, renderType string, queryParams url.Values, columns []Column, data Rows) error {
	// Fetch vars from dropdown
	if renderType == "dropdown" {
		columnName := ""
		columnIndex := -1
		for i, col := range columns {
			if col.Tag == "value" {
				columnName = col.Name
				columnIndex = i
				break
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
	}

	// Fetch vars from dropdownMulti
	if renderType == "dropdownMulti" {
		columnName := ""
		columnIndex := -1
		for i, col := range columns {
			if col.Tag == "value" {
				columnName = col.Name
				columnIndex = i
				break
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

func getTokenVars(variables map[string]interface{}) (map[string]string, map[string][]string, error) {
	singleVars := map[string]string{}
	multiVars := map[string][]string{}
	for k, v := range variables {
		switch v := v.(type) {
		case string:
			singleVars[k] = "'" + escapeSQLString(v) + "'"
		case []interface{}:
			strSlice := make([]string, 0, len(v))
			for _, item := range v {
				if str, ok := item.(string); ok {
					strSlice = append(strSlice, str)
				} else {
					return singleVars, multiVars, fmt.Errorf("invalid type in array for key %s: %T", k, item)
				}
			}
			multiVars[k] = strSlice
		default:
			return singleVars, multiVars, fmt.Errorf("unsupported type for key %s: %T", k, v)
		}
	}
	return singleVars, multiVars, nil
}

// Format as standard UUID string format (8-4-4-4-12)
func formatUUID(s []uint8) string {
	return fmt.Sprintf("%x-%x-%x-%x-%x", s[0:4], s[4:6], s[6:8], s[8:10], s[10:16])
}

// interval in milliseconds
func formatInterval(v interface{}) int64 {
	interval := v.(duckdb.Interval)
	ms := interval.Micros / 1000
	ms += int64(interval.Days) * 24 * 60 * 60 * 1000
	ms += int64(interval.Months) * 30 * 24 * 60 * 60 * 1000
	return ms
}

// Split by ; and handle ; inside single and double quotes
func splitSQLQueries(sql string) ([]string, error) {
	var queries []string
	var currentQuery strings.Builder
	var inSingleQuote bool
	var inDoubleQuote bool
	var lineNum int = 1
	var quoteStartLine int

	for i := 0; i < len(sql); i++ {
		c := sql[i]
		currentQuery.WriteByte(c)

		// Track line numbers
		if c == '\n' {
			lineNum++
		}

		// Handle single quotes
		if c == '\'' && !inDoubleQuote {
			if i+1 < len(sql) && sql[i+1] == '\'' {
				currentQuery.WriteByte(sql[i+1])
				i++
				continue
			}
			if !inSingleQuote {
				quoteStartLine = lineNum
			}
			inSingleQuote = !inSingleQuote
			continue
		}

		// Handle double quotes
		if c == '"' && !inSingleQuote {
			if i+1 < len(sql) && sql[i+1] == '"' {
				currentQuery.WriteByte(sql[i+1])
				i++
				continue
			}
			if !inDoubleQuote {
				quoteStartLine = lineNum
			}
			inDoubleQuote = !inDoubleQuote
			continue
		}

		// Handle semicolon
		if c == ';' && !inSingleQuote && !inDoubleQuote {
			query := strings.TrimSpace(currentQuery.String())
			if len(query) > 0 {
				queries = append(queries, query[:len(query)-1]) // Remove the semicolon
			}
			currentQuery.Reset()
		}
	}

	if inSingleQuote {
		return nil, fmt.Errorf("unclosed single quote starting in line %d", quoteStartLine+1)
	}
	if inDoubleQuote {
		return nil, fmt.Errorf("unclosed double quote starting in line %d", quoteStartLine+1)
	}

	lastQuery := strings.TrimSpace(currentQuery.String())
	if lastQuery != "" {
		queries = append(queries, lastQuery)
	}

	return queries, nil
}
