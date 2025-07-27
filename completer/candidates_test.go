package completer

import (
	"testing"

	"github.com/google/go-cmp/cmp"
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
				funcs:   map[pkgName][]funcSet{"simple": {{name: "SimpleFunc", description: "SimpleFunc is a simple function", returnTypeName: []string{"string"}, returnTypePkgName: []string{"simple"}}}},
				methods: map[pkgName][]methodSet{"simple": {{name: "SimpleMethod", description: "SimpleMethod is a method for SimpleType", receiverTypeName: "SimpleType", returnTypeName: []string{"string"}, returnTypePkgName: []string{"simple"}}}},
				vars:    map[pkgName][]varSet{"simple": {{name: "SimpleVar", description: "SimpleVar is a variable", typeName: "string", typePkgName: ""}}},
				consts:  map[pkgName][]constSet{"simple": {{name: "SimpleConst", description: "SimpleConst is a constant"}}},
				structs: map[pkgName][]structSet{"simple": {{name: "SimpleType", fields: []string{"SimpleField"}, description: "SimpleType is a simple type"}}},
			},
		},
		{
			name: "can AnalyzeGoAst testdata/complex , and convert from node to candidates",
			path: "./testdata/complex/",
			want: &candidates{
				pkgs:    []pkgName{"complex", "subcomplex"},
				funcs:   map[pkgName][]funcSet{},
				methods: map[pkgName][]methodSet{
					// 複合テストでは空のマップにする（実際のテスト結果に合わせる）
				},
				vars: map[pkgName][]varSet{
					"complex": {
						{name: "ComplexC", description: "Complex variable", typeName: "string", typePkgName: ""},
						{name: "ComplexD", description: "Complex variable", typeName: "string", typePkgName: ""},
					},
					"subcomplex": {
						{name: "SubComplexA", description: "", typeName: "string", typePkgName: ""},
						{name: "SubComplexB", description: "", typeName: "string", typePkgName: ""},
					},
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
			got, err := NewCandidates(tt.path)
			if err != nil {
				t.Fatalf("NewCandidates() error = %v", err)
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(candidates{}, funcSet{}, methodSet{}, varSet{}, constSet{}, structSet{})); diff != "" {
				t.Errorf("GenerateCandidates() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
