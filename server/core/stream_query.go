// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/url"
	"regexp"
	"shaper/server/util"
	"strings"
	"time"

	"github.com/duckdb/duckdb-go/v2"
	"github.com/jmoiron/sqlx"
	"github.com/xuri/excelize/v2"
)

// 1 day = 24 hours = 24 * 60 * 60 seconds = 24 * 60 * 60 * 1_000_000 micros
const MICROSECONDS_PER_DAY = 24.0 * 60.0 * 60.0 * 1_000_000.0

const EXCEL_INTERVAL_FORMAT = "[h]:mm:ss"

var excludedTypesRegex = regexp.MustCompile(`\b(LABEL|SECTION|XLINE|YLINE|DROPDOWN|DOWNLOAD_CSV|DOWNLOAD_XLSX|DOWNLOAD_JSON|DOWNLOAD_PDF|DATEPICKER|DATEPICKER_FROM|DATEPICKER_TO|PLACEHOLDER|INPUT|RELOAD|HEADER_IMAGE|FOOTER_LINK)\b`)

func resolveDownloadQueryID(app *App, sqls []string, downloadType string) (int, error) {
	upperType := "DOWNLOAD_" + strings.ToUpper(downloadType)
	foundIndex := -1
	count := 0
	for i, s := range sqls {
		if strings.Contains(strings.ToUpper(s), upperType) {
			foundIndex = i
			count++
		}
	}
	if count == 1 {
		return foundIndex + 1, nil
	}

	foundIndex = -1
	count = 0
	for i, s := range sqls {
		if isSideEffect(app, s) {
			continue
		}
		upper := strings.ToUpper(s)
		if !excludedTypesRegex.MatchString(upper) {
			foundIndex = i
			count++
		}
	}
	if count == 1 {
		return foundIndex, nil
	}

	if count == 0 {
		return -1, fmt.Errorf("could not find a matching query for %s download", strings.ToUpper(downloadType))
	}
	return -1, fmt.Errorf("found %d potential queries for %s download, please specify which one with query_id", count, strings.ToUpper(downloadType))
}

// Stream the result of a dashboard query as CSV file to client.
// Same as dashboard, it handles variables from JWT and from URL params.
func StreamQueryCSV(
	app *App,
	ctx context.Context,
	dashboardId string,
	params url.Values,
	queryID int,
	variables map[string]any,
	writer io.Writer,
) error {
	dashboard, err := GetDashboardInfo(app, ctx, dashboardId)
	if err != nil {
		return fmt.Errorf("error getting dashboard: %w", err)
	}
	cleanContent := util.StripSQLComments(dashboard.Content)
	sqls, err := util.SplitSQLQueries(cleanContent)
	if err != nil {
		return fmt.Errorf("failed to split SQL queries: %w", err)
	}

	if queryID == -1 {
		queryID, err = resolveDownloadQueryID(app, sqls, "csv")
		if err != nil {
			return err
		}
	}

	if len(sqls) <= queryID || queryID < 0 {
		return fmt.Errorf("dashboard '%s' has no query for query index: %d", dashboardId, queryID)
	}
	query := sqls[queryID]
	if !IsAllowedStatement(app, query) {
		return fmt.Errorf("disallowed SQL statement in query %d", queryID+1)
	}

	db, cleanup, err := app.GetDuckDB(ctx)
	if err != nil {
		return fmt.Errorf("Error getting DB: %v", err)
	}
	defer cleanup()
	conn, err := db.Connx(ctx)
	if err != nil {
		return fmt.Errorf("Error getting conn: %v", err)
	}
	defer conn.Close()

	// Execute the query and get rows
	varPrefix, varCleanup, err := getVarPrefix(app, conn, ctx, sqls, params, variables)
	if err != nil {
		return fmt.Errorf("failed to get variable prefix: %w", err)
	}
	defer func() {
		if varCleanup != "" {
			if _, cleanupErr := conn.ExecContext(ctx, varCleanup); cleanupErr != nil {
				app.Logger.ErrorContext(ctx, "Error cleaning up vars", "error", cleanupErr)
			}
		}
	}()

	return StreamSQLToCSVWithConn(conn, ctx, varPrefix+query+";", writer)
}

// StreamSQLToCSV executes a single SQL query and streams the result as CSV.
func StreamSQLToCSV(
	app *App,
	ctx context.Context,
	sqlQuery string,
	writer io.Writer,
) error {
	if !IsAllowedStatement(app, sqlQuery) {
		return fmt.Errorf("disallowed SQL statement")
	}
	db, cleanup, err := app.GetDuckDB(ctx)
	if err != nil {
		return fmt.Errorf("Error getting DB: %v", err)
	}
	defer cleanup()
	conn, err := db.Connx(ctx)
	if err != nil {
		return fmt.Errorf("Error getting conn: %v", err)
	}
	defer conn.Close()

	varPrefix, _ := buildVarPrefix(app, nil, nil)
	return StreamSQLToCSVWithConn(conn, ctx, varPrefix+sqlQuery, writer)
}

// Stream the result of a dashboard query as JSON file to client.
// Same as dashboard, it handles variables from JWT and from URL params.
func StreamQueryJSON(
	app *App,
	ctx context.Context,
	dashboardId string,
	params url.Values,
	queryID int,
	variables map[string]any,
	writer io.Writer,
) error {
	dashboard, err := GetDashboardInfo(app, ctx, dashboardId)
	if err != nil {
		return fmt.Errorf("error getting dashboard: %w", err)
	}
	cleanContent := util.StripSQLComments(dashboard.Content)
	sqls, err := util.SplitSQLQueries(cleanContent)
	if err != nil {
		return fmt.Errorf("failed to split SQL queries: %w", err)
	}

	if queryID == -1 {
		queryID, err = resolveDownloadQueryID(app, sqls, "json")
		if err != nil {
			return err
		}
	}

	if len(sqls) <= queryID || queryID < 0 {
		return fmt.Errorf("dashboard '%s' has no query for query index: %d", dashboardId, queryID)
	}
	query := sqls[queryID]
	if !IsAllowedStatement(app, query) {
		return fmt.Errorf("disallowed SQL statement in query %d", queryID+1)
	}

	db, cleanup, err := app.GetDuckDB(ctx)
	if err != nil {
		return fmt.Errorf("Error getting DB: %v", err)
	}
	defer cleanup()
	conn, err := db.Connx(ctx)
	if err != nil {
		return fmt.Errorf("Error getting conn: %v", err)
	}
	defer conn.Close()

	// Execute the query and get rows
	varPrefix, varCleanup, err := getVarPrefix(app, conn, ctx, sqls, params, variables)
	if err != nil {
		return fmt.Errorf("failed to get variable prefix: %w", err)
	}
	defer func() {
		if varCleanup != "" {
			if _, cleanupErr := conn.ExecContext(ctx, varCleanup); cleanupErr != nil {
				app.Logger.ErrorContext(ctx, "Error cleaning up vars", "error", cleanupErr)
			}
		}
	}()

	return StreamSQLToJSONWithConn(conn, ctx, varPrefix+query+";", writer)
}

// StreamSQLToJSON executes a single SQL query and streams the result as JSON.
func StreamSQLToJSON(
	app *App,
	ctx context.Context,
	sqlQuery string,
	writer io.Writer,
) error {
	if !IsAllowedStatement(app, sqlQuery) {
		return fmt.Errorf("disallowed SQL statement")
	}
	db, cleanup, err := app.GetDuckDB(ctx)
	if err != nil {
		return fmt.Errorf("Error getting DB: %v", err)
	}
	defer cleanup()
	conn, err := db.Connx(ctx)
	if err != nil {
		return fmt.Errorf("Error getting conn: %v", err)
	}
	defer conn.Close()

	varPrefix, _ := buildVarPrefix(app, nil, nil)
	return StreamSQLToJSONWithConn(conn, ctx, varPrefix+sqlQuery, writer)
}

// StreamSQLToJSONWithConn executes a single SQL query using an existing connection and streams the result as JSON.
func StreamSQLToJSONWithConn(
	conn *sqlx.Conn,
	ctx context.Context,
	sqlQuery string,
	writer io.Writer,
) error {
	rows, err := conn.QueryContext(ctx, sqlQuery)
	if err != nil {
		return fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("error getting columns: %w", err)
	}

	if _, err := writer.Write([]byte("[")); err != nil {
		return fmt.Errorf("error writing JSON start: %w", err)
	}

	// Prepare containers for row data
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	first := true
	// Stream rows
	for rows.Next() {
		if !first {
			if _, err := writer.Write([]byte(",")); err != nil {
				return fmt.Errorf("error writing JSON separator: %w", err)
			}
		}
		first = false

		// Scan the row into our value containers
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("error scanning row: %w", err)
		}

		// Convert values to map
		rowMap := make(map[string]any)
		for i, colName := range columns {
			rowMap[colName] = jsonValue(values[i])
		}

		// Write the record
		encoder := json.NewEncoder(writer)
		if err := encoder.Encode(rowMap); err != nil {
			return fmt.Errorf("error encoding JSON record: %w", err)
		}
	}

	if _, err := writer.Write([]byte("]")); err != nil {
		return fmt.Errorf("error writing JSON end: %w", err)
	}

	return rows.Err()
}

func jsonValue(value any) any {
	if value == nil {
		return nil
	}

	switch v := value.(type) {
	case []byte:
		if isUUID(v) {
			return formatUUID(v)
		}
		return string(v)
	case duckdb.Interval:
		return intervalToString(v)
	case duckdb.Union:
		return jsonValue(v.Value)
	case []any:
		res := make([]any, len(v))
		for i, item := range v {
			res[i] = jsonValue(item)
		}
		return res
	default:
		return v
	}
}

// StreamSQLToCSVWithConn executes a single SQL query using an existing connection and streams the result as CSV.
func StreamSQLToCSVWithConn(
	conn *sqlx.Conn,
	ctx context.Context,
	sqlQuery string,
	writer io.Writer,
) error {
	// Create a CSV writer
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	rows, err := conn.QueryContext(ctx, sqlQuery)
	if err != nil {
		return fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("error getting columns: %w", err)
	}

	// Write header
	if err := csvWriter.Write(columns); err != nil {
		return fmt.Errorf("error writing headers: %w", err)
	}

	// Prepare containers for row data
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Stream rows
	for rows.Next() {
		// Scan the row into our value containers
		if err := rows.Scan(valuePtrs...); err != nil {
			return fmt.Errorf("error scanning row: %w", err)
		}

		// Convert values to strings
		record := make([]string, len(columns))
		for i, value := range values {
			record[i] = formatValue(value)
		}

		// Write the record
		if err := csvWriter.Write(record); err != nil {
			return fmt.Errorf("error writing record: %w", err)
		}

		// Flush periodically to ensure streaming
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			return fmt.Errorf("error flushing CSV writer: %w", err)
		}
	}

	return rows.Err()
}

// Stream the result of a dashboard query as CSV file to client.
// Same as dashboard, it handles variables from JWT and from URL params.
// We format headers and different values in a way that is suitable for Excel.
// Note that while the interface is streaming, the whole file is rendered in memory.
// This is a restriction of the excelize library and maybe of the XLSX format itself.
//
// TODO: Limit file size and for large files tell user to use CSV instead.
func StreamQueryXLSX(
	app *App,
	ctx context.Context,
	dashboardId string,
	params url.Values,
	queryID int,
	variables map[string]any,
	writer io.Writer,
) error {
	// Get dashboard content
	dashboard, err := GetDashboardInfo(app, ctx, dashboardId)
	if err != nil {
		return fmt.Errorf("error getting dashboard: %w", err)
	}
	cleanContent := util.StripSQLComments(dashboard.Content)
	sqls, err := util.SplitSQLQueries(cleanContent)
	if err != nil {
		return fmt.Errorf("failed to split SQL queries: %w", err)
	}

	if queryID == -1 {
		queryID, err = resolveDownloadQueryID(app, sqls, "xlsx")
		if err != nil {
			return err
		}
	}

	if len(sqls) <= queryID || queryID < 0 {
		return fmt.Errorf("dashboard '%s' has no query for query index: %d", dashboardId, queryID)
	}
	query := sqls[queryID]
	if !IsAllowedStatement(app, query) {
		return fmt.Errorf("disallowed SQL statement in query %d", queryID+1)
	}

	// Create a new XLSX file
	xlsx := excelize.NewFile()
	// TODO: Support specifying sheet name
	sheetName := "Sheet1"

	// Create header style with bold font
	headerStyle, err := xlsx.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		return fmt.Errorf("error creating header style: %w", err)
	}

	styles := map[string]int{
		"datetime": createStyle(xlsx, &excelize.Style{
			NumFmt: 22, // "m/d/yy h:mm"
			Alignment: &excelize.Alignment{
				Horizontal: "center",
			},
		}),
		"number": createStyle(xlsx, &excelize.Style{
			Alignment: &excelize.Alignment{
				Horizontal: "right",
			},
		}),
		"text": createStyle(xlsx, &excelize.Style{
			Alignment: &excelize.Alignment{
				Horizontal: "left",
				WrapText:   true,
			},
		}),
		"interval": createStyle(xlsx, &excelize.Style{
			NumFmt: 46, // "[h]:mm:ss"
			Alignment: &excelize.Alignment{
				Horizontal: "center",
			},
		}),
	}

	db, cleanup, err := app.GetDuckDB(ctx)
	if err != nil {
		return fmt.Errorf("Error getting DB: %v", err)
	}
	defer cleanup()
	conn, err := db.Connx(ctx)
	if err != nil {
		return fmt.Errorf("Error getting conn: %v", err)
	}

	// Execute the query and get rows
	varPrefix, varCleanup, err := getVarPrefix(app, conn, ctx, sqls, params, variables)
	if err != nil {
		return fmt.Errorf("failed to get variable prefix: %w", err)
	}
	rows, err := conn.QueryContext(ctx, varPrefix+query+";")
	if varCleanup != "" {
		if _, cleanupErr := conn.ExecContext(ctx, varCleanup); cleanupErr != nil {
			return fmt.Errorf("Error cleaning up vars in query %d: %v", queryID, cleanupErr)
		}
	}
	if err != nil {
		return fmt.Errorf("error executing query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("error getting columns: %w", err)
	}

	// Initialize maxWidths slice to track maximum width for each column
	maxWidths := make([]float64, len(columns))

	// Write headers
	for colIdx, column := range columns {
		cell, err := excelize.CoordinatesToCellName(colIdx+1, 1)
		if err != nil {
			return fmt.Errorf("error converting coordinates: %w", err)
		}
		xlsx.SetCellValue(sheetName, cell, column)
		xlsx.SetCellStyle(sheetName, cell, cell, headerStyle)
		maxWidths[colIdx] = float64(len(column)) + 2 // Start with header width + padding
	}

	// Prepare containers for row data
	values := make([]any, len(columns))
	valuePtrs := make([]any, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Stream rows
	rowIdx := 2 // Start from row 2 (after headers)
	for rows.Next() {
		// Scan the row into our value containers
		if err := rows.Scan(valuePtrs...); err != nil {
			if closeErr := rows.Close(); closeErr != nil {
				return fmt.Errorf("Error closing rows after scan error. Scan Err: %v. Close Err: %v", err, closeErr)
			}
			return fmt.Errorf("error scanning row: %w", err)
		}

		// Write values to cells
		for colIdx, value := range values {
			cell, err := excelize.CoordinatesToCellName(colIdx+1, rowIdx)
			if err != nil {
				if closeErr := rows.Close(); closeErr != nil {
					return fmt.Errorf("Error closing rows after excel error. Excel Err: %w. Close Err: %v", err, closeErr)
				}
				return fmt.Errorf("error converting coordinates: %w", err)
			}
			// Apply appropriate formatting based on data type
			handleCellValue(value, xlsx, sheetName, cell, styles)

			// Update maximum width for this column
			width := getDisplayWidth(value) + 2 // +2 for padding
			maxWidths[colIdx] = math.Max(maxWidths[colIdx], width)
		}

		rowIdx++
	}

	if err := conn.Close(); err != nil {
		return fmt.Errorf("Error closing conn: %v", err)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	// Set column widths based on content
	for colIdx := range columns {
		colName, err := excelize.ColumnNumberToName(colIdx + 1)
		if err != nil {
			return fmt.Errorf("error converting column number: %w", err)
		}
		// Clamp width between minimum of 6 and maximum of 100
		width := math.Max(6, math.Min(100, maxWidths[colIdx]))
		xlsx.SetColWidth(sheetName, colName, colName, width)
	}

	// Set up autofilter
	lastCol, _ := excelize.ColumnNumberToName(len(columns))
	filterRange := fmt.Sprintf("A1:%s%d", lastCol, rowIdx-1)
	xlsx.AutoFilter(sheetName, filterRange, []excelize.AutoFilterOptions{})

	// Freeze the header row
	xlsx.SetPanes(sheetName, &excelize.Panes{
		Freeze:      true,
		Split:       false,
		XSplit:      0,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
	})

	// Write the XLSX file to the writer
	return xlsx.Write(writer)
}

// getDisplayWidth returns the approximate display width of a value.
// This is a simple implementation that could be enhanced for better accuracy.
func getDisplayWidth(value any) float64 {
	if value == nil {
		return 4 // Width of "null"
	}

	switch v := value.(type) {
	case time.Time:
		return 20 // Approximate width for RFC3339 format
	case duckdb.Interval:
		return float64(len(intervalToString(v)))
	case duckdb.Union:
		return getDisplayWidth(v.Value)
	default:
		str := fmt.Sprintf("%v", value)
		return float64(len(str))
	}
}

func handleCellValue(value any, xlsx *excelize.File, sheetName string, cell string, styles map[string]int) {
	if value == nil {
		xlsx.SetCellValue(sheetName, cell, "")
		xlsx.SetCellStyle(sheetName, cell, cell, styles["text"])
		return
	}
	switch v := value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		xlsx.SetCellValue(sheetName, cell, v)
		xlsx.SetCellStyle(sheetName, cell, cell, styles["number"])
	case time.Time:
		xlsx.SetCellValue(sheetName, cell, v)
		xlsx.SetCellStyle(sheetName, cell, cell, styles["datetime"])
	case duckdb.Interval:
		xlsx.SetCellFloat(sheetName, cell, intervalToDays(v), 6, 64) // 6 decimal places precision
		xlsx.SetCellStyle(sheetName, cell, cell, styles["interval"])
	default:
		xlsx.SetCellValue(sheetName, cell, formatValue(v))
		xlsx.SetCellStyle(sheetName, cell, cell, styles["text"])
	}
}

func isUUID(s []byte) bool {
	return len(s) == 16
}

// formatValue converts various types to their string representation
// TODO: We are not handling TIME values. Currently they are treated as timestamps with epoc date.
func formatValue(value any) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case []byte:
		if isUUID(v) {
			return formatUUID(v)
		}
		return string(v)
	case duckdb.Interval:
		return intervalToString(v)
	case time.Time:
		return v.Format(time.RFC3339)
	// handle list and array types
	case []any:
		var strValues []string
		for _, item := range v {
			strValues = append(strValues, formatValue(item))
		}
		return strings.Join(strValues, ", ")
	case duckdb.Union:
		return formatValue(v.Value)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Express an interval as a number of days with fractions
func intervalToDays(interval duckdb.Interval) float64 {
	// Handle months (approximate months to days)
	days := float64(interval.Days + (interval.Months * 30))
	// Convert micros to days
	daysFromMicros := float64(interval.Micros) / MICROSECONDS_PER_DAY
	return days + daysFromMicros
}

// string looks like "10d 5h 30m 15.068s"
func intervalToString(interval duckdb.Interval) string {
	var parts []string

	// Handle months (convert to days for simplicity)
	totalDays := interval.Days + (interval.Months * 30) // approximate months to days
	if totalDays != 0 {
		parts = append(parts, fmt.Sprintf("%dd", totalDays))
	}

	// Handle time components from micros
	remainingMicros := interval.Micros

	// Calculate hours
	hours := remainingMicros / (3600 * 1000000)
	if hours != 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
		remainingMicros -= hours * 3600 * 1000000
	}

	// Calculate minutes
	minutes := remainingMicros / (60 * 1000000)
	if minutes != 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
		remainingMicros -= minutes * 60 * 1000000
	}

	// Calculate seconds (including fractional part)
	seconds := float64(remainingMicros) / 1000000.0
	if seconds != 0 || len(parts) == 0 { // include 0s if no other parts
		parts = append(parts, fmt.Sprintf("%.3fs", seconds))
	}
	return strings.Join(parts, " ")
}

func createStyle(xlsx *excelize.File, style *excelize.Style) int {
	styleID, _ := xlsx.NewStyle(style)
	return styleID
}

func getVarPrefix(app *App, conn *sqlx.Conn, ctx context.Context, sqlQueries []string, queryParams url.Values, variables map[string]any) (string, string, error) {
	nextIsDownload := false
	// TODO: currently variables have to be defined in the order they are used. create a dependency graph for queryies instead
	singleVars, multiVars, err := getTokenVars(variables)
	if err != nil {
		return "", "", err
	}

	if app.InternalDBName != "" {
		searchPath := fmt.Sprintf("SET search_path = 'main,\"%s\".main,system';", util.EscapeSQLIdentifier(app.InternalDBName))
		if _, err := conn.ExecContext(ctx, searchPath); err != nil {
			return "", "", fmt.Errorf("Error setting search path: %v", err)
		}
	}

	for queryIndex, sqlString := range sqlQueries {
		sqlString = strings.TrimSpace(sqlString)
		if sqlString == "" {
			continue
		}
		if !IsAllowedStatement(app, sqlString) {
			return "", "", fmt.Errorf("disallowed SQL statement in query %d", queryIndex+1)
		}
		if nextIsDownload {
			nextIsDownload = false
			continue
		}
		varPrefix, varCleanup := buildVarPrefixNoSearchPath(app, singleVars, multiVars)
		// run query
		data := Rows{}
		rows, err := conn.QueryxContext(ctx, varPrefix+string(sqlString)+";")
		if varCleanup != "" {
			if _, cleanupErr := conn.ExecContext(ctx, varCleanup); cleanupErr != nil {
				return "", "", fmt.Errorf("Error cleaning up vars in query %d: %v", queryIndex, cleanupErr)
			}
		}
		if err != nil {
			return "", "", err
		}
		colTypes, err := rows.ColumnTypes()
		if err != nil {
			return "", "", err
		}
		for rows.Next() {
			row, err := rows.SliceScan()
			if err != nil {
				if closeErr := rows.Close(); closeErr != nil {
					return "", "", fmt.Errorf("Error closing rows after scan error. Scan Err: %v. Close Err: %v", err, closeErr)
				}
				return "", "", err
			}
			data = append(data, row)
		}

		if isLabel(colTypes, data) || isSectionTitle(colTypes, data) {
			continue
		}

		rInfo := getRenderInfo(colTypes, data, "", []MarkLine{})

		if rInfo.Download != "" {
			nextIsDownload = true
		}

		columns := []Column{}
		for colIndex, c := range colTypes {
			col := Column{
				Name: c.Name(),
				Tag:  mapTag(colIndex, rInfo),
			}
			columns = append(columns, col)
		}
		err = collectVars(singleVars, multiVars, rInfo.Type, queryParams, columns, data)
		if err != nil {
			return "", "", err
		}
	}
	varPrefix, varCleanup := buildVarPrefix(app, singleVars, multiVars)
	return varPrefix, varCleanup, nil
}

func isDownloadButton(columns []*sql.ColumnType) bool {
	return getDownloadType(columns) != ""
}
