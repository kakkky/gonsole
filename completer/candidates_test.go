package completer

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestGenerateCandidates(t *testing.T) {
	tests := []struct {
		name string
		path string
		want *candidates
	}{
		{
			name: "can analyze testdata/simple , and convert from node to candidates",
			path: "./testdata/simple",
			want: &candidates{
				pkgs:    []pkgName{"simple"},
				funcs:   map[pkgName][]funcSet{"simple": {{name: "SimpleFunc", description: "SimpleFunc is a simple function"}}},
				methods: map[pkgName][]methodSet{"simple": {{name: "SimpleMethod", description: "SimpleMethod is a method for SimpleType", receiverTypeName: "SimpleType"}}},
				vars:    map[pkgName][]varSet{"simple": {{name: "SimpleVar", description: "SimpleVar is a variable"}}},
				consts:  map[pkgName][]constSet{"simple": {{name: "SimpleConst", description: "SimpleConst is a constant"}}},
				types:   map[pkgName][]typeSet{"simple": {{name: "SimpleType", description: "SimpleType is a simple type"}}},
			},
		},
		{
			name: "can analyze testdata/complex , and convert from node to candidates",
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
				types: map[pkgName][]typeSet{
					"complex":    {{name: "ComplexType", description: "ComplexType is a complex type"}},
					"subcomplex": {{name: "SubComplexType", description: ""}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateCandidates(tt.path)
			if err != nil {
				t.Errorf("GenerateCandidates() error = %v", err)
				return
			}
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(candidates{}, funcSet{}, methodSet{}, varSet{}, constSet{}, typeSet{})); diff != "" {
				t.Errorf("GenerateCandidates() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
