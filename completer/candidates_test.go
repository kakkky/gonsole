package completer

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kakkky/gonsole/utils"
)

func TestConvertFromNodeToCandidates(t *testing.T) {
	tests := []struct {
		name string
		path string
		want *candidates
	}{
		{
			name: "can AnalyzeGoAst testdata/simple , and convert from node to candidates",
			path: "./testdata/simple",
			want: &candidates{
				pkgs:    []pkgName{"simple"},
				funcs:   map[pkgName][]funcSet{"simple": {{name: "SimpleFunc", description: "SimpleFunc is a simple function"}}},
				methods: map[pkgName][]methodSet{"simple": {{name: "SimpleMethod", description: "SimpleMethod is a method for SimpleType", receiverTypeName: "SimpleType"}}},
				vars:    map[pkgName][]varSet{"simple": {{name: "SimpleVar", description: "SimpleVar is a variable"}}},
				consts:  map[pkgName][]constSet{"simple": {{name: "SimpleConst", description: "SimpleConst is a constant"}}},
				structs: map[pkgName][]structSet{"simple": {{name: "SimpleType", fields: []string{"SimpleField"}, description: "SimpleType is a simple type"}}},
			},
		},
		{
			name: "can AnalyzeGoAst testdata/complex , and convert from node to candidates",
			path: "./testdata/complex/",
			want: &candidates{
				pkgs:  []pkgName{"complex", "subcomplex"},
				funcs: map[pkgName][]funcSet{},
				methods: map[pkgName][]methodSet{
					"complex":    {{name: "ComplexMethod", description: "ComplexMethod is a method for ComplexType", receiverTypeName: "ComplexType"}},
					"subcomplex": {{name: "SubComplexMethod", description: "", receiverTypeName: "SubComplexType"}},
				},
				vars: map[pkgName][]varSet{
					"complex":    {{name: "ComplexC", description: "Complex variable"}, {name: "ComplexD", description: "Complex variable"}},
					"subcomplex": {{name: "SubComplexA", description: ""}, {name: "SubComplexB", description: ""}},
				},
				consts: map[pkgName][]constSet{
					"complex":    {{name: "ComplexA", description: "ComplexConst is a constant   ComplexA is a complex constant A"}, {name: "ComplexB", description: "ComplexConst is a constant   ComplexB is a complex constant B"}},
					"subcomplex": {{name: "SubComplexC", description: ""}, {name: "SubComplexD", description: ""}},
				},
				structs: map[pkgName][]structSet{
					"complex":    {{name: "ComplexType", description: "ComplexType is a complex type"}},
					"subcomplex": {{name: "SubComplexType", fields: []string{"FieldA", "FieldB"}, description: ""}},
				},
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
			got := ConvertFromNodeToCandidates(nodes)
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(candidates{}, funcSet{}, methodSet{}, varSet{}, constSet{}, structSet{})); diff != "" {
				t.Errorf("GenerateCandidates() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
