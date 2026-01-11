package completer

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kakkky/gonsole/types"
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
				pkgs:       []types.PkgName{"simple"},
				funcs:      map[types.PkgName][]funcSet{"simple": {{name: "SimpleFunc", description: "SimpleFunc is a simple function", returnTypeNames: []string{"string"}, returnTypePkgNames: []types.PkgName{"simple"}}}},
				methods:    map[types.PkgName][]methodSet{"simple": {{name: "SimpleMethod", description: "SimpleMethod is a method for SimpleType", receiverName: "SimpleType", returnTypeNames: []string{"string"}, returnTypePkgNames: []types.PkgName{"simple"}}}},
				vars:       map[types.PkgName][]varSet{"simple": {{name: "SimpleVar", description: "SimpleVar is a variable", typeName: "string", typePkgName: ""}}},
				consts:     map[types.PkgName][]constSet{"simple": {{name: "SimpleConst", description: "SimpleConst is a constant"}}},
				structs:    map[types.PkgName][]structSet{"simple": {{name: "SimpleType", fields: []string{"SimpleField"}, description: "SimpleType is a simple type"}}},
				interfaces: map[types.PkgName][]interfaceSet{"simple": {{name: "SimpleInterface", methods: []string{"SimpleMethod"}, descriptions: []string{"SimpleMethod is a method of SimpleInterface"}}}},
			},
		},
		{
			name: "can AnalyzeGoAst testdata/complex , and convert from node to candidates",
			path: "./testdata/complex/",
			want: &candidates{
				// パッケージ名の順序を実際の結果に合わせる
				pkgs:  []types.PkgName{"complex", "subcomplex"},
				funcs: map[types.PkgName][]funcSet{},
				methods: map[types.PkgName][]methodSet{
					"complex": {
						{name: "ComplexMethod", description: "ComplexMethod is a method for ComplexType", receiverName: "ComplexType"},
					},
					"subcomplex": {
						{name: "SubComplexMethod", receiverName: "SubComplexType"},
					},
				},
				vars: map[types.PkgName][]varSet{
					"complex": {
						{name: "ComplexC", description: "Complex variable", typeName: "string", typePkgName: ""},
						{name: "ComplexD", description: "Complex variable", typeName: "string", typePkgName: ""},
					},
					"subcomplex": {
						{name: "SubComplexA", description: "", typeName: "string", typePkgName: ""},
						{name: "SubComplexB", description: "", typeName: "string", typePkgName: ""},
					},
				},
				consts: map[types.PkgName][]constSet{
					"complex":    {{name: "ComplexA", description: "ComplexConst is a constant   ComplexA is a complex constant A"}, {name: "ComplexB", description: "ComplexConst is a constant   ComplexB is a complex constant B"}},
					"subcomplex": {{name: "SubComplexC", description: ""}, {name: "SubComplexD", description: ""}},
				},
				structs: map[types.PkgName][]structSet{
					"complex":    {{name: "ComplexType", description: "ComplexType is a complex type"}},
					"subcomplex": {{name: "SubComplexType", fields: []string{"FieldA", "FieldB"}, description: ""}},
				},
				interfaces: map[types.PkgName][]interfaceSet{}, // 空のマップを期待値に追加
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, _, err := utils.AnalyzeGoAst(tt.path)
			if err != nil {
				t.Fatalf("AnalyzeGoAst() error = %v", err)
			}
			got, err := NewCandidates(nodes) // 解析したパッケージを渡して候補を生成
			if err != nil {
				t.Fatalf("NewCandidates() error = %v", err)
			}
			opts := []cmp.Option{
				cmp.AllowUnexported(candidates{}, funcSet{}, methodSet{}, varSet{}, constSet{}, structSet{}, interfaceSet{}),
				// pkgsの順序を無視
				cmpopts.SortSlices(func(a, b types.PkgName) bool { return a < b }),
			}
			if diff := cmp.Diff(tt.want, got, opts...); diff != "" {
				t.Errorf("GenerateCandidates() mismatch (-want +got):\n%s", diff)
			}
			if diff := cmp.Diff(tt.want, got, opts...); diff != "" {
				t.Errorf("GenerateCandidates() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
