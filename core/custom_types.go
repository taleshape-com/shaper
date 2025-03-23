package core

import "github.com/jmoiron/sqlx"

// TODO: Consider making _shaper_ prefix configurable
// TODO: Support DATE, TIME, INTERVAL and potential other types in axis
// TODO: Support DATE, TIME, INTERVAL, DOUBLE in CATEGORY
var dbTypes = []struct {
	Name       string
	Definition string
	ResultType string
}{
	{"LABEL", "UNION(_shaper_label_varchar VARCHAR)", "string"},
	{"XAXIS", "UNION(_shaper_xaxis_varchar VARCHAR, _shaper_xaxis_timestamp TIMESTAMP, _shaper_xaxis_double DOUBLE)", "axis"},
	{"YAXIS", "UNION(_shaper_yaxis_varchar VARCHAR, _shaper_yaxis_timestamp TIMESTAMP, _shaper_yaxis_double DOUBLE)", "axis"},
	{"LINECHART", "UNION(_shaper_linechart_interval INTERVAL, _shaper_linechart_double DOUBLE)", "chart"},
	{"LINECHART_PERCENT", "UNION(_shaper_linechart_percent_double DOUBLE)", "percent"},
	{"LINECHART_CATEGORY", "UNION(_shaper_linechart_category_varchar VARCHAR)", "string"},
	{"BARCHART", "UNION(_shaper_barchart_interval INTERVAL, _shaper_barchart_double DOUBLE)", "chart"},
	{"BARCHART_PERCENT", "UNION(_shaper_barchart_percent_double DOUBLE)", "percent"},
	{"BARCHART_STACKED", "UNION(_shaper_barchart_stacked_interval INTERVAL, _shaper_barchart_stacked_double DOUBLE)", "chart"},
	{"BARCHART_STACKED_PERCENT", "UNION(_shaper_barchart_stacked_percent DOUBLE)", "percent"},
	{"BARCHART_CATEGORY", "UNION(_shaper_barchart_category_varchar VARCHAR)", "string"},
	{"CATEGORY", "UNION(_shaper_category_varchar VARCHAR)", "string"},
	{"DROPDOWN", "UNION(_shaper_dropdown_varchar VARCHAR)", "string"},
	{"DROPDOWN_MULTI", "UNION(_shaper_dropdown_multi_varchar VARCHAR)", "string"},
	{"HINT", "UNION(_shaper_hint_varchar VARCHAR)", "string"},
	{"SECTION", "UNION(_shaper_section_varchar VARCHAR)", "string"},
	{"DOWNLOAD_CSV", "UNION(_shaper_download_csv_varchar VARCHAR)", "string"},
	{"DOWNLOAD_XLSX", "UNION(_shaper_download_xlsx_varchar VARCHAR)", "string"},
	{"DATEPICKER", "UNION(_shaper_datepicker_date DATE)", "date"},
	{"DATEPICKER_FROM", "UNION(_shaper_datepicker_from_date DATE)", "date"},
	{"DATEPICKER_TO", "UNION(_shaper_datepicker_to_date DATE)", "date"},
	{"COMPARE", "UNION(_shaper_compare_double DOUBLE)", "number"},
	{"TREND", "UNION(_shaper_trend_double DOUBLE)", "number"},
	{"PLACEHOLDER", "UNION(_shaper_placeholder_varchar VARCHAR)", "string"},
	{"PERCENT", "UNION(_shaper_percent_double DOUBLE)", "percent"},
}

func createType(db *sqlx.DB, name string, definition string) error {
	_, err := db.Exec("CREATE TYPE " + name + " AS " + definition + ";")
	if err != nil && err.Error() != "Catalog Error: Type with name \""+name+"\" already exists!" {
		return err
	}
	return nil
}
