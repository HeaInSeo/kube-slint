package slint

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsEnabledByEnv(t *testing.T) {
	tests := []struct {
		name  string
		value string // empty string means env var is unset
		unset bool
		want  bool
	}{
		{name: "unset defaults to enabled", unset: true, want: true},
		{name: "empty string defaults to enabled", value: "", want: true},
		{name: "explicit true", value: "true", want: true},
		{name: "explicit 1", value: "1", want: true},
		{name: "arbitrary value treated as enabled", value: "yes", want: true},
		{name: "0 disables", value: "0", want: false},
		{name: "false disables", value: "false", want: false},
		{name: "FALSE disables (case-insensitive)", value: "FALSE", want: false},
		{name: "False disables (mixed case)", value: "False", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unset {
				t.Setenv("SLINT_ENABLED", "")
			} else {
				t.Setenv("SLINT_ENABLED", tt.value)
			}
			assert.Equal(t, tt.want, isEnabledByEnv())
		})
	}
}
