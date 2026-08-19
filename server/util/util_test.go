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

