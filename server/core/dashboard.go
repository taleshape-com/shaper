// SPDX-License-Identifier: MPL-2.0

package core

import "time"

type Dashboard struct {
	ID         string    `db:"id" json:"id"`
	Path       string    `db:"path" json:"path"`
	Name       string    `db:"name" json:"name"`
	Content    string    `db:"content" json:"content"`
	CreatedAt  time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt  time.Time `db:"updated_at" json:"updatedAt"`
	CreatedBy  *string   `db:"created_by" json:"createdBy,omitempty"`
	UpdatedBy  *string   `db:"updated_by" json:"updatedBy,omitempty"`
	Visibility *string   `db:"visibility" json:"visibility,omitempty"`
}

type GetResult struct {
	Name         string    `json:"name"`
	Visibility   *string   `json:"visibility,omitempty"`
	Sections     []Section `json:"sections"`
	MinTimeValue int64     `json:"minTimeValue"`
	MaxTimeValue int64     `json:"maxTimeValue"`
	ReloadAt     int64     `json:"reloadAt"`
}

type Section struct {
	Title   *string `json:"title"`
	Type    string  `json:"type"`
	Queries []Query `json:"queries"`
}

type Rows [][]any

type Query struct {
	Render  Render   `json:"render"`
	Columns []Column `json:"columns"`
	Rows    Rows     `json:"rows"`
}

type Render struct {
	Type            string          `json:"type"`
	Label           *string         `json:"label"`
	GaugeCategories []GaugeCategory `json:"gaugeCategories,omitempty"`
}

type GaugeCategory struct {
	From  float64 `json:"from"`
	To    float64 `json:"to"`
	Label string  `json:"label,omitempty"`
	Color string  `json:"color,omitempty"`
}

type renderInfo struct {
	Type            string
	Label           *string
	IndexAxisIndex  *int
	ValueAxisIndex  *int
	CategoryIndex   *int
	ColorIndex      *int
	ValueIndex      *int
	FromIndex       *int
	ToIndex         *int
	LabelIndex      *int
	HintIndex       *int
	Download        string
	CompareIndex    *int
	TrendIndex      *int
	GaugeCategories []GaugeCategory
}

type Column struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
	Tag      string `json:"tag"`
}
