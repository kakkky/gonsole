package completer

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kakkky/gonsole/types"
)

func TestCandidates(t *testing.T) {
	// テスト時は標準ライブラリのマージをスキップ
	SkipStdPkgMergeMode = true
	defer func() { SkipStdPkgMergeMode = false }()

	tests := []struct {
		name string
		path string
		want *candidates
	}{
		{
			name: "functions",
			path: "./testdata/candidates/funcs",
			want: &candidates{
				Pkgs: []types.PkgName{"funcs"},
				Funcs: map[types.PkgName][]funcSet{
					"funcs": {
						{Name: "Add", Description: "Add adds two integers and returns the sum\n", Returns: []returnSet{{TypeName: "int", TypePkgName: ""}}},
						{Name: "ReturnMultiple", Description: "ReturnMultiple returns multiple values\n", Returns: []returnSet{{TypeName: "int", TypePkgName: ""}, {TypeName: "string", TypePkgName: ""}}},
					},
				},
				Methods:    map[types.PkgName][]methodSet{},
				Vars:       map[types.PkgName][]varSet{},
				Consts:     map[types.PkgName][]constSet{},
				Structs:    map[types.PkgName][]structSet{},
				Interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "methods",
			path: "./testdata/candidates/methods",
			want: &candidates{
				Pkgs:  []types.PkgName{"methods"},
				Funcs: map[types.PkgName][]funcSet{},
				Methods: map[types.PkgName][]methodSet{
					"methods": {
						{Name: "Increment", Description: "Increment increments the counter value\n", ReceiverTypeName: "Counter", Returns: []returnSet{{TypeName: "int", TypePkgName: ""}}},
						{Name: "GetValue", Description: "GetValue returns the current value\n", ReceiverTypeName: "Counter", Returns: []returnSet{{TypeName: "int", TypePkgName: ""}}},
					},
				},
				Vars:   map[types.PkgName][]varSet{},
				Consts: map[types.PkgName][]constSet{},
				Structs: map[types.PkgName][]structSet{
					"methods": {{Name: "Counter", Fields: []types.StructFieldName{"Value"}, Description: ""}},
				},
				Interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "vars_basiclit",
			path: "./testdata/candidates/vars_basiclit",
			want: &candidates{
				Pkgs:    []types.PkgName{"varsbasiclit"},
				Funcs:   map[types.PkgName][]funcSet{},
				Methods: map[types.PkgName][]methodSet{},
				Vars: map[types.PkgName][]varSet{
					"varsbasiclit": {
						{Name: "IntVar", Description: "IntVar is an integer variable\n", TypeName: "int", TypePkgName: ""},
						{Name: "StringVar", Description: "StringVar is a string variable\n", TypeName: "string", TypePkgName: ""},
						{Name: "FloatVar", Description: "FloatVar is a float variable\n", TypeName: "float64", TypePkgName: ""},
					},
				},
				Consts:     map[types.PkgName][]constSet{},
				Structs:    map[types.PkgName][]structSet{},
				Interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "vars_compositelit",
			path: "./testdata/candidates/vars_compositelit",
			want: &candidates{
				Pkgs:    []types.PkgName{"varscompositelit"},
				Funcs:   map[types.PkgName][]funcSet{},
				Methods: map[types.PkgName][]methodSet{},
				Vars: map[types.PkgName][]varSet{
					"varscompositelit": {
						{Name: "SimplePerson", Description: "SimplePerson is initialized with a composite literal\n", TypeName: "Person", TypePkgName: "varscompositelit"},
						{Name: "PersonPtr", Description: "PersonPtr is initialized with a pointer to composite literal\n", TypeName: "*Person", TypePkgName: "varscompositelit"},
					},
				},
				Consts: map[types.PkgName][]constSet{},
				Structs: map[types.PkgName][]structSet{
					"varscompositelit": {{Name: "Person", Fields: []types.StructFieldName{"Name", "Age"}, Description: ""}},
				},
				Interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "vars_funccall",
			path: "./testdata/candidates/vars_funccall",
			want: &candidates{
				Pkgs: []types.PkgName{"varsfunccall"},
				Funcs: map[types.PkgName][]funcSet{
					"varsfunccall": {{Name: "NewConfig", Description: "NewConfig creates a new Config instance\n", Returns: []returnSet{{TypeName: "Config", TypePkgName: "varsfunccall"}}}},
				},
				Methods: map[types.PkgName][]methodSet{},
				Vars: map[types.PkgName][]varSet{
					"varsfunccall": {{Name: "ConfigVar", Description: "ConfigVar is initialized by a function call\n", TypeName: "Config", TypePkgName: "varsfunccall"}},
				},
				Consts: map[types.PkgName][]constSet{},
				Structs: map[types.PkgName][]structSet{
					"varsfunccall": {{Name: "Config", Fields: []types.StructFieldName{"Name"}, Description: ""}},
				},
				Interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "vars_methodcall",
			path: "./testdata/candidates/vars_methodcall",
			want: &candidates{
				Pkgs: []types.PkgName{"varsmethodcall"},
				Funcs: map[types.PkgName][]funcSet{
					"varsmethodcall": {{Name: "NewLogger", Description: "NewLogger creates a new Logger instance\n", Returns: []returnSet{{TypeName: "Logger", TypePkgName: "varsmethodcall"}}}},
				},
				Methods: map[types.PkgName][]methodSet{
					"varsmethodcall": {
						{Name: "Info", Description: "Info logs an info message and returns the logged message\n", ReceiverTypeName: "Logger", Returns: []returnSet{{TypeName: "string", TypePkgName: ""}}},
					},
				},
				Vars: map[types.PkgName][]varSet{
					"varsmethodcall": {
						{Name: "GlobalLogger", Description: "GlobalLogger is a top-level variable\n", TypeName: "Logger", TypePkgName: "varsmethodcall"},
						{Name: "ResultFromMethod", Description: "ResultFromMethod is initialized by a method call on a top-level variable\n", TypeName: "string", TypePkgName: ""},
					},
				},
				Consts: map[types.PkgName][]constSet{},
				Structs: map[types.PkgName][]structSet{
					"varsmethodcall": {{Name: "Logger", Fields: []types.StructFieldName{"Name"}, Description: ""}},
				},
				Interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "consts",
			path: "./testdata/candidates/consts",
			want: &candidates{
				Pkgs:    []types.PkgName{"consts"},
				Funcs:   map[types.PkgName][]funcSet{},
				Methods: map[types.PkgName][]methodSet{},
				Vars:    map[types.PkgName][]varSet{},
				Consts: map[types.PkgName][]constSet{
					"consts": {
						{Name: "MaxSize", Description: "MaxSize is the maximum size\n"},
						{Name: "MinValue", Description: "MinValue is the minimum value\n"},
						{Name: "MaxValue", Description: "MaxValue is the maximum value\n"},
						{Name: "DefaultWidth", Description: "Multiple names in one spec\n"},
						{Name: "DefaultHeight", Description: "Multiple names in one spec\n"},
					},
				},
				Structs:    map[types.PkgName][]structSet{},
				Interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "structs",
			path: "./testdata/candidates/structs",
			want: &candidates{
				Pkgs:    []types.PkgName{"structs"},
				Funcs:   map[types.PkgName][]funcSet{},
				Methods: map[types.PkgName][]methodSet{},
				Vars:    map[types.PkgName][]varSet{},
				Consts:  map[types.PkgName][]constSet{},
				Structs: map[types.PkgName][]structSet{
					"structs": {
						{Name: "SimpleStruct", Fields: []types.StructFieldName{"FieldA", "FieldB"}, Description: ""},
						{Name: "MultiFieldStruct", Fields: []types.StructFieldName{"X", "Y", "Z"}, Description: ""},
						{Name: "Base", Fields: []types.StructFieldName{"ID"}, Description: ""},
						{Name: "Derived", Fields: []types.StructFieldName{"Base", "Name"}, Description: ""},
					},
				},
				Interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "interfaces",
			path: "./testdata/candidates/interfaces",
			want: &candidates{
				Pkgs:    []types.PkgName{"interfaces"},
				Funcs:   map[types.PkgName][]funcSet{},
				Methods: map[types.PkgName][]methodSet{},
				Vars:    map[types.PkgName][]varSet{},
				Consts:  map[types.PkgName][]constSet{},
				Structs: map[types.PkgName][]structSet{},
				Interfaces: map[types.PkgName][]interfaceSet{
					"interfaces": {
						{Name: "Reader", Methods: []types.DeclName{"Read"}, Descriptions: nil},
						{Name: "Writer", Methods: []types.DeclName{"Write"}, Descriptions: nil},
					},
				},
			},
		},
		{
			name: "multipackage",
			path: "./testdata/candidates/multipackage",
			want: &candidates{
				Pkgs: []types.PkgName{"main", "types"},
				Funcs: map[types.PkgName][]funcSet{
					"main": {
						{Name: "GetConfig", Description: "GetConfig returns a Config from another package\n", Returns: []returnSet{{TypeName: "Config", TypePkgName: "types"}}},
						{Name: "GetLogger", Description: "GetLogger returns a Logger from another package\n", Returns: []returnSet{{TypeName: "Logger", TypePkgName: "types"}}},
					},
				},
				Methods: map[types.PkgName][]methodSet{
					"main": {
						{Name: "GetConfigFromMethod", Description: "GetConfigFromMethod returns a Config from another package via method\n", ReceiverTypeName: "Service", Returns: []returnSet{{TypeName: "Config", TypePkgName: "types"}}},
					},
					"types": {
						{Name: "Info", Description: "Info logs an info message\n", ReceiverTypeName: "Logger", Returns: []returnSet{{TypeName: "string", TypePkgName: ""}}},
					},
				},
				Vars:   map[types.PkgName][]varSet{},
				Consts: map[types.PkgName][]constSet{},
				Structs: map[types.PkgName][]structSet{
					"main": {{Name: "Service", Fields: []types.StructFieldName{"Name"}, Description: ""}},
					"types": {
						{Name: "Config", Fields: []types.StructFieldName{"Name", "Value"}, Description: ""},
						{Name: "Logger", Fields: []types.StructFieldName{"Level"}, Description: ""},
					},
				},
				Interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "multidecl",
			path: "./testdata/candidates/multidecl",
			want: &candidates{
				Pkgs:    []types.PkgName{"multidecl"},
				Funcs:   map[types.PkgName][]funcSet{},
				Methods: map[types.PkgName][]methodSet{},
				Vars: map[types.PkgName][]varSet{
					"multidecl": {
						{Name: "VarA", Description: "Multiple variables in one line\n", TypeName: "string", TypePkgName: ""},
						{Name: "VarB", Description: "Multiple variables in one line\n", TypeName: "string", TypePkgName: ""},
						{Name: "VarC", Description: "Multiple variables in one var block\n", TypeName: "string", TypePkgName: ""},
						{Name: "VarD", Description: "Multiple variables in one var block\n", TypeName: "string", TypePkgName: ""},
					},
				},
				Consts: map[types.PkgName][]constSet{
					"multidecl": {
						{Name: "ConstA", Description: "Multiple constants in one line\n"},
						{Name: "ConstB", Description: "Multiple constants in one line\n"},
						{Name: "ConstC", Description: "Multiple constants in one const block\n"},
						{Name: "ConstD", Description: "Multiple constants in one const block\n"},
					},
				},
				Structs:    map[types.PkgName][]structSet{},
				Interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewCandidates(tt.path)
			if err != nil {
				t.Fatalf("NewCandidates() error = %v", err)
			}

			opts := []cmp.Option{
				cmp.AllowUnexported(candidates{}, funcSet{}, methodSet{}, varSet{}, constSet{}, structSet{}, interfaceSet{}, returnSet{}),
				cmpopts.SortSlices(func(a, b types.PkgName) bool { return a < b }),
				cmpopts.SortSlices(func(a, b funcSet) bool { return a.Name < b.Name }),
				cmpopts.SortSlices(func(a, b methodSet) bool { return a.Name < b.Name }),
				cmpopts.SortSlices(func(a, b varSet) bool { return a.Name < b.Name }),
				cmpopts.SortSlices(func(a, b constSet) bool { return a.Name < b.Name }),
				cmpopts.SortSlices(func(a, b structSet) bool { return a.Name < b.Name }),
				cmpopts.SortSlices(func(a, b interfaceSet) bool { return a.Name < b.Name }),
			}
			if diff := cmp.Diff(tt.want, got, opts...); diff != "" {
				t.Errorf("%s mismatch (-want +got):\n%s", tt.name, diff)
			}
		})
	}
}
