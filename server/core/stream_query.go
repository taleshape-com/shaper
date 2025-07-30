package core

import (
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"net/url"
	"shaper/server/util"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/marcboeker/go-duckdb/v2"
	"github.com/xuri/excelize/v2"
)

// 1 day = 24 hours = 24 * 60 * 60 seconds = 24 * 60 * 60 * 1_000_000 micros
const MICROSECONDS_PER_DAY = 24.0 * 60.0 * 60.0 * 1_000_000.0

const EXCEL_INTERVAL_FORMAT = "[h]:mm:ss"

// Stream the result of a dashboard query as CSV file to client.
// Same as dashboard, it handles variables from JWT and from URL params.
func StreamQueryCSV(
	app *App,
	ctx context.Context,
	dashboardId string,
	params url.Values,
	queryID string,
	variables map[string]any,
	writer io.Writer,
) error {
	dashboard, err := GetDashboardQuery(app, ctx, dashboardId)
	if err != nil {
		return fmt.Errorf("error getting dashboard: %w", err)
	}
	cleanContent := util.StripSQLComments(dashboard.Content)
	sqls, err := splitSQLQueries(cleanContent)
	if err != nil {
		return err
	}

	queryIndex, err := strconv.Atoi(queryID)
	if err != nil {
		return err
	}
	if len(sqls) <= queryIndex || queryIndex < 0 {
		return fmt.Errorf("dashboard '%s' has no query for query index: %d", dashboardId, queryIndex)
	}
	query := sqls[queryIndex]

	// Create a CSV writer
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	conn, err := app.DB.Connx(ctx)
	if err != nil {
		return fmt.Errorf("Error getting conn: %v", err)
	}

	// Execute the query and get rows
	varPrefix, varCleanup, err := getVarPrefix(conn, ctx, sqls, params, variables)
	if err != nil {
		return err
	}
	rows, err := conn.QueryContext(ctx, varPrefix+query+";")
	if varCleanup != "" {
		if _, cleanupErr := conn.ExecContext(ctx, varCleanup); cleanupErr != nil {
			return fmt.Errorf("Error cleaning up vars in query %d: %v", queryIndex, cleanupErr)
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
			if closeErr := rows.Close(); closeErr != nil {
				return fmt.Errorf("Error closing rows after scan error. Scan Err: %v. Close Err: %v", err, closeErr)
			}
			return fmt.Errorf("error scanning row: %w", err)
		}

		// Convert values to strings
		record := make([]string, len(columns))
		for i, value := range values {
			record[i] = formatValue(value)
		}

		// Write the record
		if err := csvWriter.Write(record); err != nil {
			if closeErr := rows.Close(); closeErr != nil {
				return fmt.Errorf("Error closing rows after csv write error. CSV Err: %v. Close Err: %v", err, closeErr)
			}
			return fmt.Errorf("error writing record: %w", err)
		}

		// Flush periodically to ensure streaming
		csvWriter.Flush()
		if err := csvWriter.Error(); err != nil {
			if closeErr := rows.Close(); closeErr != nil {
				return fmt.Errorf("Error closing rows after csv flush error. CSV Err: %v. Close Err: %v", err, closeErr)
			}
			return fmt.Errorf("error flushing CSV writer: %w", err)
		}
	}
	if err := conn.Close(); err != nil {
		return fmt.Errorf("Error closing conn: %v", err)
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
	queryID string,
	variables map[string]any,
	writer io.Writer,
) error {
	// Get dashboard content
	dashboard, err := GetDashboardQuery(app, ctx, dashboardId)
	if err != nil {
		return fmt.Errorf("error getting dashboard: %w", err)
	}
	cleanContent := util.StripSQLComments(dashboard.Content)
	sqls, err := splitSQLQueries(cleanContent)
	if err != nil {
		return err
	}

	queryIndex, err := strconv.Atoi(queryID)
	if err != nil {
		return err
	}
	if len(sqls) <= queryIndex || queryIndex < 0 {
		return fmt.Errorf("dashboard '%s' has no query for query index: %d", dashboardId, queryIndex)
	}
	query := sqls[queryIndex]

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

	conn, err := app.DB.Connx(ctx)
	if err != nil {
		return fmt.Errorf("Error getting conn: %v", err)
	}

	// Execute the query and get rows
	varPrefix, varCleanup, err := getVarPrefix(conn, ctx, sqls, params, variables)
	if err != nil {
		return err
	}
	rows, err := conn.QueryContext(ctx, varPrefix+query+";")
	if varCleanup != "" {
		if _, cleanupErr := conn.ExecContext(ctx, varCleanup); cleanupErr != nil {
			return fmt.Errorf("Error cleaning up vars in query %d: %v", queryIndex, cleanupErr)
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

func getVarPrefix(conn *sqlx.Conn, ctx context.Context, sqlQueries []string, queryParams url.Values, variables map[string]any) (string, string, error) {
	nextIsDownload := false
	// TODO: currently variables have to be defined in the order they are used. create a dependency graph for queryies instead
	singleVars, multiVars, err := getTokenVars(variables)
	if err != nil {
		return "", "", err
	}

	for queryIndex, sqlString := range sqlQueries {
		if queryIndex == len(sqlQueries)-1 {
			// Ignore text after last semicolon
			break
		}
		if nextIsDownload {
			nextIsDownload = false
			continue
		}
		varPrefix, varCleanup := buildVarPrefix(singleVars, multiVars)
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

		rInfo := getRenderInfo(colTypes, data, "")

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
	varPrefix, varCleanup := buildVarPrefix(singleVars, multiVars)
	return varPrefix, varCleanup, nil
}

func isDownloadButton(columns []*sql.ColumnType) bool {
	return getDownloadType(columns) != ""
}
