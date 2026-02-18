package challenge

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTernary(t *testing.T) {
	tests := []struct {
		name string
		cond bool
		t    string
		f    string
		want string
	}{
		{"true returns t", true, "yes", "no", "yes"},
		{"false returns f", false, "yes", "no", "no"},
		{"true with empty f", true, "value", "", "value"},
		{"false with empty t", false, "", "fallback", "fallback"},
		{"both empty true", true, "", "", ""},
		{"both empty false", false, "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Ternary(tt.cond, tt.t, tt.f))
		})
	}
}
