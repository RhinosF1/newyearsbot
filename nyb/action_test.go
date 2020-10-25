package nyb

import (
	"fmt"
	"testing"
)

func TestNormalize(t *testing.T) {
	tt := []struct {
		input  string
		output string
	}{
		{"", ""},
		{" ", ""},
		{"a", "a"},
		{"a ", "a"},
		{" a ", "a"},
		{"a  a", "a a"},
		{" a  a ", "a a"},
		{"A", "a"},
	}
	for i, tc := range tt {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			if got := normalize(tc.input); got != tc.output {
				t.Errorf("expected %v; got %v", tc.output, got)
			}
		})
	}
}
