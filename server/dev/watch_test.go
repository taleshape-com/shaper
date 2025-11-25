package dev

import "testing"

func TestHasLeadingShaperIDComment(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "missing comment",
			content:  "select 1",
			expected: false,
		},
		{
			name:     "valid comment with newline",
			content:  "-- shaperid:ckb0example12345678901234\nselect 1",
			expected: true,
		},
		{
			name:     "valid comment without newline",
			content:  "-- shaperid:ckb0example12345678901234",
			expected: true,
		},
		{
			name:     "comment with trailing spaces is invalid",
			content:  "-- shaperid:ckb0example12345678901234   \nselect 1",
			expected: false,
		},
		{
			name:     "comment with extra text is invalid",
			content:  "-- shaperid:ckb0example12345678901234 extra\nselect 1",
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasLeadingShaperIDComment(tc.content); got != tc.expected {
				t.Fatalf("expected %v, got %v for content %q", tc.expected, got, tc.content)
			}
		})
	}
}

func TestPrependShaperIDComment(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "content without leading newline gets newline inserted",
			content:  "select 1",
			expected: "-- shaperid:testid\nselect 1",
		},
		{
			name:     "content with leading newline keeps newline",
			content:  "\nselect 1",
			expected: "-- shaperid:testid\nselect 1",
		},
		{
			name:     "empty content still ends with newline",
			content:  "",
			expected: "-- shaperid:testid\n",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := prependShaperIDComment("testid", tc.content)
			if got != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}
