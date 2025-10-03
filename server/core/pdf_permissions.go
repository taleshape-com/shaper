// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"fmt"
	"net/url"
	"shaper/server/util"
	"strings"

	"github.com/marcboeker/go-duckdb/v2"
)

// When downloading a PDF we also need to allow access if the users has the permission to see the parent dashboard
// TODO: This shares a lot of code with QueryDashboard
func DashboardContainsMatchingPdfDownload(app *App, ctx context.Context, parentDashboardId string, pdfDashboardId string, queryParams url.Values, variables map[string]any) (bool, error) {
	dashboardQuery, err := GetDashboardQuery(app, ctx, parentDashboardId)
	if err != nil {
		return false, err
	}
	nextIsDownload := false
	cleanContent := util.StripSQLComments(dashboardQuery.Content)
	sqls, err := util.SplitSQLQueries(cleanContent)
	if err != nil {
		return false, err
	}

	singleVars, multiVars, err := getTokenVars(variables)
	if err != nil {
		return false, err
	}

	conn, err := app.DuckDB.Connx(ctx)
	if err != nil {
		return false, fmt.Errorf("Error getting conn: %v", err)
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
				return false, fmt.Errorf("Error cleaning up vars in query %d: %v", queryIndex, cleanupErr)
			}
		}
		if err != nil {
			return false, fmt.Errorf("Error querying DB in query %d: %v", queryIndex, err)
		}
		colTypes, err := rows.ColumnTypes()
		if err != nil {
			return false, err
		}
		for rows.Next() {
			row, err := rows.SliceScan()
			if err != nil {
				if closeErr := rows.Close(); closeErr != nil {
					return false, fmt.Errorf("Error closing rows after scan error. Scan Err: %v. Close Err: %v", err, closeErr)
				}
				return false, err
			}
			query.Rows = append(query.Rows, row)
			if len(query.Rows) > QUERY_MAX_ROWS {
				// TODO: add a warning to the result and show to user
				app.Logger.InfoContext(ctx, "Query result too large, truncating", "dashboard", parentDashboardId, "queryIndex", queryIndex, "maxRows", QUERY_MAX_ROWS)
				if err := rows.Close(); err != nil {
					return false, fmt.Errorf("Error closing rows while truncating (dashboard '%v'): %v", parentDashboardId, err)
				}
				break
			}
		}

		if isSideEffect(sqlString) || isLabel(colTypes, query.Rows) || isSectionTitle(colTypes, query.Rows) || isReload(colTypes, query.Rows) || isHeaderImage(colTypes, query.Rows) || isFooterLink(colTypes, query.Rows) {
			continue
		}
		if _, ok := getMarkLines(colTypes, query.Rows); ok {
			continue
		}

		rInfo := getRenderInfo(colTypes, query.Rows, "", []MarkLine{})
		if rInfo.Download == "csv" || rInfo.Download == "xlsx" {
			nextIsDownload = true
		}

		timeColumnIndices := map[int]bool{}

		for colIndex, c := range colTypes {
			nullable, ok := c.Nullable()
			tag := mapTag(colIndex, rInfo)
			colType, err := mapDBType(c.DatabaseTypeName(), colIndex, query.Rows)
			if err != nil {
				return false, err
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
			if tag == "download" && len(query.Rows) > 0 {
				if rInfo.Download == "pdf" {
					if rInfo.DownloadIdIndex != nil {
						v := query.Rows[0][*rInfo.DownloadIdIndex]
						if v != nil {
							id := v.(duckdb.Union).Value.(string)
							if id == pdfDashboardId {
								return true, nil
							}
						}
					}
				}
			}
		}

		err = collectVars(singleVars, multiVars, rInfo.Type, queryParams, query.Columns, query.Rows)
		if err != nil {
			return false, err
		}
	}

	if err := conn.Close(); err != nil {
		return false, fmt.Errorf("Error closing conn: %v", err)
	}
	return false, err
}
