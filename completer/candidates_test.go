package completer

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kakkky/gonsole/types"
	"github.com/kakkky/gonsole/utils"
)

func TestCandidates(t *testing.T) {
	tests := []struct {
		name string
		path string
		want *candidates
	}{
		{
			name: "functions",
			path: "./testdata/candidates/funcs",
			want: &candidates{
				pkgs: []types.PkgName{"funcs"},
				funcs: map[types.PkgName][]funcSet{
					"funcs": {
						{name: "Add", description: "Add adds two integers and returns the sum", returns: []returnSet{{typeName: "int", pkgName: "funcs"}}},
						{name: "ReturnMultiple", description: "ReturnMultiple returns multiple values", returns: []returnSet{{typeName: "int", pkgName: "funcs"}, {typeName: "string", pkgName: "funcs"}}},
					},
				},
				methods:    map[types.PkgName][]methodSet{},
				vars:       map[types.PkgName][]varSet{},
				consts:     map[types.PkgName][]constSet{},
				structs:    map[types.PkgName][]structSet{},
				interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "methods",
			path: "./testdata/candidates/methods",
			want: &candidates{
				pkgs:  []types.PkgName{"methods"},
				funcs: map[types.PkgName][]funcSet{},
				methods: map[types.PkgName][]methodSet{
					"methods": {
						{name: "Increment", description: "Increment increments the counter value", receiverTypeName: "Counter", returns: []returnSet{{typeName: "int", pkgName: "methods"}}},
						{name: "GetValue", description: "GetValue returns the current value", receiverTypeName: "Counter", returns: []returnSet{{typeName: "int", pkgName: "methods"}}},
					},
				},
				vars:   map[types.PkgName][]varSet{},
				consts: map[types.PkgName][]constSet{},
				structs: map[types.PkgName][]structSet{
					"methods": {{name: "Counter", fields: []types.StructFieldName{"Value"}, description: "Counter is a simple counter type"}},
				},
				interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "vars_basiclit",
			path: "./testdata/candidates/vars_basiclit",
			want: &candidates{
				pkgs:    []types.PkgName{"varsbasiclit"},
				funcs:   map[types.PkgName][]funcSet{},
				methods: map[types.PkgName][]methodSet{},
				vars: map[types.PkgName][]varSet{
					"varsbasiclit": {
						{name: "IntVar", description: "Basic literal variable declarations   IntVar is an integer variable", typeName: "int", pkgName: ""},
						{name: "StringVar", description: "Basic literal variable declarations   StringVar is a string variable", typeName: "string", pkgName: ""},
						{name: "FloatVar", description: "Basic literal variable declarations   FloatVar is a float variable", typeName: "float64", pkgName: ""},
					},
				},
				consts:     map[types.PkgName][]constSet{},
				structs:    map[types.PkgName][]structSet{},
				interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "vars_compositelit",
			path: "./testdata/candidates/vars_compositelit",
			want: &candidates{
				pkgs:    []types.PkgName{"varscompositelit"},
				funcs:   map[types.PkgName][]funcSet{},
				methods: map[types.PkgName][]methodSet{},
				vars: map[types.PkgName][]varSet{
					"varscompositelit": {
						{name: "SimplePerson", description: "SimplePerson is initialized with a composite literal", typeName: "Person", pkgName: "varscompositelit"},
						{name: "PersonPtr", description: "PersonPtr is initialized with a pointer to composite literal", typeName: "Person", pkgName: "varscompositelit"},
					},
				},
				consts: map[types.PkgName][]constSet{},
				structs: map[types.PkgName][]structSet{
					"varscompositelit": {{name: "Person", fields: []types.StructFieldName{"Name", "Age"}, description: "Person is a simple struct"}},
				},
				interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "vars_funccall",
			path: "./testdata/candidates/vars_funccall",
			want: &candidates{
				pkgs: []types.PkgName{"varsfunccall"},
				funcs: map[types.PkgName][]funcSet{
					"varsfunccall": {{name: "NewConfig", description: "NewConfig creates a new Config instance", returns: []returnSet{{typeName: "Config", pkgName: "varsfunccall"}}}},
				},
				methods: map[types.PkgName][]methodSet{},
				vars: map[types.PkgName][]varSet{
					"varsfunccall": {{name: "ConfigVar", description: "ConfigVar is initialized by a function call", typeName: "Config", pkgName: "varsfunccall"}},
				},
				consts: map[types.PkgName][]constSet{},
				structs: map[types.PkgName][]structSet{
					"varsfunccall": {{name: "Config", fields: []types.StructFieldName{"Name"}, description: "Config is a configuration struct"}},
				},
				interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "vars_methodcall",
			path: "./testdata/candidates/vars_methodcall",
			want: &candidates{
				pkgs: []types.PkgName{"varsmethodcall"},
				funcs: map[types.PkgName][]funcSet{
					"varsmethodcall": {{name: "NewLogger", description: "NewLogger creates a new Logger instance", returns: []returnSet{{typeName: "Logger", pkgName: "varsmethodcall"}}}},
				},
				methods: map[types.PkgName][]methodSet{
					"varsmethodcall": {{name: "Info", description: "Info logs an info message and returns the logged message", receiverTypeName: "Logger", returns: []returnSet{{typeName: "string", pkgName: "varsmethodcall"}}}},
				},
				vars: map[types.PkgName][]varSet{
					"varsmethodcall": {
						{name: "GlobalLogger", description: "GlobalLogger is a top-level variable", typeName: "Logger", pkgName: "varsmethodcall"},
						{name: "ResultFromMethod", description: "ResultFromMethod is initialized by a method call on a top-level variable", typeName: "string", pkgName: "varsmethodcall"},
					},
				},
				consts: map[types.PkgName][]constSet{},
				structs: map[types.PkgName][]structSet{
					"varsmethodcall": {{name: "Logger", fields: []types.StructFieldName{"Name"}, description: "Logger is a simple logger type"}},
				},
				interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "consts",
			path: "./testdata/candidates/consts",
			want: &candidates{
				pkgs:    []types.PkgName{"consts"},
				funcs:   map[types.PkgName][]funcSet{},
				methods: map[types.PkgName][]methodSet{},
				vars:    map[types.PkgName][]varSet{},
				consts: map[types.PkgName][]constSet{
					"consts": {
						{name: "MaxSize", description: "MaxSize is the maximum size"},
						{name: "MinValue", description: "Multiple constants in one declaration   MinValue is the minimum value"},
						{name: "MaxValue", description: "Multiple constants in one declaration   MaxValue is the maximum value"},
						{name: "DefaultWidth", description: "Multiple names in one spec"},
						{name: "DefaultHeight", description: "Multiple names in one spec"},
					},
				},
				structs:    map[types.PkgName][]structSet{},
				interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "structs",
			path: "./testdata/candidates/structs",
			want: &candidates{
				pkgs:    []types.PkgName{"structs"},
				funcs:   map[types.PkgName][]funcSet{},
				methods: map[types.PkgName][]methodSet{},
				vars:    map[types.PkgName][]varSet{},
				consts:  map[types.PkgName][]constSet{},
				structs: map[types.PkgName][]structSet{
					"structs": {
						{name: "SimpleStruct", fields: []types.StructFieldName{"FieldA", "FieldB"}, description: "SimpleStruct has simple fields"},
						{name: "MultiFieldStruct", fields: []types.StructFieldName{"X", "Y", "Z"}, description: "MultiFieldStruct has multiple fields on same line"},
						{name: "Base", fields: []types.StructFieldName{"ID"}, description: "Base is used for embedding"},
						{name: "Derived", fields: []types.StructFieldName{"Base", "Name"}, description: "Derived embeds Base"},
					},
				},
				interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "interfaces",
			path: "./testdata/candidates/interfaces",
			want: &candidates{
				pkgs:    []types.PkgName{"interfaces"},
				funcs:   map[types.PkgName][]funcSet{},
				methods: map[types.PkgName][]methodSet{},
				vars:    map[types.PkgName][]varSet{},
				consts:  map[types.PkgName][]constSet{},
				structs: map[types.PkgName][]structSet{},
				interfaces: map[types.PkgName][]interfaceSet{
					"interfaces": {
						{name: "Reader", methods: []types.DeclName{"Read"}, descriptions: []string{"Read reads data"}},
						{name: "Writer", methods: []types.DeclName{"Write"}, descriptions: []string{"Write writes data"}},
					},
				},
			},
		},
		{
			name: "multipackage",
			path: "./testdata/candidates/multipackage",
			want: &candidates{
				pkgs: []types.PkgName{"main", "types"},
				funcs: map[types.PkgName][]funcSet{
					"main": {
						{name: "GetConfig", description: "GetConfig returns a Config from another package", returns: []returnSet{{typeName: "Config", pkgName: "types"}}},
						{name: "GetLogger", description: "GetLogger returns a Logger from another package", returns: []returnSet{{typeName: "Logger", pkgName: "types"}}},
					},
				},
				methods: map[types.PkgName][]methodSet{
					"main": {
						{name: "GetConfigFromMethod", description: "GetConfigFromMethod returns a Config from another package via method", receiverTypeName: "Service", returns: []returnSet{{typeName: "Config", pkgName: "types"}}},
					},
					"types": {
						{name: "Info", description: "Info logs an info message", receiverTypeName: "Logger", returns: []returnSet{{typeName: "string", pkgName: "types"}}},
					},
				},
				vars:   map[types.PkgName][]varSet{},
				consts: map[types.PkgName][]constSet{},
				structs: map[types.PkgName][]structSet{
					"main": {
						{name: "Service", fields: []types.StructFieldName{"Name"}, description: "Service has a method that returns a type from another package"},
					},
					"types": {
						{name: "Config", fields: []types.StructFieldName{"Name", "Value"}, description: "Config is a configuration struct from another package"},
						{name: "Logger", fields: []types.StructFieldName{"Level"}, description: "Logger is a logger struct from another package"},
					},
				},
				interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
		{
			name: "multidecl",
			path: "./testdata/candidates/multidecl",
			want: &candidates{
				pkgs:    []types.PkgName{"multidecl"},
				funcs:   map[types.PkgName][]funcSet{},
				methods: map[types.PkgName][]methodSet{},
				vars: map[types.PkgName][]varSet{
					"multidecl": {
						{name: "VarA", description: "Multiple variables in one line", typeName: "string", pkgName: ""},
						{name: "VarB", description: "Multiple variables in one line", typeName: "string", pkgName: ""},
						{name: "VarC", description: "Multiple variables in one var block", typeName: "string", pkgName: ""},
						{name: "VarD", description: "Multiple variables in one var block", typeName: "string", pkgName: ""},
					},
				},
				consts: map[types.PkgName][]constSet{
					"multidecl": {
						{name: "ConstA", description: "Multiple constants in one line"},
						{name: "ConstB", description: "Multiple constants in one line"},
						{name: "ConstC", description: "Multiple constants in one const block"},
						{name: "ConstD", description: "Multiple constants in one const block"},
					},
				},
				structs:    map[types.PkgName][]structSet{},
				interfaces: map[types.PkgName][]interfaceSet{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes, _, err := utils.AnalyzeGoAst(tt.path)
			if err != nil {
				t.Fatalf("AnalyzeGoAst() error = %v", err)
			}
			got, err := NewCandidates(nodes)
			if err != nil {
				t.Fatalf("NewCandidates() error = %v", err)
			}

			opts := []cmp.Option{
				cmp.AllowUnexported(candidates{}, funcSet{}, methodSet{}, varSet{}, constSet{}, structSet{}, interfaceSet{}, returnSet{}),
				cmpopts.SortSlices(func(a, b types.PkgName) bool { return a < b }),
				cmpopts.SortSlices(func(a, b funcSet) bool { return a.name < b.name }),
				cmpopts.SortSlices(func(a, b methodSet) bool { return a.name < b.name }),
				cmpopts.SortSlices(func(a, b varSet) bool { return a.name < b.name }),
				cmpopts.SortSlices(func(a, b constSet) bool { return a.name < b.name }),
				cmpopts.SortSlices(func(a, b structSet) bool { return a.name < b.name }),
				cmpopts.SortSlices(func(a, b interfaceSet) bool { return a.name < b.name }),
			}
			if diff := cmp.Diff(tt.want, got, opts...); diff != "" {
				t.Errorf("%s mismatch (-want +got):\n%s", tt.name, diff)
			}
		})
	}
}
