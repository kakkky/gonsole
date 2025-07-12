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
			name: "testdata/simple",
			path: "./testdata/simple",
			want: &candidates{
				pkgs:    []pkgName{"simple"},
				funcs:   map[pkgName][]funcName{"simple": {"SimpleFunc"}},
				methods: map[pkgName][]receiverMap{"simple": {{typeName("SimpleType"): "SimpleMethod"}}},
				vars:    map[pkgName][]varName{"simple": {"SimpleVar"}},
				consts:  map[pkgName][]constName{"simple": {"SimpleConst"}},
				types:   map[pkgName][]typeName{"simple": {"SimpleType"}},
			},
		},
		{
			name: "testdata/complex",
			path: "./testdata/complex/",
			want: &candidates{
				pkgs:  []pkgName{"complex", "subcomplex"},
				funcs: map[pkgName][]funcName{},
				methods: map[pkgName][]receiverMap{
					"complex":    {{typeName("ComplexType"): "ComplexMethod"}},
					"subcomplex": {{typeName("SubComplexType"): "SubComplexMethod"}},
				},
				vars: map[pkgName][]varName{
					"complex":    {"ComplexA", "ComplexB", "ComplexC", "ComplexD"},
					"subcomplex": {"SubComplexA", "SubComplexB", "SubComplexC", "SubComplexD"},
				},
				consts: map[pkgName][]constName{},
				types: map[pkgName][]typeName{
					"complex":    {"ComplexType"},
					"subcomplex": {"SubComplexType"},
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
			if diff := cmp.Diff(tt.want, got, cmp.AllowUnexported(candidates{})); diff != "" {
				t.Errorf("GenerateCandidates() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
