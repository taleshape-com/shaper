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
)

func StreamQuery(
	app *App,
	ctx context.Context,
	dashboardName string,
	params url.Values,
	queryID string,
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
	// TODO: handle invalid index
	query := sqls[queryIndex]

	// Create a CSV writer
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Execute the query and get rows
	rows, err := app.db.QueryContext(ctx, query+";")
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
