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

type Rows [][]interface{}

type Query struct {
	Render  Render   `json:"render"`
	Columns []Column `json:"columns"`
	Rows    Rows     `json:"rows"`
}

type Render struct {
	Type  string  `json:"type"`
	Label *string `json:"label"`
}

type renderInfo struct {
	Type           string
	Label          *string
	IndexAxisIndex *int
	ValueAxisIndex *int
	CategoryIndex  *int
	ValueIndex     *int
	FromIndex      *int
	ToIndex        *int
	LabelIndex     *int
	HintIndex      *int
	Download       string
	CompareIndex   *int
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
