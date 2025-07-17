package executor

import (
	"testing"

	"github.com/kakkky/gonsole/utils"
)

func TestExtractImportPaths(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{
			name: "Test with valid import paths",
			path: "./testdata/",
			expected: []string{
				`"executor/testdata/pkgb"`,
				`"executor/testdata/pkgb/pkgc"`,
				`"executor/testdata/pkga"`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, err := utils.AnalyzeGoAst(tt.path)
			if err != nil {
				t.Errorf("AnalyzeGoAst() error = %v", err)
				return
			}
			got := ExtractImportPaths(nodes, "../go.mod")
			if len(got) != len(tt.expected) {
				t.Errorf("ExtractImportPaths() got = %v, want %v", got, tt.expected)
				return
			}
			for i, path := range got {
				if path != tt.expected[i] {
					t.Errorf("ExtractImportPaths() got[%d] = %v, want %v", i, path, tt.expected[i])
				}
			}

		})
	}
}
