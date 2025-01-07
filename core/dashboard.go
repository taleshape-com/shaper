package core

import "time"

type Dashboard struct {
	ID        string    `db:"id" json:"id"`
	Path      string    `db:"path" json:"path"`
	Name      string    `db:"name" json:"name"`
	Content   string    `db:"content" json:"content"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
	CreatedBy *string   `db:"created_by" json:"createdBy,omitempty"`
	UpdatedBy *string   `db:"updated_by" json:"updatedBy,omitempty"`
}

type ListResult struct {
	Dashboards []Dashboard `json:"dashboards"`
}

type GetResult struct {
	Name         string    `json:"name"`
	Sections     []Section `json:"sections"`
	MinTimeValue int64     `json:"minTimeValue"`
	MaxTimeValue int64     `json:"maxTimeValue"`
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
	TrendIndex     *int
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
