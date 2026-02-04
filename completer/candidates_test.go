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
						{Name: "Add", Description: "Add adds two integers and returns the sum", Returns: []returnSet{{TypeName: "int", PkgName: "funcs"}}},
						{Name: "ReturnMultiple", Description: "ReturnMultiple returns multiple values", Returns: []returnSet{{TypeName: "int", PkgName: "funcs"}, {TypeName: "string", PkgName: "funcs"}}},
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
						{Name: "Increment", Description: "Increment increments the counter value", ReceiverTypeName: "Counter", Returns: []returnSet{{TypeName: "int", PkgName: "methods"}}},
						{Name: "GetValue", Description: "GetValue returns the current value", ReceiverTypeName: "Counter", Returns: []returnSet{{TypeName: "int", PkgName: "methods"}}},
					},
				},
				Vars:   map[types.PkgName][]varSet{},
				Consts: map[types.PkgName][]constSet{},
				Structs: map[types.PkgName][]structSet{
					"methods": {{Name: "Counter", Fields: []types.StructFieldName{"Value"}, Description: "Counter is a simple counter type"}},
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
						{Name: "IntVar", Description: "Basic literal variable declarations   IntVar is an integer variable", TypeName: "int", PkgName: ""},
						{Name: "StringVar", Description: "Basic literal variable declarations   StringVar is a string variable", TypeName: "string", PkgName: ""},
						{Name: "FloatVar", Description: "Basic literal variable declarations   FloatVar is a float variable", TypeName: "float64", PkgName: ""},
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
						{Name: "SimplePerson", Description: "SimplePerson is initialized with a composite literal", TypeName: "Person", PkgName: "varscompositelit"},
						{Name: "PersonPtr", Description: "PersonPtr is initialized with a pointer to composite literal", TypeName: "Person", PkgName: "varscompositelit"},
					},
				},
				Consts: map[types.PkgName][]constSet{},
				Structs: map[types.PkgName][]structSet{
					"varscompositelit": {{Name: "Person", Fields: []types.StructFieldName{"Name", "Age"}, Description: "Person is a simple struct"}},
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
					"varsfunccall": {{Name: "NewConfig", Description: "NewConfig creates a new Config instance", Returns: []returnSet{{TypeName: "Config", PkgName: "varsfunccall"}}}},
				},
				Methods: map[types.PkgName][]methodSet{},
				Vars: map[types.PkgName][]varSet{
					"varsfunccall": {{Name: "ConfigVar", Description: "ConfigVar is initialized by a function call", TypeName: "Config", PkgName: "varsfunccall"}},
				},
				Consts: map[types.PkgName][]constSet{},
				Structs: map[types.PkgName][]structSet{
					"varsfunccall": {{Name: "Config", Fields: []types.StructFieldName{"Name"}, Description: "Config is a configuration struct"}},
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
					"varsmethodcall": {{Name: "NewLogger", Description: "NewLogger creates a new Logger instance", Returns: []returnSet{{TypeName: "Logger", PkgName: "varsmethodcall"}}}},
				},
				Methods: map[types.PkgName][]methodSet{
					"varsmethodcall": {{Name: "Info", Description: "Info logs an info message and returns the logged message", ReceiverTypeName: "Logger", Returns: []returnSet{{TypeName: "string", PkgName: "varsmethodcall"}}}},
				},
				Vars: map[types.PkgName][]varSet{
					"varsmethodcall": {
						{Name: "GlobalLogger", Description: "GlobalLogger is a top-level variable", TypeName: "Logger", PkgName: "varsmethodcall"},
						{Name: "ResultFromMethod", Description: "ResultFromMethod is initialized by a method call on a top-level variable", TypeName: "string", PkgName: "varsmethodcall"},
					},
				},
				Consts: map[types.PkgName][]constSet{},
				Structs: map[types.PkgName][]structSet{
					"varsmethodcall": {{Name: "Logger", Fields: []types.StructFieldName{"Name"}, Description: "Logger is a simple logger type"}},
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
						{Name: "MaxSize", Description: "MaxSize is the maximum size"},
						{Name: "MinValue", Description: "Multiple constants in one declaration   MinValue is the minimum value"},
						{Name: "MaxValue", Description: "Multiple constants in one declaration   MaxValue is the maximum value"},
						{Name: "DefaultWidth", Description: "Multiple names in one spec"},
						{Name: "DefaultHeight", Description: "Multiple names in one spec"},
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
						{Name: "SimpleStruct", Fields: []types.StructFieldName{"FieldA", "FieldB"}, Description: "SimpleStruct has simple fields"},
						{Name: "MultiFieldStruct", Fields: []types.StructFieldName{"X", "Y", "Z"}, Description: "MultiFieldStruct has multiple fields on same line"},
						{Name: "Base", Fields: []types.StructFieldName{"ID"}, Description: "Base is used for embedding"},
						{Name: "Derived", Fields: []types.StructFieldName{"Base", "Name"}, Description: "Derived embeds Base"},
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
						{Name: "Reader", Methods: []types.DeclName{"Read"}, Descriptions: []string{"Read reads data"}},
						{Name: "Writer", Methods: []types.DeclName{"Write"}, Descriptions: []string{"Write writes data"}},
					},
				},
			},
		},
		{
			name: "multipackage",
			path: "./testdata/candidates/multipackage",
			want: &candidates{
				Pkgs:  []types.PkgName{"types"},
				Funcs: map[types.PkgName][]funcSet{},
				Methods: map[types.PkgName][]methodSet{
					"types": {
						{Name: "Info", Description: "Info logs an info message", ReceiverTypeName: "Logger", Returns: []returnSet{{TypeName: "string", PkgName: "types"}}},
					},
				},
				Vars:   map[types.PkgName][]varSet{},
				Consts: map[types.PkgName][]constSet{},
				Structs: map[types.PkgName][]structSet{
					"types": {
						{Name: "Config", Fields: []types.StructFieldName{"Name", "Value"}, Description: "Config is a configuration struct from another package"},
						{Name: "Logger", Fields: []types.StructFieldName{"Level"}, Description: "Logger is a logger struct from another package"},
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
						{Name: "VarA", Description: "Multiple variables in one line", TypeName: "string", PkgName: ""},
						{Name: "VarB", Description: "Multiple variables in one line", TypeName: "string", PkgName: ""},
						{Name: "VarC", Description: "Multiple variables in one var block", TypeName: "string", PkgName: ""},
						{Name: "VarD", Description: "Multiple variables in one var block", TypeName: "string", PkgName: ""},
					},
				},
				Consts: map[types.PkgName][]constSet{
					"multidecl": {
						{Name: "ConstA", Description: "Multiple constants in one line"},
						{Name: "ConstB", Description: "Multiple constants in one line"},
						{Name: "ConstC", Description: "Multiple constants in one const block"},
						{Name: "ConstD", Description: "Multiple constants in one const block"},
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
