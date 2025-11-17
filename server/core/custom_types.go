// SPDX-License-Identifier: MPL-2.0

package core

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// Custom SQL types Shaper provides that can be used to build rich dashboards.
// Internally they are all UNION types so we can identify them in the query result,
// and so that they work transparently with other SQL types.
//
// TODO: Support DATE, INTERVAL and potential other types in axis
// TODO: Support DATE, TIME, INTERVAL, DOUBLE in CATEGORY
var dbTypes = []struct {
	Name       string
	Definition string
	ResultType string
}{
	{"LABEL", "UNION(\"label_varchar\" VARCHAR)", "string"},
	{"XAXIS", "UNION(\"xaxis_varchar\" VARCHAR, \"xaxis_timestamp\" TIMESTAMP, \"xaxis_time\" TIME, \"xaxis_double\" DOUBLE, \"xaxis_interval\" INTERVAL)", "axis"},
	{"YAXIS", "UNION(\"yaxis_varchar\" VARCHAR, \"xaxis_timestamp\" TIMESTAMP, \"yaxis_time\" TIME, \"yaxis_double\" DOUBLE, \"yaxis_interval\" INTERVAL)", "axis"},
	{"XLINE", "UNION(\"xline_varchar\" VARCHAR, \"xline_timestamp\" TIMESTAMP, \"xline_time\" TIME, \"xline_double\" DOUBLE, \"xline_interval\" INTERVAL)", "axis"},
	{"YLINE", "UNION(\"yline_timestamp\" TIMESTAMP, \"yline_time\" TIME, \"yline_double\" DOUBLE, \"yline_interval\" INTERVAL)", "axis"},
	{"LINECHART", "UNION(\"linechart_interval\" INTERVAL, \"linechart_double\" DOUBLE)", "chart"},
	{"LINECHART_PERCENT", "UNION(\"linechart_percent_double\" DOUBLE)", "percent"},
	{"LINECHART_CATEGORY", "UNION(\"linechart_category_varchar\" VARCHAR)", "string"},
	{"BARCHART", "UNION(\"barchart_interval\" INTERVAL, \"barchart_double\" DOUBLE)", "chart"},
	{"BARCHART_PERCENT", "UNION(\"barchart_percent_double\" DOUBLE)", "percent"},
	{"BARCHART_STACKED", "UNION(\"barchart_stacked_interval\" INTERVAL, \"barchart_stacked_double\" DOUBLE)", "chart"},
	{"BARCHART_STACKED_PERCENT", "UNION(\"barchart_stacked_percent\" DOUBLE)", "percent"},
	// Alias for BARCHART_STACKED_PERCENT
	{"BARCHART_PERCENT_STACKED", "UNION(\"barchart_stacked_percent\" DOUBLE)", "percent"},
	{"BARCHART_CATEGORY", "UNION(\"barchart_category_varchar\" VARCHAR)", "string"},
	{"CATEGORY", "UNION(\"category_varchar\" VARCHAR)", "string"},
	{"DROPDOWN", "UNION(\"dropdown_varchar\" VARCHAR)", "string"},
	{"DROPDOWN_MULTI", "UNION(\"dropdown_multi_varchar\" VARCHAR)", "string"},
	{"HINT", "UNION(\"hint_varchar\" VARCHAR)", "string"},
	{"SECTION", "UNION(\"section_varchar\" VARCHAR)", "string"},
	{"DOWNLOAD_CSV", "UNION(\"download_csv_varchar\" VARCHAR)", "string"},
	{"DOWNLOAD_XLSX", "UNION(\"download_xlsx_varchar\" VARCHAR)", "string"},
	{"DOWNLOAD_PDF", "UNION(\"download_pdf_varchar\" VARCHAR)", "string"},
	{"DATEPICKER", "UNION(\"datepicker_date\" DATE)", "date"},
	{"DATEPICKER_FROM", "UNION(\"datepicker_from_date\" DATE)", "date"},
	{"DATEPICKER_TO", "UNION(\"datepicker_to_date\" DATE)", "date"},
	{"COMPARE", "UNION(\"compare_double\" DOUBLE)", "number"},
	{"TREND", "UNION(\"trend_double\" DOUBLE)", "number"},
	{"PLACEHOLDER", "UNION(\"placeholder_varchar\" VARCHAR)", "string"},
	{"INPUT", "UNION(\"input_varchar\" VARCHAR)", "string"},
	{"PERCENT", "UNION(\"percent_double\" DOUBLE)", "percent"},
	{"RELOAD", "UNION(\"reload_timestamp\" TIMESTAMP, \"reload_timestamptz\" TIMESTAMPTZ, \"reload_interval\" INTERVAL)", "timestamp"},
	{"SCHEDULE", "UNION(\"schedule_timestamp\" TIMESTAMP, \"schedule_timestamptz\" TIMESTAMPTZ, \"schedule_interval\" INTERVAL)", "timestamp"},
	{"SCHEDULE_ALL", "UNION(\"schedule_all_timestamp\" TIMESTAMP, \"schedule_all_timestamptz\" TIMESTAMPTZ, \"schedule_all_interval\" INTERVAL)", "timestamp"},
	{"GAUGE", "UNION(\"gauge_interval\" INTERVAL, \"gauge_double\" DOUBLE)", "chart"},
	{"GAUGE_PERCENT", "UNION(\"gauge_percent\" DOUBLE)", "percent"},
	{"RANGE", "UNION(\"range_interval\" INTERVAL[], \"range_double\" DOUBLE[])", "array"},
	{"LABELS", "UNION(\"labels_varchar\" VARCHAR[])", "array"},
	{"COLORS", "UNION(\"colors_varchar\" VARCHAR[])", "array"},
	{"COLOR", "UNION(\"color_varchar\" VARCHAR)", "string"},
	{"LINECHART_COLOR", "UNION(\"linechart_color_varchar\" VARCHAR)", "string"},
	{"BARCHART_COLOR", "UNION(\"barchart_color_varchar\" VARCHAR)", "string"},
	{"HEADER_IMAGE", "UNION(\"header_image_varchar\" VARCHAR)", "string"},
	{"FOOTER_LINK", "UNION(\"footer_link_varchar\" VARCHAR)", "string"},
	{"ID", "UNION(\"id_varchar\" VARCHAR)", "string"},
}

func createType(db *sqlx.DB, name string, definition string) error {
	// drop types first
	_, err := db.Exec("DROP TYPE IF EXISTS " + name + ";")
	if err != nil {
		return fmt.Errorf("failed to drop type %s: %w", name, err)
	}
	_, err = db.Exec("CREATE TYPE " + name + " AS " + definition + ";")
	if err != nil && err.Error() != "Catalog Error: Type with name \""+name+"\" already exists!" {
		return fmt.Errorf("failed to create type %s: %w", name, err)
	}
	return nil
}

const boxplotType = `STRUCT("max" DOUBLE, "min" DOUBLE, "outliers" STRUCT("value" DOUBLE, "info" MAP(VARCHAR, VARCHAR))[], "q1" DOUBLE, "q2" DOUBLE, "q3" DOUBLE)`

const boxplotFunction = `
CREATE OR REPLACE MACRO BOXPLOT(val, outlier_info := NULL) AS CASE
  WHEN count(*) filter(WHERE outlier_info IS NOT NULL) > 0 THEN
    {
      'max': list(val).filter(lambda v: v <= quantile_cont(val, 0.75) + 1.5 * (quantile_cont(val, 0.75) - quantile_cont(val, 0.25))).list_max()::DOUBLE,
      'min': list(val).filter(lambda v: v >= quantile_cont(val, 0.25) - 1.5 * (quantile_cont(val, 0.75) - quantile_cont(val, 0.25))).list_min()::DOUBLE,
      'outliers': list({ value: val, info: outlier_info }).filter(lambda outlier:
        outlier.value < quantile_cont(val, 0.25) - 1.5 * (quantile_cont(val, 0.75) - quantile_cont(val, 0.25))
        OR
        outlier.value > quantile_cont(val, 0.75) + 1.5 * (quantile_cont(val, 0.75) - quantile_cont(val, 0.25))
      )::STRUCT(value DOUBLE, info MAP(VARCHAR, VARCHAR))[],
      'q1': quantile_cont(val, 0.25)::DOUBLE,
      'q2': quantile_cont(val, 0.5)::DOUBLE,
      'q3': quantile_cont(val, 0.75)::DOUBLE,
    }
  ELSE
    {
      'max': max(val)::DOUBLE,
      'min': min(val)::DOUBLE,
      'outliers': []::STRUCT(value DOUBLE, info MAP(VARCHAR, VARCHAR))[],
      'q1': quantile_cont(val, 0.25)::DOUBLE,
      'q2': quantile_cont(val, 0.5)::DOUBLE,
      'q3': quantile_cont(val, 0.75)::DOUBLE,
    }
END;
`

func createBoxlotFunction(db *sqlx.DB) error {
	_, err := db.Exec(boxplotFunction)
	return err
}
