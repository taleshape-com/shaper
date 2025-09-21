// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/url"
	"regexp"
	"shaper/server/util"
	"strconv"
	"strings"
	"time"

	"github.com/marcboeker/go-duckdb/v2"
)

const QUERY_MAX_ROWS = 3000

// These SQL statements are used only for their side effects and not to display anything.
// They are not visible in the dashboard output.
var sideEffectSQLStatements = [][]string{
	{"ATTACH"},
	{"USE"},
	{"SET", "VARIABLE"},
	{"RESET", "VARIABLE"},
	{"CREATE", "TEMPORARY", "TABLE"},
	{"CREATE", "TEMPORARY", "VIEW"},
	{"CREATE", "TEMP", "TABLE"},
	{"CREATE", "TEMP", "VIEW"},
	{"CREATE", "OR", "REPLACE", "TEMPORARY", "TABLE"},
	{"CREATE", "OR", "REPLACE", "TEMPORARY", "VIEW"},
	{"CREATE", "OR", "REPLACE", "TEMP", "TABLE"},
	{"CREATE", "OR", "REPLACE", "TEMP", "VIEW"},
	{"CREATE", "TEMP", "MACRO"},
	{"CREATE", "TEMP", "FUNCTION"},
	{"CREATE", "TEMPORARY", "MACRO"},
	{"CREATE", "TEMPORARY", "FUNCTION"},
	{"CREATE", "TEMP", "MACRO", "IF", "NOT", "EXISTS"},
	{"CREATE", "TEMP", "FUNCTION", "IF", "NOT", "EXISTS"},
	{"CREATE", "TEMPORARY", "MACRO", "IF", "NOT", "EXISTS"},
	{"CREATE", "TEMPORARY", "FUNCTION", "IF", "NOT", "EXISTS"},
	{"CREATE", "OR", "REPLACE", "TEMP", "MACRO"},
	{"CREATE", "OR", "REPLACE", "TEMP", "FUNCTION"},
	{"CREATE", "OR", "REPLACE", "TEMPORARY", "MACRO"},
	{"CREATE", "OR", "REPLACE", "TEMPORARY", "FUNCTION"},
}

type DashboardQuery struct {
	Content    string
	ID         string
	Name       string
	Visibility *string
}

// QueryDashboard is where most of the complexity of the dashboarding functionality lies.
// It executes the SQL and generates a dashboard definition from the results that can be rendered in the frontend.
func QueryDashboard(app *App, ctx context.Context, dashboardQuery DashboardQuery, queryParams url.Values, variables map[string]any) (GetResult, error) {
	result := GetResult{
		Name:       dashboardQuery.Name,
		Visibility: dashboardQuery.Visibility,
		Sections:   []Section{},
	}
	nextLabel := ""
	hideNextContentSection := false
	nextIsDownload := false
	nextMarkLines := []MarkLine{}
	cleanContent := util.StripSQLComments(dashboardQuery.Content)
	sqls, err := util.SplitSQLQueries(cleanContent)
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
	headerImage := ""
	footerLink := ""

	conn, err := app.DuckDB.Connx(ctx)
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
		rows, err := conn.QueryxContext(ctx, varPrefix+sqlString+";")
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
				if closeErr := rows.Close(); closeErr != nil {
					return result, fmt.Errorf("Error closing rows after scan error. Scan Err: %v. Close Err: %v", err, closeErr)
				}
				return result, err
			}
			query.Rows = append(query.Rows, row)
			if len(query.Rows) > QUERY_MAX_ROWS {
				// TODO: add a warning to the result and show to user
				app.Logger.InfoContext(ctx, "Query result too large, truncating", "dashboard", dashboardQuery.ID, "queryIndex", queryIndex, "maxRows", QUERY_MAX_ROWS)
				if err := rows.Close(); err != nil {
					return result, fmt.Errorf("Error closing rows while truncating (dashboard '%v'): %v", dashboardQuery.ID, err)
				}
				break
			}
		}

		if isSideEffect(sqlString) {
			continue
		}

		if isLabel(colTypes, query.Rows) {
			u, ok := query.Rows[0][0].(duckdb.Union)
			if !ok {
				nextLabel = ""
				continue
			}
			l, ok := u.Value.(string)
			if !ok {
				l = ""
			}
			nextLabel = l
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
			u, ok := query.Rows[0][0].(duckdb.Union)
			if !ok {
				lastSection.Title = nil
				continue
			}
			sectionTitle, ok := u.Value.(string)
			if !ok || sectionTitle == "" {
				lastSection.Title = nil
			} else {
				lastSection.Title = &sectionTitle
			}
			continue
		}

		if isReload(colTypes, query.Rows) {
			if result.ReloadAt != 0 {
				return result, fmt.Errorf("Multiple RELOAD queries in dashboard %s", dashboardQuery.ID)
			}
			result.ReloadAt = getReloadValue(query.Rows)
			continue
		}

		if isHeaderImage(colTypes, query.Rows) {
			headerImage = getSingleValue(query.Rows)
			continue
		}
		if isFooterLink(colTypes, query.Rows) {
			footerLink = getSingleValue(query.Rows)
			continue
		}

		if lines, ok := getMarkLines(colTypes, query.Rows); ok {
			nextMarkLines = append(nextMarkLines, lines...)
			continue
		}

		rInfo := getRenderInfo(colTypes, query.Rows, nextLabel, nextMarkLines)
		query.Render = Render{
			Type:            rInfo.Type,
			Label:           rInfo.Label,
			GaugeCategories: rInfo.GaugeCategories,
			MarkLines:       rInfo.MarkLines,
		}

		if rInfo.Download == "csv" || rInfo.Download == "xlsx" {
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
			if (rInfo.Download == "csv" || rInfo.Download == "xlsx" || rInfo.Download == "pdf") && len(query.Rows) > 0 {
				filename := query.Rows[0][colIndex].(duckdb.Union).Value.(string)
				queryString := ""
				if len(queryParams) > 0 {
					queryString = "?" + queryParams.Encode()
				}
				if rInfo.Download == "pdf" {
					query.Rows[0][colIndex] = fmt.Sprintf("api/dashboards/%s/pdf/%s.%s%s", dashboardQuery.ID, url.QueryEscape(filename), rInfo.Download, queryString)
				} else {
					query.Rows[0][colIndex] = fmt.Sprintf("api/dashboards/%s/query/%d/%s.%s%s", dashboardQuery.ID, queryIndex+1, url.QueryEscape(filename), rInfo.Download, queryString)
				}
			}
		}

		err = collectVars(singleVars, multiVars, rInfo.Type, queryParams, query.Columns, query.Rows)
		if err != nil {
			return result, err
		}

		for _, row := range query.Rows {
			for i, cell := range row {
				colType := query.Columns[i].Type
				if u, ok := cell.(duckdb.Union); ok {
					cell = u.Value
					row[i] = u.Value
				}
				if t, ok := cell.(time.Time); ok {
					if colType == "time" {
						row[i] = formatTime(t)
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
					if colType == "string" {
						row[i] = strconv.FormatInt(ms, 10)
					} else {
						row[i] = ms
					}
					continue
				}
				if n, ok := cell.(float64); ok {
					if math.IsNaN(n) {
						row[i] = nil
					} else if colType == "string" {
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
				if colType == "duration" {
					v := row[i]
					if v != nil {
						row[i] = formatInterval(v)
					}
					continue
				}
				if colType == "stringArray" {
					if arr, ok := cell.([]any); ok {
						s := make([]string, len(arr))
						for i, v := range arr {
							s[i] = fmt.Sprintf("%v", v)
						}
						row[i] = strings.Join(s, ", ")
						continue
					}
				}
				if colType == "number" {
					if d, ok := cell.(duckdb.Decimal); ok {
						row[i] = d.Float64()
					}
				}
				if colType == "object" {
					if d, ok := cell.(duckdb.Map); ok {
						allGood := true
						m := make(map[string]string)
						for k, v := range d {
							kStr, ok := k.(string)
							if !ok {
								allGood = false
								break
							}
							vStr, ok := v.(string)
							if !ok {
								allGood = false
								break
							}
							m[kStr] = vStr
						}
						if allGood {
							row[i] = m
						}
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
		nextMarkLines = []MarkLine{}
	}
	if err := conn.Close(); err != nil {
		return result, fmt.Errorf("Error closing conn: %v", err)
	}
	result.MinTimeValue = minTimeValue
	result.MaxTimeValue = maxTimeValue
	if headerImage != "" {
		result.HeaderImage = &headerImage
	}
	if footerLink != "" {
		result.FooterLink = &footerLink
	}
	return result, err
}

func GetDashboard(app *App, ctx context.Context, dashboardId string, queryParams url.Values, variables map[string]any) (GetResult, error) {
	dashboard, err := GetDashboardQuery(app, ctx, dashboardId)
	if err != nil {
		return GetResult{}, err
	}

	return QueryDashboard(app, ctx, DashboardQuery{
		Content:    dashboard.Content,
		ID:         dashboardId,
		Name:       dashboard.Name,
		Visibility: dashboard.Visibility,
	}, queryParams, variables)
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
		if rInfo.ColorIndex != nil && index == *rInfo.ColorIndex {
			return "color"
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
	if rInfo.Type == "gauge" {
		if rInfo.ValueAxisIndex != nil && index == *rInfo.ValueAxisIndex {
			return "value"
		}
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
// TODO: Support ARRAY, LIST, STRUCT, more MAP types and generic UNION types
func mapDBType(dbType string, index int, rows Rows) (string, error) {
	t := dbType
	for _, dbType := range dbTypes {
		if dbType.Definition == t {
			if dbType.ResultType == "chart" {
				return getChartType(rows, index)
			}
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
			if _, ok := cell.(map[string]any); ok {
				return "object", nil
			}
			if _, ok := cell.([]any); ok {
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
	case "JSON":
		return "object", nil
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
	case "ENUM":
		return "string", nil
	case "VARCHAR[]":
		return "stringArray", nil
	case "MAP(VARCHAR, VARCHAR)":
		return "object", nil
	}
	if matchDecimal.MatchString(t) {
		return "number", nil
	}
	return "", fmt.Errorf("unsupported type: %s", t)
}

func getFirstNonEmtpyCell(rows Rows, index int) any {
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
		return nil, -1
	}
	for i, c := range columns {
		if c.DatabaseTypeName() == unionDefinition {
			return c, i
		}
	}
	return nil, -1
}

// Some SQL statements are only used for their side effects and should not be shown on the dashboard.
// We ignore case and extra whitespace when matching the statements
func isSideEffect(sqlString string) bool {
	normalized := strings.ToUpper(strings.TrimSpace(sqlString))
	for _, stmt := range sideEffectSQLStatements {
		mismatch := false
		sub := normalized
		for _, s := range stmt {
			if !strings.HasPrefix(sub, s) {
				mismatch = true
				break
			}
			sub = strings.TrimSpace(strings.TrimPrefix(sub, s))
		}
		if !mismatch {
			return true
		}
	}
	return false
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

func getMarkLines(columns []*sql.ColumnType, rows Rows) ([]MarkLine, bool) {
	axis := ""
	valueIndex := -1
	if col, i := findColumnByTag(columns, "XLINE"); col != nil {
		axis = "x"
		valueIndex = i
	} else if col, i := findColumnByTag(columns, "YLINE"); col != nil {
		axis = "y"
		valueIndex = i
	}
	_, labelIndex := findColumnByTag(columns, "LABEL")
	lines := []MarkLine{}
	if axis == "" || valueIndex == -1 {
		return lines, false
	}
	for _, row := range rows {
		if len(row) <= valueIndex {
			continue
		}
		v := row[valueIndex]
		if v == nil {
			continue
		}
		u, ok := v.(duckdb.Union)
		if !ok || u.Value == nil {
			continue
		}
		line := MarkLine{IsYaxis: axis == "y"}
		// Format values according to same logic as chart values
		switch val := u.Value.(type) {
		case string:
			line.Value = val
		case float64:
			if math.IsNaN(val) || math.IsInf(val, 0) {
				continue
			}
			line.Value = val
		case time.Time:
			if strings.HasSuffix(u.Tag, "_time") {
				line.Value = formatTime(val)
			} else {
				line.Value = val.UnixMilli()
			}
		case duckdb.Interval:
			line.Value = formatInterval(val)
		}
		// Set label if specified
		if labelIndex != -1 && labelIndex < len(row) {
			if u, ok := row[labelIndex].(duckdb.Union); ok {
				if l, ok := u.Value.(string); ok {
					line.Label = l
				}
			}
		}
		lines = append(lines, line)
	}
	return lines, true
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
	pdfColumn, _ := findColumnByTag(columns, "DOWNLOAD_PDF")
	if pdfColumn != nil {
		return "pdf"
	}
	return ""
}

func getRenderInfo(columns []*sql.ColumnType, rows Rows, label string, markLines []MarkLine) renderInfo {
	var labelValue *string
	if label != "" {
		labelValue = &label
	}
	xaxis, xaxisIndex := findColumnByTag(columns, "XAXIS")

	linechart, linechartIndex := findColumnByTag(columns, "LINECHART")
	if linechart == nil {
		linechart, linechartIndex = findColumnByTag(columns, "LINECHART_PERCENT")
	}
	if linechart != nil && xaxis != nil {
		lineCat, lineCatIndex := findColumnByTag(columns, "LINECHART_CATEGORY")
		if lineCat == nil {
			lineCat, lineCatIndex = findColumnByTag(columns, "CATEGORY")
		}
		lineColor, lineColorIndex := findColumnByTag(columns, "LINECHART_COLOR")
		if lineColor == nil {
			lineColor, lineColorIndex = findColumnByTag(columns, "COLOR")
		}
		r := renderInfo{
			Label:          labelValue,
			Type:           "linechart",
			IndexAxisIndex: &xaxisIndex,
			ValueAxisIndex: &linechartIndex,
			MarkLines:      markLines,
		}
		if lineCat != nil {
			r.CategoryIndex = &lineCatIndex
		}
		if lineColor != nil {
			r.ColorIndex = &lineColorIndex
		}
		return r
	}

	barchart, barchartIndex := findColumnByTag(columns, "BARCHART")
	if barchart == nil {
		barchart, barchartIndex = findColumnByTag(columns, "BARCHART_PERCENT")
	}
	barCat, barCatIndex := findColumnByTag(columns, "BARCHART_CATEGORY")
	if barCat == nil {
		barCat, barCatIndex = findColumnByTag(columns, "CATEGORY")
	}
	barColor, barColorIndex := findColumnByTag(columns, "BARCHART_COLOR")
	if barColor == nil {
		barColor, barColorIndex = findColumnByTag(columns, "COLOR")
	}
	if barchart != nil && xaxis != nil {
		r := renderInfo{
			Label:          labelValue,
			Type:           "barchartHorizontal",
			IndexAxisIndex: &xaxisIndex,
			ValueAxisIndex: &barchartIndex,
			MarkLines:      markLines,
		}
		if barCat != nil {
			r.CategoryIndex = &barCatIndex
		}
		if barColor != nil {
			r.ColorIndex = &barColorIndex
		}
		return r
	}
	barchartStacked, barchartStackedIndex := findColumnByTag(columns, "BARCHART_STACKED")
	if barchartStacked == nil {
		barchartStacked, barchartStackedIndex = findColumnByTag(columns, "BARCHART_STACKED_PERCENT")
	}
	if barchartStacked == nil {
		// Alias for BARCHART_STACKED_PERCENT
		barchartStacked, barchartStackedIndex = findColumnByTag(columns, "BARCHART_PERCENT_STACKED")
	}
	if barchartStacked != nil && xaxis != nil && barCat != nil {
		r := renderInfo{
			Label:          labelValue,
			Type:           "barchartHorizontalStacked",
			CategoryIndex:  &barCatIndex,
			IndexAxisIndex: &xaxisIndex,
			ValueAxisIndex: &barchartStackedIndex,
			MarkLines:      markLines,
		}
		if barColor != nil {
			r.ColorIndex = &barColorIndex
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
			MarkLines:      markLines,
		}
		if barCat != nil {
			r.CategoryIndex = &barCatIndex
		}
		if barColor != nil {
			r.ColorIndex = &barColorIndex
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
			MarkLines:      markLines,
		}
		if barColor != nil {
			r.ColorIndex = &barColorIndex
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

	gauge, gaugeIndex := findColumnByTag(columns, "GAUGE")
	isGaugePercent := false
	if gauge == nil {
		gauge, gaugeIndex = findColumnByTag(columns, "GAUGE_PERCENT")
		isGaugePercent = true
	}
	if gauge != nil && len(rows) == 1 {
		rangeCol, rangeIndex := findColumnByTag(columns, "RANGE")
		labelsCol, labelsIndex := findColumnByTag(columns, "LABELS")
		colorsCol, colorsIndex := findColumnByTag(columns, "COLORS")
		row := rows[0]
		rangeArr := []any{}
		if rangeCol != nil {
			if rangeUnion, ok := row[rangeIndex].(duckdb.Union); ok {
				// TODO: Assert that gaugeUnion.Tag and rangeUnion.Tag match; gaugeUnion := row[gaugeIndex].(duckdb.Union)
				if arr, ok := rangeUnion.Value.([]any); ok {
					rangeArr = arr
				}
			}
		}
		// TODO: warn if range length is less than 2
		if lessThanTwoUniqueRangeValues(rangeArr) {
			// default values
			var gaugeValue float64 = 0.0
			var isInterval bool = false
			var singleValue float64
			hasSingleValue := false
			if len(rangeArr) == 1 {
				switch v := rangeArr[0].(type) {
				case float64:
					singleValue = v
					hasSingleValue = true
				case duckdb.Interval:
					singleValue = float64(formatInterval(v))
					hasSingleValue = true
				}
			}
			if gauge, ok := row[gaugeIndex].(duckdb.Union); ok {
				switch v := gauge.Value.(type) {
				case float64:
					gaugeValue = v
				case duckdb.Interval:
					isInterval = true
				}
			}
			if hasSingleValue && singleValue > 0 && gaugeValue >= 0 {
				rangeArr = []any{0.0, singleValue}
			} else if isInterval {
				// 1 hour in milliseconds
				rangeArr = []any{0.0, float64(60 * 60 * 1000)}
			} else if isGaugePercent && gaugeValue >= 0 && gaugeValue <= 1 {
				rangeArr = []any{0.0, 1.0}
			} else {
				absValue := math.Abs(gaugeValue)
				var nextPower float64 = 10.0
				if absValue > 0 {
					nextPower = math.Pow(10, math.Ceil(math.Log10(absValue)))
				}
				if gaugeValue < 0 {
					rangeArr = []any{-nextPower, nextPower}
				} else if gaugeValue > 0 {
					rangeArr = []any{0.0, nextPower}
				} else {
					rangeArr = []any{0.0, 10.0}
				}
			}
		}
		labelsArr := []any{}
		if labelsCol != nil {
			if labelsUnion, ok := row[labelsIndex].(duckdb.Union); ok {
				if arr, ok := labelsUnion.Value.([]any); ok {
					labelsArr = arr
				}
			}
		}
		// TODO: warn if labels length doesn't match range length
		labelsLen := len(labelsArr)
		colorsArr := []any{}
		if colorsCol != nil {
			if colorsUnion, ok := row[colorsIndex].(duckdb.Union); ok {
				if arr, ok := colorsUnion.Value.([]any); ok {
					colorsArr = arr
				}
			}
		}
		// TODO: warn if colors length doesn't match range length
		colorsLen := len(colorsArr)
		categories := []GaugeCategory{}
		fromAny := rangeArr[0]
		from, ok := fromAny.(float64)
		if !ok {
			from = float64(formatInterval(fromAny))
		}
		for i := 1; i < len(rangeArr); i++ {
			toAny := rangeArr[i]
			to, ok := toAny.(float64)
			if !ok {
				to = float64(formatInterval(toAny))
			}
			g := GaugeCategory{
				From: from,
				To:   to,
			}
			if labelsLen >= i {
				if labelValue, ok := labelsArr[i-1].(string); ok {
					g.Label = labelValue
				}
			}
			if colorsLen >= i {
				if colorValue, ok := colorsArr[i-1].(string); ok {
					g.Color = colorValue
				}
			}
			categories = append(categories, g)
			from = to
		}
		r := renderInfo{
			Label:           labelValue,
			Type:            "gauge",
			ValueAxisIndex:  &gaugeIndex,
			GaugeCategories: categories,
		}
		return r
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
	hasYear := false
	hasMonth := false
	hasDay := false
	hasHour := false
	hasMSN := false
	for _, row := range rows {
		r := row[index]
		if r == nil {
			continue
		}
		t, ok := r.(time.Time)
		if !ok {
			return "", fmt.Errorf("invalid timestamp value: %v", row[index])
		}
		if t.Minute() != 0 || t.Second() != 0 || t.Nanosecond() != 0 {
			hasMSN = true
		}
		if t.Hour() != 0 {
			hasHour = true
		}
		if t.Year() != 1 {
			hasYear = true
		}
		if t.Month() != 1 {
			hasMonth = true
		}
		if t.Day() != 1 {
			hasDay = true
		}
		if hasMSN && (hasYear || hasMonth || hasDay) {
			// timestamp is the only type that allows to stop checking values early
			// for the rest we have to check all values to be sure
			return "timestamp", nil
		}
	}
	if len(rows) < 2 {
		return "timestamp", nil
	}
	if !hasDay && !hasMonth && !hasYear && (hasHour || hasMSN) {
		return "time", nil
	}
	if hasMSN {
		return "timestamp", nil
	}
	if hasHour {
		return "hour", nil
	}
	if hasDay {
		return "date", nil
	}
	if hasMonth {
		return "month", nil
	}
	return "year", nil
}

// TODO: We can make this more performant for TIME values since we can know if it's time by checking union.Tag
func getUnionTimestampType(rows Rows, index int) (string, error) {
	hasYear := false
	hasMonth := false
	hasDay := false
	hasHour := false
	hasMSN := false
	for _, row := range rows {
		r := row[index]
		if r == nil {
			continue
		}
		u, ok := r.(duckdb.Union)
		if !ok {
			return "", fmt.Errorf("invalid timestamp union value: %v", row[index])
		}
		t, ok := u.Value.(time.Time)
		if !ok {
			return "", fmt.Errorf("invalid timestamp value: %v", row[index])
		}
		if t.Minute() != 0 || t.Second() != 0 || t.Nanosecond() != 0 {
			hasMSN = true
		}
		if t.Hour() != 0 {
			hasHour = true
		}
		if t.Year() != 1 {
			hasYear = true
		}
		if t.Month() != 1 {
			hasMonth = true
		}
		if t.Day() != 1 {
			hasDay = true
		}
		if hasMSN && (hasYear || hasMonth || hasDay) {
			// timestamp is the only type that allows to stop checking values early
			// for the rest we have to check all values to be sure
			return "timestamp", nil
		}
	}
	if len(rows) < 2 {
		return "timestamp", nil
	}
	if !hasDay && !hasMonth && !hasYear && (hasHour || hasMSN) {
		return "time", nil
	}
	if hasMSN {
		return "timestamp", nil
	}
	if hasHour {
		return "hour", nil
	}
	if hasDay {
		return "date", nil
	}
	if hasMonth {
		return "month", nil
	}
	return "year", nil
}

func getChartType(rows Rows, index int) (string, error) {
	if len(rows) == 0 {
		return "number", nil
	}
	if union, ok := rows[0][index].(duckdb.Union); ok {
		if _, ok := union.Value.(duckdb.Interval); ok {
			return "duration", nil
		}
	}
	return "number", nil
}

func getAxisType(rows Rows, index int) (string, error) {
	if len(rows) == 0 {
		return "string", nil
	}
	// Try timestamp first
	if s, err := getUnionTimestampType(rows, index); err == nil {
		return s, nil
	}
	// Then try number and fallback to string
	for _, row := range rows {
		union, ok := row[index].(duckdb.Union)
		if !ok {
			return "", fmt.Errorf("invalid union value for axis value, got: %v (type %T, column %v)", row[index], row[index], index)
		}
		if strings.HasSuffix(union.Tag, "_interval") {
			return "duration", nil
		}
		if strings.HasSuffix(union.Tag, "_time") {
			return "time", nil
		}
		if strings.HasSuffix(union.Tag, "_double") {
			return "number", nil
		}
	}
	return "string", nil
}

// TODO: assert that variable names are alphanumeric
// TODO: test and harden variable escaping
// TODO: assert that variables in query are set. otherwise it silently falls back to empty string
// NOTE: Technically we don't need to reset variables since we are not reusing connections. I just have a better feeling with this.
func buildVarPrefix(singleVars map[string]string, multiVars map[string][]string) (string, string) {
	varPrefix := strings.Builder{}
	varCleanup := strings.Builder{}
	for k, v := range singleVars {
		varPrefix.WriteString(fmt.Sprintf("SET VARIABLE \"%s\" = %s;\n", util.EscapeSQLIdentifier(k), v))
		varCleanup.WriteString(fmt.Sprintf("RESET VARIABLE \"%s\";\n", util.EscapeSQLIdentifier(k)))
	}
	for k, v := range multiVars {
		l := ""
		for i, p := range v {
			prefix := ", "
			if i == 0 {
				prefix = ""
			}
			l += fmt.Sprintf("%s'%s'", prefix, util.EscapeSQLString(p))
		}
		varPrefix.WriteString(fmt.Sprintf("SET VARIABLE \"%s\" = [%s]::VARCHAR[];\n", util.EscapeSQLIdentifier(k), l))
		varCleanup.WriteString(fmt.Sprintf("RESET VARIABLE \"%s\";\n", util.EscapeSQLIdentifier(k)))
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
		if param != "" {
			// Check if param actually exists in the dropdown
			isValidVar := false
			for i, row := range data {
				union, ok := row[columnIndex].(duckdb.Union)
				if !ok {
					return fmt.Errorf("invalid union value for dropdown value, got: %v (type %t, row, %v, column %v)", row[columnIndex], row[columnIndex], i, columnIndex)
				}
				val, ok := union.Value.(string)
				if !ok {
					if union.Value == nil {
						val = ""
					} else {
						return fmt.Errorf("invalid string value for dropdown value, got: %v (type %t, row, %v, column %v)", row[columnIndex], row[columnIndex], i, columnIndex)
					}
				}
				if val == param {
					isValidVar = true
					break
				}
			}
			if !isValidVar {
				// Ignore invalid param
				param = ""
			}
		}
		if param == "" {
			if len(data) == 0 {
				// No vars for dropdown without options
				return nil
			}
			// Set default value to first row
			union, ok := data[0][columnIndex].(duckdb.Union)
			if !ok {
				return fmt.Errorf("invalid union value as first value for default dropdown value, got: %v (type %T, column %v)", data[0][columnIndex], data[0][columnIndex], columnIndex)
			}
			param, ok = union.Value.(string)
			if !ok {
				if union.Value == nil {
					param = ""
				} else {
					return fmt.Errorf("invalid string value as first value for default dropdown value, got: %v (type %T, column %v)", data[0][columnIndex], data[0][columnIndex], columnIndex)
				}
			}
		}
		singleVars[columnName] = "'" + util.EscapeSQLString(param) + "'"
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
		if len(params) > 0 {
			paramsToCheck := map[string]bool{}
			for _, param := range params {
				paramsToCheck[param] = true
			}
			for i, row := range data {
				union, ok := row[columnIndex].(duckdb.Union)
				var val string
				if !ok {
					return fmt.Errorf("invalid union value for dropdown-multi value, got: %v (type %T, row %v, column %v)", row[columnIndex], row[columnIndex], i, columnIndex)
				}
				val, ok = union.Value.(string)
				if !ok {
					if union.Value == nil {
						val = ""
					} else {
						return fmt.Errorf("invalid string value for dropdown-multi value, got: %v (type %T, row %v, column %v)", row[columnIndex], row[columnIndex], i, columnIndex)
					}
				}
				if paramsToCheck[val] {
					delete(paramsToCheck, val)
					if len(paramsToCheck) == 0 {
						break
					}
				}
			}
			if len(paramsToCheck) > 0 {
				// Remove invalid params
				cleanedParams := make([]string, 0, len(params)-len(paramsToCheck))
				for _, param := range params {
					if paramsToCheck[param] {
						continue
					}
					cleanedParams = append(cleanedParams, param)
				}
				params = cleanedParams
			}
		}
		if len(params) == 0 {
			// Set default value to all rows
			for i, row := range data {
				union, ok := row[columnIndex].(duckdb.Union)
				if !ok {
					return fmt.Errorf("invalid union value for default dropdown-multi value, got: %v (type %T, row %v, column %v)", row[columnIndex], row[columnIndex], i, columnIndex)
				} else {
					val, ok := union.Value.(string)
					if !ok {
						if union.Value == nil {
							val = ""
						} else {
							return fmt.Errorf("invalid string value for default dropdown-multi value, got: %v (type %T, row %v, column %v)", row[columnIndex], row[columnIndex], i, columnIndex)
						}
					}
					params = append(params, val)
				}
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
				val := data[0][defaultValueIndex].(duckdb.Union).Value
				if val != nil {
					date := val.(time.Time)
					param = date.Format(time.DateOnly)
				}
			}
		} else {
			// Check if param is a valid date
			if !isDateString(param) {
				return fmt.Errorf("invalid date for datepicker query param '%s': %s", columnName, param)
			}
		}
		if param != "" {
			singleVars[columnName] = "DATE '" + util.EscapeSQLString(param) + "'"
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
				val := data[0][fromDefaultValueIndex].(duckdb.Union).Value
				if val != nil {
					date := val.(time.Time)
					fromParam = date.Format(time.DateOnly)
				}
			}
		} else {
			// Check if fromParam is a valid date
			if !isDateString(fromParam) {
				return fmt.Errorf("invalid date for datepicker query fromParam '%s': %s", fromColumnName, fromParam)
			}
		}
		if fromParam != "" {
			singleVars[fromColumnName] = "TIMESTAMP '" + util.EscapeSQLString(fromParam) + "'"
		}
		toParam := queryParams.Get(toColumnName)
		if toParam == "" {
			// Set default value
			if toDefaultValueIndex != -1 {
				val := data[0][toDefaultValueIndex].(duckdb.Union).Value
				if val != nil {
					date := val.(time.Time)
					toParam = date.Format(time.DateOnly)
				}
			}
		} else {
			// Check if toParam is a valid date
			if !isDateString(toParam) {
				return fmt.Errorf("invalid date for datepicker query toParam '%s': %s", toColumnName, toParam)
			}
		}
		if toParam != "" {
			singleVars[toColumnName] = "TIMESTAMP '" + util.EscapeSQLString(toParam) + " 23:59:59.999999'"
		}
	}
	return nil
}

func isDateString(stringDate string) bool {
	_, err := time.Parse(time.DateOnly, stringDate)
	return err == nil
}

func getTokenVars(variables map[string]any) (map[string]string, map[string][]string, error) {
	singleVars := map[string]string{}
	multiVars := map[string][]string{}
	for k, v := range variables {
		switch v := v.(type) {
		case string:
			singleVars[k] = "'" + util.EscapeSQLString(v) + "'"
		case []any:
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
func formatInterval(v any) int64 {
	interval := v.(duckdb.Interval)
	ms := interval.Micros / 1000
	ms += int64(interval.Days) * 24 * 60 * 60 * 1000
	ms += int64(interval.Months) * 30 * 24 * 60 * 60 * 1000
	return ms
}

// Convert time to ms since midnight
func formatTime(t time.Time) int64 {
	seconds := t.Hour()*3600 + t.Minute()*60 + t.Second()
	return int64(seconds*1000) + int64(t.Nanosecond()/1000000)
}

// Must be a single column and a single row.
// Row value must be a timestamp or interval.
// Column type must the be custom RELOAD type
func isReload(columns []*sql.ColumnType, rows Rows) bool {
	col, _ := findColumnByTag(columns, "RELOAD")
	if col == nil {
		return false
	}
	return (len(rows) == 0 || (len(rows) == 1 && len(rows[0]) == 1))
}

func getReloadValue(rows Rows) int64 {
	if len(rows) == 0 {
		return 0
	}
	row := rows[0]
	if len(row) == 0 {
		return 0
	}
	val := rows[0][0]
	if len(row) == 0 || val == nil {
		return 0
	}
	if union, ok := val.(duckdb.Union); ok {
		if interval, ok := union.Value.(duckdb.Interval); ok {
			return time.Now().Add(time.Millisecond * time.Duration(formatInterval(interval))).UnixMilli()
		}
		if t, ok := union.Value.(time.Time); ok {
			// Convert to milliseconds since epoch
			return t.UnixMilli()
		}
		return 0
	}
	return 0
}

func isHeaderImage(columns []*sql.ColumnType, rows Rows) bool {
	col, _ := findColumnByTag(columns, "HEADER_IMAGE")
	if col == nil {
		return false
	}
	return len(rows) == 1 && len(rows[0]) == 1
}
func isFooterLink(columns []*sql.ColumnType, rows Rows) bool {
	col, _ := findColumnByTag(columns, "FOOTER_LINK")
	if col == nil {
		return false
	}
	return len(rows) == 1 && len(rows[0]) == 1
}

func getSingleValue(rows Rows) string {
	if len(rows) == 0 {
		return ""
	}
	row := rows[0]
	if len(row) == 0 {
		return ""
	}
	val := rows[0][0]
	if len(row) == 0 || val == nil {
		return ""
	}
	if union, ok := val.(duckdb.Union); ok {
		if str, ok := union.Value.(string); ok {
			return str
		}
		return ""
	}
	return ""
}

func lessThanTwoUniqueRangeValues(r []any) bool {
	if len(r) < 2 {
		return true
	}
	uniqueValues := make(map[float64]bool)
	for _, v := range r {
		switch v := v.(type) {
		case float64:
			uniqueValues[v] = true
		case duckdb.Interval:
			ms := float64(formatInterval(v))
			uniqueValues[ms] = true
		default:
			return true // Unsupported type
		}
		if len(uniqueValues) >= 2 {
			return false
		}
	}
	return true
}
