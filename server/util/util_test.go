// SPDX-License-Identifier: MPL-2.0

package util

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRandomString(t *testing.T) {
	s1 := GenerateRandomString(32)
	s2 := GenerateRandomString(32)

	assert.Equal(t, 32, len(s1))
	assert.Equal(t, 32, len(s2))
	assert.NotEqual(t, s1, s2)

	assert.Equal(t, "", GenerateRandomString(0))
}

func TestIsValidVariableName(t *testing.T) {
	valid := []string{
		"a",
		"_a",
		"varName",
		"var_name_123",
		"_123",
		"VARIABLE",
		"a1b2c3",
		strings.Repeat("a", 64),
	}
	for _, name := range valid {
		assert.True(t, IsValidVariableName(name), "expected %q to be valid", name)
	}

	invalid := []string{
		"",
		"123var",
		"var-name",
		"var name",
		"var$name",
		"var;DROP TABLE;",
		"var'name",
		"var\"name",
		"var\nname",
		"var\x00name",
		strings.Repeat("a", 65),
	}
	for _, name := range invalid {
		assert.False(t, IsValidVariableName(name), "expected %q to be invalid", name)
	}
}

func TestStripSQLComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"No comments", "SELECT 1", "SELECT 1"},
		{"Line comment", "SELECT 1 -- comment\nSELECT 2", "SELECT 1 \nSELECT 2"},
		{"Block comment", "SELECT /* comment */ 1", "SELECT   1"},
		{"Nested block comment", "SELECT /* outer /* inner */ still outer */ 1", "SELECT   1"},
		{"Block comment with single quote", "SELECT /* ' */ 1", "SELECT   1"},
		{"Block comment with double quote", "SELECT /* \" */ 1", "SELECT   1"},
		{"Block comment preserves token separation", "SELECT/**/1", "SELECT 1"},
		{"Block comment inside single-quoted string", "SELECT '/* not a comment */' FROM t", "SELECT '/* not a comment */' FROM t"},
		{"Block comment inside double-quoted identifier", "SELECT \"/* not a comment */\" FROM t", "SELECT \"/* not a comment */\" FROM t"},
		{"Multiple block comments", "SELECT /* a */ 1, /* b */ 2", "SELECT   1,   2"},
		{"Unterminated block comment", "SELECT /* unterminated", "SELECT  "},
		{"Block comment with semicolon", "SELECT /* ; */ 1", "SELECT   1"},
		{"Block and line comments mixed", "SELECT /* block */ 1 -- line\nFROM t", "SELECT   1 \nFROM t"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, StripSQLComments(tt.input))
		})
	}
}

func TestSplitSQLQueriesBlockComments(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			"Block comment with quote does not affect splitting",
			"SELECT /* ' */ 1; SELECT 2",
			[]string{"SELECT /* ' */ 1", "SELECT 2"},
		},
		{
			"Block comment with semicolon does not split",
			"SELECT /* ; */ 1",
			[]string{"SELECT /* ; */ 1"},
		},
		{
			"Nested block comment",
			"SELECT /* /* ; */ ; */ 1; SELECT 2",
			[]string{"SELECT /* /* ; */ ; */ 1", "SELECT 2"},
		},
		{
			"Block comment injection attempt",
			"SELECT /* ' */ ; DROP TABLE foo; SELECT /* ' */ 1",
			[]string{"SELECT /* ' */ ", "DROP TABLE foo", "SELECT /* ' */ 1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitSQLQueries(tt.input)
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestEscapeSQLString(t *testing.T) {
	assert.Equal(t, "hello", EscapeSQLString("hello"))
	assert.Equal(t, "O''Reilly", EscapeSQLString("O'Reilly"))
	assert.Equal(t, "test test", EscapeSQLString("test\ntest"))
	assert.Equal(t, "test test", EscapeSQLString("test\rtest"))
	assert.Equal(t, "test", EscapeSQLString("te\x00st"))
	assert.Equal(t, "test", EscapeSQLString("te\x1ast"))
}

func TestEscapeSQLIdentifier(t *testing.T) {
	assert.Equal(t, "my_col", EscapeSQLIdentifier("my_col"))
	assert.Equal(t, "my\"\"col", EscapeSQLIdentifier("my\"col"))
	assert.Equal(t, "my col", EscapeSQLIdentifier("my\ncol"))
	assert.Equal(t, "my col", EscapeSQLIdentifier("my\rcol"))
	assert.Equal(t, "mycol", EscapeSQLIdentifier("my\x00col"))
}

