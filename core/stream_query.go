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

func StreamQueryCSV(
	app *App,
	ctx context.Context,
	dashboardName string,
	params url.Values,
	queryID string,
	variables map[string][]string,
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
	query := sqls[queryIndex]

	// Create a CSV writer
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Execute the query and get rows
	varPrefix, err := getVarPrefix(app, ctx, sqls, params, variables)
	if err != nil {
		return err
	}
	rows, err := app.db.QueryContext(ctx, varPrefix+query+";")
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

func getVarPrefix(app *App, ctx context.Context, sqlQueries []string, queryParams url.Values, variables map[string][]string) (string, error) {
	nextIsDownload := false
	// TODO: currently variables have to be defined in the order they are used. create a dependency graph for queryies instead
	singleVars := map[string]string{}
	multiVars := map[string][]string{}

	for queryIndex, sqlString := range sqlQueries {
		if queryIndex == len(sqlQueries)-1 {
			// Ignore text after last semicolon
			break
		}
		if nextIsDownload {
			nextIsDownload = false
			continue
		}
		varPrefix := buildVarPrefix(singleVars, multiVars)
		// run query
		data := Rows{}
		rows, err := app.db.QueryxContext(ctx, varPrefix+string(sqlString)+";")
		if err != nil {
			return "", err
		}
		colTypes, err := rows.ColumnTypes()
		if err != nil {
			return "", err
		}
		for rows.Next() {
			row, err := rows.SliceScan()
			if err != nil {
				return "", err
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
		err = collectVars(singleVars, multiVars, rInfo.Type, queryParams, columns, data, variables)
		if err != nil {
			return "", err
		}
	}
	return buildVarPrefix(singleVars, multiVars), nil
}
