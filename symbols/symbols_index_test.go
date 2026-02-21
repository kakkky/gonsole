package symbols

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
		want *SymbolIndex
	}{
		{
			name: "functions",
			path: "./testdata/funcs",
			want: &SymbolIndex{
				Pkgs: []types.PkgName{"funcs"},
				Funcs: map[types.PkgName][]FuncSet{
					"funcs": {
						{Name: "Add", Description: "Add adds two integers and returns the sum\n", Returns: []ReturnSet{{TypeName: "int", TypePkgName: ""}}},
						{Name: "ReturnMultiple", Description: "ReturnMultiple returns multiple values\n", Returns: []ReturnSet{{TypeName: "int", TypePkgName: ""}, {TypeName: "string", TypePkgName: ""}}},
					},
				},
				Methods:      map[types.PkgName][]MethodSet{},
				Vars:         map[types.PkgName][]VarSet{},
				Consts:       map[types.PkgName][]ConstSet{},
				Structs:      map[types.PkgName][]StructSet{},
				Interfaces:   map[types.PkgName][]InterfaceSet{},
				DefinedTypes: map[types.PkgName][]DefinedTypeSet{},
			},
		},
		{
			name: "methods",
			path: "./testdata/methods",
			want: &SymbolIndex{
				Pkgs:  []types.PkgName{"methods"},
				Funcs: map[types.PkgName][]FuncSet{},
				Methods: map[types.PkgName][]MethodSet{
					"methods": {
						{Name: "Increment", Description: "Increment increments the counter value\n", ReceiverTypeName: "Counter", Returns: []ReturnSet{{TypeName: "int", TypePkgName: ""}}},
						{Name: "GetValue", Description: "GetValue returns the current value\n", ReceiverTypeName: "Counter", Returns: []ReturnSet{{TypeName: "int", TypePkgName: ""}}},
					},
				},
				Vars:   map[types.PkgName][]VarSet{},
				Consts: map[types.PkgName][]ConstSet{},
				Structs: map[types.PkgName][]StructSet{
					"methods": {{Name: "Counter", Fields: []types.StructFieldName{"Value"}, Description: "Counter is a simple counter type\n"}},
				},
				Interfaces:   map[types.PkgName][]InterfaceSet{},
				DefinedTypes: map[types.PkgName][]DefinedTypeSet{},
			},
		},
		{
			name: "vars_basiclit",
			path: "./testdata/vars_basiclit",
			want: &SymbolIndex{
				Pkgs:    []types.PkgName{"varsbasiclit"},
				Funcs:   map[types.PkgName][]FuncSet{},
				Methods: map[types.PkgName][]MethodSet{},
				Vars: map[types.PkgName][]VarSet{
					"varsbasiclit": {
						{Name: "IntVar", Description: "IntVar is an integer variable\n", TypeName: "int", TypePkgName: ""},
						{Name: "StringVar", Description: "StringVar is a string variable\n", TypeName: "string", TypePkgName: ""},
						{Name: "FloatVar", Description: "FloatVar is a float variable\n", TypeName: "float64", TypePkgName: ""},
					},
				},
				Consts:       map[types.PkgName][]ConstSet{},
				Structs:      map[types.PkgName][]StructSet{},
				Interfaces:   map[types.PkgName][]InterfaceSet{},
				DefinedTypes: map[types.PkgName][]DefinedTypeSet{},
			},
		},
		{
			name: "vars_compositelit",
			path: "./testdata/vars_compositelit",
			want: &SymbolIndex{
				Pkgs:    []types.PkgName{"varscompositelit"},
				Funcs:   map[types.PkgName][]FuncSet{},
				Methods: map[types.PkgName][]MethodSet{},
				Vars: map[types.PkgName][]VarSet{
					"varscompositelit": {
						{Name: "SimplePerson", Description: "SimplePerson is initialized with a composite literal\n", TypeName: "Person", TypePkgName: "varscompositelit"},
						{Name: "PersonPtr", Description: "PersonPtr is initialized with a pointer to composite literal\n", TypeName: "Person", TypePkgName: "varscompositelit"},
					},
				},
				Consts: map[types.PkgName][]ConstSet{},
				Structs: map[types.PkgName][]StructSet{
					"varscompositelit": {{Name: "Person", Fields: []types.StructFieldName{"Name", "Age"}, Description: "Person is a simple struct\n"}},
				},
				Interfaces:   map[types.PkgName][]InterfaceSet{},
				DefinedTypes: map[types.PkgName][]DefinedTypeSet{},
			},
		},
		{
			name: "vars_funccall",
			path: "./testdata/vars_funccall",
			want: &SymbolIndex{
				Pkgs: []types.PkgName{"varsfunccall"},
				Funcs: map[types.PkgName][]FuncSet{
					"varsfunccall": {{Name: "NewConfig", Description: "NewConfig creates a new Config instance\n", Returns: []ReturnSet{{TypeName: "Config", TypePkgName: "varsfunccall"}}}},
				},
				Methods: map[types.PkgName][]MethodSet{},
				Vars: map[types.PkgName][]VarSet{
					"varsfunccall": {{Name: "ConfigVar", Description: "ConfigVar is initialized by a function call\n", TypeName: "Config", TypePkgName: "varsfunccall"}},
				},
				Consts: map[types.PkgName][]ConstSet{},
				Structs: map[types.PkgName][]StructSet{
					"varsfunccall": {{Name: "Config", Fields: []types.StructFieldName{"Name"}, Description: "Config is a configuration struct\n"}},
				},
				Interfaces:   map[types.PkgName][]InterfaceSet{},
				DefinedTypes: map[types.PkgName][]DefinedTypeSet{},
			},
		},
		{
			name: "vars_methodcall",
			path: "./testdata/vars_methodcall",
			want: &SymbolIndex{
				Pkgs: []types.PkgName{"varsmethodcall"},
				Funcs: map[types.PkgName][]FuncSet{
					"varsmethodcall": {{Name: "NewLogger", Description: "NewLogger creates a new Logger instance\n", Returns: []ReturnSet{{TypeName: "Logger", TypePkgName: "varsmethodcall"}}}},
				},
				Methods: map[types.PkgName][]MethodSet{
					"varsmethodcall": {
						{Name: "Info", Description: "Info logs an info message and returns the logged message\n", ReceiverTypeName: "Logger", Returns: []ReturnSet{{TypeName: "string", TypePkgName: ""}}},
					},
				},
				Vars: map[types.PkgName][]VarSet{
					"varsmethodcall": {
						{Name: "GlobalLogger", Description: "GlobalLogger is a top-level variable\n", TypeName: "Logger", TypePkgName: "varsmethodcall"},
						{Name: "ResultFromMethod", Description: "ResultFromMethod is initialized by a method call on a top-level variable\n", TypeName: "string", TypePkgName: ""},
					},
				},
				Consts: map[types.PkgName][]ConstSet{},
				Structs: map[types.PkgName][]StructSet{
					"varsmethodcall": {{Name: "Logger", Fields: []types.StructFieldName{"Name"}, Description: "Logger is a simple logger type\n"}},
				},
				Interfaces:   map[types.PkgName][]InterfaceSet{},
				DefinedTypes: map[types.PkgName][]DefinedTypeSet{},
			},
		},
		{
			name: "consts",
			path: "./testdata/consts",
			want: &SymbolIndex{
				Pkgs:    []types.PkgName{"consts"},
				Funcs:   map[types.PkgName][]FuncSet{},
				Methods: map[types.PkgName][]MethodSet{},
				Vars:    map[types.PkgName][]VarSet{},
				Consts: map[types.PkgName][]ConstSet{
					"consts": {
						{Name: "MaxSize", Description: "MaxSize is the maximum size\n"},
						{Name: "MinValue", Description: "MinValue is the minimum value\n"},
						{Name: "MaxValue", Description: "MaxValue is the maximum value\n"},
						{Name: "DefaultWidth", Description: "Multiple names in one spec\n"},
						{Name: "DefaultHeight", Description: "Multiple names in one spec\n"},
					},
				},
				Structs:      map[types.PkgName][]StructSet{},
				Interfaces:   map[types.PkgName][]InterfaceSet{},
				DefinedTypes: map[types.PkgName][]DefinedTypeSet{},
			},
		},
		{
			name: "structs",
			path: "./testdata/structs",
			want: &SymbolIndex{
				Pkgs:    []types.PkgName{"structs"},
				Funcs:   map[types.PkgName][]FuncSet{},
				Methods: map[types.PkgName][]MethodSet{},
				Vars:    map[types.PkgName][]VarSet{},
				Consts:  map[types.PkgName][]ConstSet{},
				Structs: map[types.PkgName][]StructSet{
					"structs": {
						{Name: "SimpleStruct", Fields: []types.StructFieldName{"FieldA", "FieldB"}, Description: "SimpleStruct has simple fields\n"},
						{Name: "MultiFieldStruct", Fields: []types.StructFieldName{"X", "Y", "Z"}, Description: "MultiFieldStruct has multiple fields on same line\n"},
						{Name: "Base", Fields: []types.StructFieldName{"ID"}, Description: "Base is used for embedding\n"},
						{Name: "Derived", Fields: []types.StructFieldName{"Base", "Name"}, Description: "Derived embeds Base\n"},
					},
				},
				Interfaces:   map[types.PkgName][]InterfaceSet{},
				DefinedTypes: map[types.PkgName][]DefinedTypeSet{},
			},
		},
		{
			name: "interfaces",
			path: "./testdata/interfaces",
			want: &SymbolIndex{
				Pkgs:    []types.PkgName{"interfaces"},
				Funcs:   map[types.PkgName][]FuncSet{},
				Methods: map[types.PkgName][]MethodSet{},
				Vars:    map[types.PkgName][]VarSet{},
				Consts:  map[types.PkgName][]ConstSet{},
				Structs: map[types.PkgName][]StructSet{},
				Interfaces: map[types.PkgName][]InterfaceSet{
					"interfaces": {
						{Name: "Reader", Methods: []types.DeclName{"Read"}, Descriptions: []string{"Read reads data\n"}},
						{Name: "Writer", Methods: []types.DeclName{"Write"}, Descriptions: []string{"Write writes data\n"}},
					},
				},
				DefinedTypes: map[types.PkgName][]DefinedTypeSet{},
			},
		},
		{
			name: "multipackage",
			path: "./testdata/multipackage",
			want: &SymbolIndex{
				Pkgs: []types.PkgName{"myapp", "types"},
				Funcs: map[types.PkgName][]FuncSet{
					"myapp": {
						{Name: "GetConfig", Description: "GetConfig returns a Config from another package\n", Returns: []ReturnSet{{TypeName: "Config", TypePkgName: "types"}}},
						{Name: "GetLogger", Description: "GetLogger returns a Logger from another package\n", Returns: []ReturnSet{{TypeName: "Logger", TypePkgName: "types"}}},
					},
				},
				Methods: map[types.PkgName][]MethodSet{
					"myapp": {
						{Name: "GetConfigFromMethod", Description: "GetConfigFromMethod returns a Config from another package via method\n", ReceiverTypeName: "Service", Returns: []ReturnSet{{TypeName: "Config", TypePkgName: "types"}}},
					},
					"types": {
						{Name: "Info", Description: "Info logs an info message\n", ReceiverTypeName: "Logger", Returns: []ReturnSet{{TypeName: "string", TypePkgName: ""}}},
					},
				},
				Vars:   map[types.PkgName][]VarSet{},
				Consts: map[types.PkgName][]ConstSet{},
				Structs: map[types.PkgName][]StructSet{
					"myapp": {{Name: "Service", Fields: []types.StructFieldName{"Name"}, Description: "Service has a method that returns a type from another package\n"}},
					"types": {
						{Name: "Config", Fields: []types.StructFieldName{"Name", "Value"}, Description: "Config is a configuration struct from another package\n"},
						{Name: "Logger", Fields: []types.StructFieldName{"Level"}, Description: "Logger is a logger struct from another package\n"},
					},
				},
				Interfaces:   map[types.PkgName][]InterfaceSet{},
				DefinedTypes: map[types.PkgName][]DefinedTypeSet{},
			},
		},
		{
			name: "multidecl",
			path: "./testdata/multidecl",
			want: &SymbolIndex{
				Pkgs:    []types.PkgName{"multidecl"},
				Funcs:   map[types.PkgName][]FuncSet{},
				Methods: map[types.PkgName][]MethodSet{},
				Vars: map[types.PkgName][]VarSet{
					"multidecl": {
						{Name: "VarA", Description: "Multiple variables in one line\n", TypeName: "string", TypePkgName: ""},
						{Name: "VarB", Description: "Multiple variables in one line\n", TypeName: "string", TypePkgName: ""},
						{Name: "VarC", Description: "Multiple variables in one var block\n", TypeName: "string", TypePkgName: ""},
						{Name: "VarD", Description: "Multiple variables in one var block\n", TypeName: "string", TypePkgName: ""},
					},
				},
				Consts: map[types.PkgName][]ConstSet{
					"multidecl": {
						{Name: "ConstA", Description: "Multiple constants in one line\n"},
						{Name: "ConstB", Description: "Multiple constants in one line\n"},
						{Name: "ConstC", Description: "Multiple constants in one const block\n"},
						{Name: "ConstD", Description: "Multiple constants in one const block\n"},
					},
				},
				Structs:      map[types.PkgName][]StructSet{},
				Interfaces:   map[types.PkgName][]InterfaceSet{},
				DefinedTypes: map[types.PkgName][]DefinedTypeSet{},
			},
		},
		{
			name: "defined_type",
			path: "./testdata/defined_type",
			want: &SymbolIndex{
				Pkgs:    []types.PkgName{"definedtype"},
				Funcs:   map[types.PkgName][]FuncSet{},
				Methods: map[types.PkgName][]MethodSet{},
				Vars:    map[types.PkgName][]VarSet{},
				Consts:  map[types.PkgName][]ConstSet{},
				Structs: map[types.PkgName][]StructSet{
					"definedtype": {{Name: "Config", Fields: []types.StructFieldName{"Name"}, Description: "Config is a struct type\n"}},
				},
				Interfaces: map[types.PkgName][]InterfaceSet{},
				DefinedTypes: map[types.PkgName][]DefinedTypeSet{
					"definedtype": {
						{Name: "MyInt", UnderlyingType: "int", UnderlyingTypePkgName: "", Description: "MyInt is a defined type based on int\n"},
						{Name: "MyString", UnderlyingType: "string", UnderlyingTypePkgName: "", Description: "MyString is a defined type based on string\n"},
						{Name: "MySlice", UnderlyingType: "[]string", UnderlyingTypePkgName: "", Description: "MySlice is a defined type based on slice\n"},
						{Name: "MyMap", UnderlyingType: "map[string]int", UnderlyingTypePkgName: "", Description: "MyMap is a defined type based on map\n"},
						{Name: "MyPtr", UnderlyingType: "Config", UnderlyingTypePkgName: "definedtype", Description: "MyPtr is a pointer to struct\n"},
						{Name: "MyIntPtr", UnderlyingType: "*int", UnderlyingTypePkgName: "", Description: "MyIntPtr is a pointer to int\n"},
						{Name: "MyFunc", UnderlyingType: "func(int) string", UnderlyingTypePkgName: "", Description: "MyFunc is a function type\n"},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewSymbolIndex(tt.path)
			if err != nil {
				t.Fatalf("NewSymbolIndex() error = %v", err)
			}

			opts := []cmp.Option{
				cmpopts.SortSlices(func(a, b types.PkgName) bool { return a < b }),
				cmpopts.SortSlices(func(a, b FuncSet) bool { return a.Name < b.Name }),
				cmpopts.SortSlices(func(a, b MethodSet) bool { return a.Name < b.Name }),
				cmpopts.SortSlices(func(a, b VarSet) bool { return a.Name < b.Name }),
				cmpopts.SortSlices(func(a, b ConstSet) bool { return a.Name < b.Name }),
				cmpopts.SortSlices(func(a, b StructSet) bool { return a.Name < b.Name }),
				cmpopts.SortSlices(func(a, b InterfaceSet) bool { return a.Name < b.Name }),
				cmpopts.SortSlices(func(a, b DefinedTypeSet) bool { return a.Name < b.Name }),
			}
			if diff := cmp.Diff(tt.want, got, opts...); diff != "" {
				t.Errorf("%s mismatch (-want +got):\n%s", tt.name, diff)
			}
		})
	}
}
