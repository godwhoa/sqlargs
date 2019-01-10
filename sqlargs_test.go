package sqlargs

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestQueries(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, Analyzer, "a") // loads testdata/src/a/a.go.
}

func Test_contains(t *testing.T) {
	tests := []struct {
		name   string
		needle string
		hay    []string
		want   bool
	}{
		{
			name:   "Contains",
			needle: "a",
			hay:    []string{"a", "b", "c"},
			want:   true,
		},
		{
			name:   "Doesn't contain",
			needle: "d",
			hay:    []string{"a", "b", "c"},
			want:   false,
		},
		{
			name:   "Empty slice",
			needle: "a",
			hay:    []string{},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.needle, tt.hay); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_stripVendor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "Strip",
			input: "github.com/godwhoa/upboat/vendor/github.com/jmoiron/sqlx.DB",
			want:  "github.com/jmoiron/sqlx.DB",
		},
		{
			name:  "Ignore",
			input: "github.com/jmoiron/sqlx.DB",
			want:  "github.com/jmoiron/sqlx.DB",
		},
		{
			name:  "\"vendor\" in pkg url",
			input: "github.com/vendor/upboat/vendor/github.com/jmoiron/sqlx.DB",
			want:  "github.com/jmoiron/sqlx.DB",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripVendor(tt.input); got != tt.want {
				t.Errorf("stripVendor() = %v, want %v", got, tt.want)
			}
		})
	}
}
