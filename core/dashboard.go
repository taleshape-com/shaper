package core

type ListResult struct {
	Dashboards []string `json:"dashboards"`
}

type GetResult struct {
	Title    string    `json:"title"`
	Sections []Section `json:"sections"`
}

type Section struct {
	Title   *string `json:"title"`
	Type    string  `json:"type"`
	Queries []Query `json:"queries"`
}

type Query struct {
	Render  Render          `json:"render"`
	Columns []Column        `json:"columns"`
	Rows    [][]interface{} `json:"rows"`
}

type Render struct {
	Type  string  `json:"type"`
	Label *string `json:"label"`
}

type renderInfo struct {
	Type          string
	Label         *string
	XAxisIndex    *int
	YAxisIndex    *int
	CategoryIndex *int
	ValueIndex    *int
	LabelIndex    *int
	HintIndex     *int
	Download      string
}

//	type Desc struct {
//		ColumnName string  `db:"column_name"`
//		ColumnType string  `db:"column_type"`
//		Null       string  `db:"null"`
//		Key        *string `db:"key"`
//		Default    *string `db:"default"`
//		Extra      *string `db:"extra"`
//	}
type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Tag      string `json:"tag"`
}
