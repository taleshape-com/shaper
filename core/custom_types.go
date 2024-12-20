package core

import "github.com/jmoiron/sqlx"

var dbTypes = []struct {
	Name       string
	Definition string
}{
	{"LABEL", "VARCHAR"},
	{"XAXIS", "VARCHAR"},
	{"YAXIS", "VARCHAR"},
	{"LINECHART", "DOUBLE"},
	{"LINECHART_CATEGORY", "VARCHAR"},
	{"BARCHART", "DOUBLE"},
	{"BARCHART_STACKED", "DOUBLE"},
	{"BARCHART_CATEGORY", "VARCHAR"},
	{"DROPDOWN", "VARCHAR"},
	{"DROPDOWN_MULTI", "VARCHAR"},
	{"HINT", "VARCHAR"},
	{"SECTION", "VARCHAR"},
	{"DOWNLOAD_CSV", "VARCHAR"},
	{"DOWNLOAD_XLSX", "VARCHAR"},
	{"DATEPICKER", "DATE"},
	{"DATEPICKER_FROM", "DATE"},
	{"DATEPICKER_TO", "DATE"},
	{"COMPARE", "DOUBLE"},
	{"TREND", "DOUBLE"},
	{"PLACEHOLDER", "VARCHAR"},
}

func createType(db *sqlx.DB, name string, definition string) error {
	_, err := db.Exec("CREATE TYPE " + name + " AS " + definition + ";")
	if err != nil && err.Error() != "Catalog Error: Type with name \""+name+"\" already exists!" {
		return err
	}
	return nil
}

func getTypeByDefinition(definition string) string {
	for _, t := range dbTypes {
		if t.Definition == definition {
			return t.Name
		}
	}
	return ""
}
