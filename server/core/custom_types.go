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
	{"LABEL", "UNION(\"_shaper_label_varchar\" VARCHAR)", "string"},
	{"XAXIS", "UNION(\"_shaper_xaxis_varchar\" VARCHAR, \"_shaper_xaxis_timestamp\" TIMESTAMP, \"_shaper_xaxis_time\" TIME, \"_shaper_xaxis_double\" DOUBLE, \"_shaper_xaxis_interval\" INTERVAL)", "axis"},
	{"YAXIS", "UNION(\"_shaper_yaxis_varchar\" VARCHAR, \"_shaper_xaxis_timestamp\" TIMESTAMP, \"_shaper_yaxis_time\" TIME, \"_shaper_yaxis_double\" DOUBLE, \"_shaper_yaxis_interval\" INTERVAL)", "axis"},
	{"XLINE", "UNION(\"_shaper_xline_varchar\" VARCHAR, \"_shaper_xline_timestamp\" TIMESTAMP, \"_shaper_xline_time\" TIME, \"_shaper_xline_double\" DOUBLE, \"_shaper_xline_interval\" INTERVAL)", "axis"},
	{"YLINE", "UNION(\"_shaper_yline_timestamp\" TIMESTAMP, \"_shaper_yline_time\" TIME, \"_shaper_yline_double\" DOUBLE, \"_shaper_yline_interval\" INTERVAL)", "axis"},
	{"LINECHART", "UNION(\"_shaper_linechart_interval\" INTERVAL, \"_shaper_linechart_double\" DOUBLE)", "chart"},
	{"LINECHART_PERCENT", "UNION(\"_shaper_linechart_percent_double\" DOUBLE)", "percent"},
	{"LINECHART_CATEGORY", "UNION(\"_shaper_linechart_category_varchar\" VARCHAR)", "string"},
	{"BARCHART", "UNION(\"_shaper_barchart_interval\" INTERVAL, \"_shaper_barchart_double\" DOUBLE)", "chart"},
	{"BARCHART_PERCENT", "UNION(\"_shaper_barchart_percent_double\" DOUBLE)", "percent"},
	{"BARCHART_STACKED", "UNION(\"_shaper_barchart_stacked_interval\" INTERVAL, \"_shaper_barchart_stacked_double\" DOUBLE)", "chart"},
	{"BARCHART_STACKED_PERCENT", "UNION(\"_shaper_barchart_stacked_percent\" DOUBLE)", "percent"},
	// Alias for BARCHART_STACKED_PERCENT
	{"BARCHART_PERCENT_STACKED", "UNION(\"_shaper_barchart_stacked_percent\" DOUBLE)", "percent"},
	{"BARCHART_CATEGORY", "UNION(\"_shaper_barchart_category_varchar\" VARCHAR)", "string"},
	{"CATEGORY", "UNION(\"_shaper_category_varchar\" VARCHAR)", "string"},
	{"DROPDOWN", "UNION(\"_shaper_dropdown_varchar\" VARCHAR)", "string"},
	{"DROPDOWN_MULTI", "UNION(\"_shaper_dropdown_multi_varchar\" VARCHAR)", "string"},
	{"HINT", "UNION(\"_shaper_hint_varchar\" VARCHAR)", "string"},
	{"SECTION", "UNION(\"_shaper_section_varchar\" VARCHAR)", "string"},
	{"DOWNLOAD_CSV", "UNION(\"_shaper_download_csv_varchar\" VARCHAR)", "string"},
	{"DOWNLOAD_XLSX", "UNION(\"_shaper_download_xlsx_varchar\" VARCHAR)", "string"},
	{"DOWNLOAD_PDF", "UNION(\"_shaper_download_pdf_varchar\" VARCHAR)", "string"},
	{"DATEPICKER", "UNION(\"_shaper_datepicker_date\" DATE)", "date"},
	{"DATEPICKER_FROM", "UNION(\"_shaper_datepicker_from_date\" DATE)", "date"},
	{"DATEPICKER_TO", "UNION(\"_shaper_datepicker_to_date\" DATE)", "date"},
	{"COMPARE", "UNION(\"_shaper_compare_double\" DOUBLE)", "number"},
	{"TREND", "UNION(\"_shaper_trend_double\" DOUBLE)", "number"},
	{"PLACEHOLDER", "UNION(\"_shaper_placeholder_varchar\" VARCHAR)", "string"},
	{"INPUT", "UNION(\"_shaper_input_varchar\" VARCHAR)", "string"},
	{"PERCENT", "UNION(\"_shaper_percent_double\" DOUBLE)", "percent"},
	{"RELOAD", "UNION(\"_shaper_reload_timestamp\" TIMESTAMP, \"_shaper_reload_timestamptz\" TIMESTAMPTZ, \"_shaper_reload_interval\" INTERVAL)", "timestamp"},
	{"SCHEDULE", "UNION(\"_shaper_schedule_timestamp\" TIMESTAMP, \"_shaper_schedule_timestamptz\" TIMESTAMPTZ, \"_shaper_schedule_interval\" INTERVAL)", "timestamp"},
	{"SCHEDULE_ALL", "UNION(\"_shaper_schedule_all_timestamp\" TIMESTAMP, \"_shaper_schedule_all_timestamptz\" TIMESTAMPTZ, \"_shaper_schedule_all_interval\" INTERVAL)", "timestamp"},
	{"GAUGE", "UNION(\"_shaper_gauge_interval\" INTERVAL, \"_shaper_gauge_double\" DOUBLE)", "chart"},
	{"GAUGE_PERCENT", "UNION(\"_shaper_gauge_percent\" DOUBLE)", "percent"},
	{"RANGE", "UNION(\"_shaper_range_interval\" INTERVAL[], \"_shaper_range_double\" DOUBLE[])", "array"},
	{"LABELS", "UNION(\"_shaper_labels_varchar\" VARCHAR[])", "array"},
	{"COLORS", "UNION(\"_shaper_colors_varchar\" VARCHAR[])", "array"},
	{"COLOR", "UNION(\"_shaper_color_varchar\" VARCHAR)", "string"},
	{"LINECHART_COLOR", "UNION(\"_shaper_linechart_color_varchar\" VARCHAR)", "string"},
	{"BARCHART_COLOR", "UNION(\"_shaper_barchart_color_varchar\" VARCHAR)", "string"},
	{"HEADER_IMAGE", "UNION(\"_shaper_header_image_varchar\" VARCHAR)", "string"},
	{"FOOTER_LINK", "UNION(\"_shaper_footer_link_varchar\" VARCHAR)", "string"},
	{"ID", "UNION(\"_shaper_id_varchar\" VARCHAR)", "string"},
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
