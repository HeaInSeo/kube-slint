package slint

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Normal string",
			input:    "my-test-case",
			expected: "my-test-case",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "unknown",
		},
		{
			name:     "Whitespace only",
			input:    "   \t\n  ",
			expected: "unknown", // TrimSpace removes everything, making it empty
		},
		{
			name:     "Contains spaces",
			input:    "my test case",
			expected: "my_test_case",
		},
		{
			name:     "Contains path separators",
			input:    "path/to\\file",
			expected: "path_to_file",
		},
		{
			name:     "Contains special chars",
			input:    `foo:"bar";'baz'`,
			expected: "foo__bar___baz_",
		},
		{
			name:     "Contains newlines and tabs",
			input:    "a\nb\rc\td",
			expected: "a_b_c_d",
		},
		{
			name:     "Dots and hidden files (currently preserved)",
			input:    "../.hidden.dir/file",
			expected: ".._.hidden.dir_file",
		},
		{
			name:     "Max length exactly 120",
			input:    strings.Repeat("a", 120),
			expected: strings.Repeat("a", 120),
		},
		{
			name:     "Max length exceeded",
			input:    strings.Repeat("a", 150),
			expected: strings.Repeat("a", 120),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("SanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSanitizeKubernetesLabelValue(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "  ", want: "unknown"},
		{name: "selector delimiters", in: "run/id, one", want: "run-id--one"},
		{name: "trim invalid edges", in: "...run...", want: "run"},
		{name: "unicode", in: "실행-1", want: "1"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, SanitizeKubernetesLabelValue(tc.in))
		})
	}

	long := SanitizeKubernetesLabelValue(strings.Repeat("a", 80))
	assert.Len(t, long, 63)
}
