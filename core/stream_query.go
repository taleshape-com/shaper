package core

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

func StreamQueryCSV(
	app *App,
	ctx context.Context,
	dashboardName string,
	params url.Values,
	queryID string,
	variables map[string]interface{},
	writer io.Writer,
) error {
	fileName := path.Join(app.DashboardDir, dashboardName+".sql")
	// read sql file
	sqlFile, err := os.ReadFile(fileName)
	if err != nil {
		return err
	}
	sqls := strings.Split(string(sqlFile), ";")

	queryIndex, err := strconv.Atoi(queryID)
	if err != nil {
		return err
	}
	if len(sqls) <= queryIndex || queryIndex < 0 {
		return fmt.Errorf("dashboard '%s' has no query for query index: %d", dashboardName, queryIndex)
	}
	if queryIndex < 1 || !isDownloadButton(sqls[queryIndex-1]) {
		return fmt.Errorf("query must be download query")
	}
	query := sqls[queryIndex]

	// Create a CSV writer
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	conn, err := app.db.Connx(ctx)
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
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
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
	if closeErr := conn.Close(); closeErr != nil {
		return fmt.Errorf("Error closing conn %d: %v", closeErr)
	}

	return rows.Err()
}

// formatValue converts various types to their string representation
func formatValue(value interface{}) string {
	if value == nil {
		return ""
	}

	switch v := value.(type) {
	case []byte:
		return string(v)
	case time.Time:
		return v.Format(time.RFC3339)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func getVarPrefix(conn *sqlx.Conn, ctx context.Context, sqlQueries []string, queryParams url.Values, variables map[string]interface{}) (string, string, error) {
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
				return "", "", err
			}
			data = append(data, row)
		}

		if isLabel(sqlString, data) || isSectionTitle(sqlString, data) {
			continue
		}

		rInfo := getRenderInfo(colTypes, data, sqlString, "")

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
