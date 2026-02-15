package completer

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kakkky/go-prompt"
	"github.com/kakkky/gonsole/declregistry"
	"github.com/kakkky/gonsole/symbols"
	"github.com/kakkky/gonsole/types"
)

func TestCompleter_Complete(t *testing.T) {
	tests := []struct {
		name             string
		inputText        string
		setupSymbolIndex *symbols.SymbolIndex
		setupRegistry    *declregistry.DeclRegistry
		expected         []prompt.Suggest
	}{
		{
			name:      "Complete package name",
			inputText: "myapp",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp", "mylib", "myutil"},
			},
			setupRegistry: declregistry.NewRegistry(),
			expected: []prompt.Suggest{
				{
					Text:        "myapp",
					DisplayText: "myapp",
					Description: "Package: ",
				},
			},
		},
		{
			name:      "Complete package name with multiple candidates",
			inputText: "my",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp", "mylib", "myutil"},
			},
			setupRegistry: declregistry.NewRegistry(),
			expected: []prompt.Suggest{
				{
					Text:        "myapp",
					DisplayText: "myapp",
					Description: "Package: ",
				},
				{
					Text:        "mylib",
					DisplayText: "mylib",
					Description: "Package: ",
				},
				{
					Text:        "myutil",
					DisplayText: "myutil",
					Description: "Package: ",
				},
			},
		},
		{
			name:      "Complete package name with & operator",
			inputText: "&myapp",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp", "mylib", "myutil"},
			},
			setupRegistry: declregistry.NewRegistry(),
			expected: []prompt.Suggest{
				{
					Text:        "&myapp",
					DisplayText: "myapp",
					Description: "Package: ",
				},
			},
		},
		{
			name:      "Complete functions",
			inputText: "myapp.P",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Funcs: map[types.PkgName][]symbols.FuncSet{
					"myapp": {
						{Name: "Print", Description: "Print outputs a message"},
						{Name: "Printf", Description: "Printf formats a message"},
						{Name: "Println", Description: "Println outputs a message with newline"},
					},
				},
			},
			setupRegistry: declregistry.NewRegistry(),
			expected: []prompt.Suggest{
				{
					Text:        "myapp.Print()",
					DisplayText: "Print",
					Description: "Function: Print outputs a message",
				},
				{
					Text:        "myapp.Printf()",
					DisplayText: "Printf",
					Description: "Function: Printf formats a message",
				},
				{
					Text:        "myapp.Println()",
					DisplayText: "Println",
					Description: "Function: Println outputs a message with newline",
				},
			},
		},
		{
			name:      "Complete variables",
			inputText: "myapp.S",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Vars: map[types.PkgName][]symbols.VarSet{
					"myapp": {
						{Name: "StdIn", Description: "Standard input", TypeName: types.TypeName("Stream"), TypePkgName: "myapp"},
						{Name: "StdOut", Description: "Standard output", TypeName: types.TypeName("Stream"), TypePkgName: "myapp"},
						{Name: "StdErr", Description: "Standard error", TypeName: types.TypeName("Stream"), TypePkgName: "myapp"},
					},
				},
			},
			setupRegistry: declregistry.NewRegistry(),
			expected: []prompt.Suggest{
				{
					Text:        "myapp.StdIn",
					DisplayText: "StdIn",
					Description: "Variable: Standard input",
				},
				{
					Text:        "myapp.StdOut",
					DisplayText: "StdOut",
					Description: "Variable: Standard output",
				},
				{
					Text:        "myapp.StdErr",
					DisplayText: "StdErr",
					Description: "Variable: Standard error",
				},
			},
		},
		{
			name:      "Complete constants",
			inputText: "mylib.M",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"mylib"},
				Consts: map[types.PkgName][]symbols.ConstSet{
					"mylib": {
						{Name: "MaxRetries", Description: "Maximum number of retries"},
						{Name: "MinBufferSize", Description: "Minimum buffer size"},
					},
				},
			},
			setupRegistry: declregistry.NewRegistry(),
			expected: []prompt.Suggest{
				{
					Text:        "mylib.MaxRetries",
					DisplayText: "MaxRetries",
					Description: "Constant: Maximum number of retries",
				},
				{
					Text:        "mylib.MinBufferSize",
					DisplayText: "MinBufferSize",
					Description: "Constant: Minimum buffer size",
				},
			},
		},
		{
			name:      "Complete structs",
			inputText: "myapp.C",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Structs: map[types.PkgName][]symbols.StructSet{
					"myapp": {
						{Name: "Client", Description: "A client for API calls", Fields: []types.StructFieldName{"Timeout", "BaseURL"}},
						{Name: "Config", Description: "A configuration structure", Fields: []types.StructFieldName{"Name", "Value"}},
					},
				},
			},
			setupRegistry: declregistry.NewRegistry(),
			expected: []prompt.Suggest{
				{
					Text:        "myapp.Client{Timeout: ,BaseURL: }",
					DisplayText: "Client",
					Description: "Struct: A client for API calls",
				},
				{
					Text:        "myapp.Config{Name: ,Value: }",
					DisplayText: "Config",
					Description: "Struct: A configuration structure",
				},
			},
		},
		{
			name:      "Complete structs with & operator",
			inputText: "&myapp.C",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Structs: map[types.PkgName][]symbols.StructSet{
					"myapp": {
						{Name: "Client", Description: "A client for API calls", Fields: []types.StructFieldName{"Timeout", "BaseURL"}},
						{Name: "Config", Description: "A configuration structure", Fields: []types.StructFieldName{"Name", "Value"}},
					},
				},
			},
			setupRegistry: declregistry.NewRegistry(),
			expected: []prompt.Suggest{
				{
					Text:        "&myapp.Client{Timeout: ,BaseURL: }",
					DisplayText: "Client",
					Description: "Struct: A client for API calls",
				},
				{
					Text:        "&myapp.Config{Name: ,Value: }",
					DisplayText: "Config",
					Description: "Struct: A configuration structure",
				},
			},
		},
		{
			name:      "Complete defined types",
			inputText: "myapp.M",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				DefinedTypes: map[types.PkgName][]symbols.DefinedTypeSet{
					"myapp": {
						{Name: "MyInt", UnderlyingType: "int", Description: "MyInt is a custom int type"},
						{Name: "MyString", UnderlyingType: "string", Description: "MyString is a custom string type"},
					},
				},
			},
			setupRegistry: declregistry.NewRegistry(),
			expected: []prompt.Suggest{
				{
					Text:        "myapp.MyInt()",
					DisplayText: "MyInt",
					Description: "DefinedType: MyInt is a custom int type",
				},
				{
					Text:        "myapp.MyString()",
					DisplayText: "MyString",
					Description: "DefinedType: MyString is a custom string type",
				},
			},
		},
		{
			name:      "Complete after variable declaration with =",
			inputText: "var client = myapp.C",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Structs: map[types.PkgName][]symbols.StructSet{
					"myapp": {
						{Name: "Client", Description: "A client for API calls", Fields: []types.StructFieldName{"Timeout", "BaseURL"}},
						{Name: "Config", Description: "A configuration structure", Fields: []types.StructFieldName{"Name", "Value"}},
					},
				},
			},
			setupRegistry: declregistry.NewRegistry(),
			expected: []prompt.Suggest{
				{
					Text:        "myapp.Client{Timeout: ,BaseURL: }",
					DisplayText: "Client",
					Description: "Struct: A client for API calls",
				},
				{
					Text:        "myapp.Config{Name: ,Value: }",
					DisplayText: "Config",
					Description: "Struct: A configuration structure",
				},
			},
		},
		{
			name:      "Complete methods of variable declared from struct literal",
			inputText: "client.",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Methods: map[types.PkgName][]symbols.MethodSet{
					"myapp": {
						{
							ReceiverTypeName: "Client",
							Name:             "Do",
							Description:      "Do executes a request",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Response"), TypePkgName: types.PkgName("myapp")},
								{TypeName: types.TypeName("error"), TypePkgName: types.PkgName("")},
							},
						},
						{
							ReceiverTypeName: "Client",
							Name:             "Get",
							Description:      "Get sends a GET request",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Response"), TypePkgName: types.PkgName("myapp")},
								{TypeName: types.TypeName("error"), TypePkgName: types.PkgName("")},
							},
						},
					},
				},
			},
			setupRegistry: func() *declregistry.DeclRegistry {
				registry := declregistry.NewRegistry()
				registry.Decls = append(registry.Decls, declregistry.Decl{
					Name:        "client",
					TypeName:    "Client",
					TypePkgName: "myapp",
				})

				return registry
			}(),
			expected: []prompt.Suggest{
				{
					Text:        "client.Do()",
					DisplayText: "Do",
					Description: "Method: Do executes a request",
				},
				{
					Text:        "client.Get()",
					DisplayText: "Get",
					Description: "Method: Get sends a GET request",
				},
			},
		},
		{
			name:      "Complete methods of variable declared from another variable",
			inputText: "stream.",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Vars: map[types.PkgName][]symbols.VarSet{
					"myapp": {
						{Name: "StdOut", Description: "Standard output", TypeName: types.TypeName("Stream"), TypePkgName: "myapp"},
					},
				},
				Methods: map[types.PkgName][]symbols.MethodSet{
					"myapp": {
						{
							ReceiverTypeName: "Stream",
							Name:             "Write",
							Description:      "Write writes data to the stream",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("int"), TypePkgName: types.PkgName("")},
								{TypeName: types.TypeName("error"), TypePkgName: types.PkgName("")},
							},
						},
						{
							ReceiverTypeName: "Stream",
							Name:             "Close",
							Description:      "Close closes the stream",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("error"), TypePkgName: types.PkgName("")},
							},
						},
					},
				},
			},
			setupRegistry: func() *declregistry.DeclRegistry {
				registry := declregistry.NewRegistry()
				registry.Decls = append(registry.Decls, declregistry.Decl{
					Name:        "stream",
					TypeName:    "Stream",
					TypePkgName: "myapp",
				})
				return registry
			}(),
			expected: []prompt.Suggest{
				{
					Text:        "stream.Write()",
					DisplayText: "Write",
					Description: "Method: Write writes data to the stream",
				},
				{
					Text:        "stream.Close()",
					DisplayText: "Close",
					Description: "Method: Close closes the stream",
				},
			},
		},
		{
			name:      "Complete methods of variable declared from function return",
			inputText: "response.",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Funcs: map[types.PkgName][]symbols.FuncSet{
					"myapp": {
						{
							Name:        "FetchData",
							Description: "FetchData retrieves data from a source",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Response"), TypePkgName: types.PkgName("myapp")},
								{TypeName: types.TypeName("error"), TypePkgName: types.PkgName("")},
							},
						},
					},
				},
				Methods: map[types.PkgName][]symbols.MethodSet{
					"myapp": {
						{
							ReceiverTypeName: "Response",
							Name:             "GetContent",
							Description:      "GetContent returns the response content",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Content"), TypePkgName: types.PkgName("myapp")},
							},
						},
					},
				},
			},
			setupRegistry: func() *declregistry.DeclRegistry {
				registry := declregistry.NewRegistry()
				registry.Decls = append(registry.Decls, declregistry.Decl{
					Name:        "response",
					TypeName:    "Response",
					TypePkgName: "myapp",
				})
				return registry
			}(),
			expected: []prompt.Suggest{
				{
					Text:        "response.GetContent()",
					DisplayText: "GetContent",
					Description: "Method: GetContent returns the response content",
				},
			},
		},
		{
			name:      "Complete methods of variable declared from method return",
			inputText: "content.",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Funcs: map[types.PkgName][]symbols.FuncSet{
					"myapp": {
						{
							Name:        "FetchData",
							Description: "FetchData retrieves data from a source",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Response"), TypePkgName: types.PkgName("myapp")},
								{TypeName: types.TypeName("error"), TypePkgName: types.PkgName("")},
							},
						},
					},
				},
				Methods: map[types.PkgName][]symbols.MethodSet{
					"myapp": {
						{
							ReceiverTypeName: "Response",
							Name:             "GetContent",
							Description:      "GetContent returns the response content",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Content"), TypePkgName: types.PkgName("myapp")},
							},
						},
						{
							ReceiverTypeName: "Content",
							Name:             "Read",
							Description:      "Read reads data from the content",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("int"), TypePkgName: types.PkgName("")},
								{TypeName: types.TypeName("error"), TypePkgName: types.PkgName("")},
							},
						},
						{
							ReceiverTypeName: "Content",
							Name:             "Type",
							Description:      "Type returns the content type",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("string"), TypePkgName: types.PkgName("")},
							},
						},
					},
				},
			},
			setupRegistry: func() *declregistry.DeclRegistry {
				registry := declregistry.NewRegistry()
				registry.Decls = append(registry.Decls, declregistry.Decl{
					Name:        "response",
					TypeName:    "Response",
					TypePkgName: "myapp",
				},
					declregistry.Decl{
						Name:        "content",
						TypeName:    "Content",
						TypePkgName: "myapp",
					})
				return registry
			}(),
			expected: []prompt.Suggest{
				{
					Text:        "content.Read()",
					DisplayText: "Read",
					Description: "Method: Read reads data from the content",
				},
				{
					Text:        "content.Type()",
					DisplayText: "Type",
					Description: "Method: Type returns the content type",
				},
			},
		},
		{
			name:      "Complete methods of variable storing interface return value",
			inputText: "reader.",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Funcs: map[types.PkgName][]symbols.FuncSet{
					"myapp": {
						{
							Name:        "NewReader",
							Description: "NewReader creates a new reader",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Reader"), TypePkgName: types.PkgName("myapp")},
								{TypeName: types.TypeName("error"), TypePkgName: types.PkgName("")},
							},
						},
					},
				},
				Methods: map[types.PkgName][]symbols.MethodSet{
					"myapp": {
						{
							ReceiverTypeName: "MyReader",
							Name:             "Read",
							Description:      "Read reads data from the reader",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("int"), TypePkgName: types.PkgName("")},
								{TypeName: types.TypeName("error"), TypePkgName: types.PkgName("")},
							},
						},
						{
							ReceiverTypeName: "MyReader",
							Name:             "Close",
							Description:      "Close closes the reader",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("error"), TypePkgName: types.PkgName("")},
							},
						},
					},
				},
				Interfaces: map[types.PkgName][]symbols.InterfaceSet{
					"myapp": {
						{
							Name:         "Reader",
							Methods:      []types.DeclName{"Read", "Close"},
							Descriptions: []string{"Read reads data from the reader", "Close closes the reader"},
						},
					},
				},
			},
			setupRegistry: func() *declregistry.DeclRegistry {
				registry := declregistry.NewRegistry()
				registry.Decls = append(registry.Decls, declregistry.Decl{
					Name:        "reader",
					TypeName:    "MyReader",
					TypePkgName: "myapp",
				})
				return registry
			}(),
			expected: []prompt.Suggest{
				{
					Text:        "reader.Read()",
					DisplayText: "Read",
					Description: "Method: Read reads data from the reader",
				},
				{
					Text:        "reader.Close()",
					DisplayText: "Close",
					Description: "Method: Close closes the reader",
				},
			},
		},
		{
			name:      "Complete methods of variable storing interface return value from method",
			inputText: "resource.",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Funcs: map[types.PkgName][]symbols.FuncSet{
					"myapp": {
						{
							Name:        "CreateClient",
							Description: "CreateClient creates a new client",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Client"), TypePkgName: types.PkgName("myapp")},
								{TypeName: types.TypeName("error"), TypePkgName: types.PkgName("")},
							},
						},
					},
				},
				Methods: map[types.PkgName][]symbols.MethodSet{
					"myapp": {
						{
							ReceiverTypeName: "Client",
							Name:             "GetResource",
							Description:      "GetResource returns a resource interface",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Resource"), TypePkgName: types.PkgName("myapp")},
							},
						},
					},
				},
				Interfaces: map[types.PkgName][]symbols.InterfaceSet{
					"myapp": {
						{
							Name:    "Resource",
							Methods: []types.DeclName{"Open", "Save", "Delete"},
							Descriptions: []string{
								"Open opens the resource",
								"Save saves the resource",
								"Delete deletes the resource",
							},
						},
					},
				},
			},
			setupRegistry: func() *declregistry.DeclRegistry {
				registry := declregistry.NewRegistry()
				registry.Decls = append(registry.Decls, declregistry.Decl{
					Name:        "client",
					TypeName:    "Client",
					TypePkgName: "myapp",
				},
					declregistry.Decl{
						Name:        "resource",
						TypeName:    "Resource",
						TypePkgName: "myapp",
					},
				)
				return registry
			}(),
			expected: []prompt.Suggest{
				{
					Text:        "resource.Open()",
					DisplayText: "Open",
					Description: "Method: Open opens the resource",
				},
				{
					Text:        "resource.Save()",
					DisplayText: "Save",
					Description: "Method: Save saves the resource",
				},
				{
					Text:        "resource.Delete()",
					DisplayText: "Delete",
					Description: "Method: Delete deletes the resource",
				},
			},
		},
		{
			name:      "Do not complete private symbols",
			inputText: "myapp.",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Funcs: map[types.PkgName][]symbols.FuncSet{
					"myapp": {
						{Name: "Print", Description: "Print outputs a message"},
						{Name: "printf", Description: "Internal printing function"},
					},
				},
				Vars: map[types.PkgName][]symbols.VarSet{
					"myapp": {
						{Name: "Version", Description: "Package:  version"},
						{Name: "privateVar", Description: "Internal variable"},
					},
				},
			},
			setupRegistry: declregistry.NewRegistry(),
			expected: []prompt.Suggest{
				{
					Text:        "myapp.Print()",
					DisplayText: "Print",
					Description: "Function: Print outputs a message",
				},
				{
					Text:        "myapp.Version",
					DisplayText: "Version",
					Description: "Variable: Package:  version",
				},
			},
		},
		// --- ここからメソッドチェーンのテストケース追加 ---
		{
			name:      "Method chain after function with single return value",
			inputText: "myapp.NewClient().",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Funcs: map[types.PkgName][]symbols.FuncSet{
					"myapp": {
						{
							Name:        "NewClient",
							Description: "Create new client",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Client"), TypePkgName: "myapp"},
							},
						},
					},
				},
				Methods: map[types.PkgName][]symbols.MethodSet{
					"myapp": {
						{
							ReceiverTypeName: "Client",
							Name:             "Do",
							Description:      "Do something",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Result"), TypePkgName: "myapp"},
							},
						},
					},
				},
			},
			setupRegistry: declregistry.NewRegistry(),
			expected: []prompt.Suggest{
				{
					Text:        "myapp.NewClient().Do()",
					DisplayText: "Do",
					Description: "Method: Do something",
				},
			},
		},
		{
			name:      "No method chain after function with multiple return values",
			inputText: "myapp.NewClient().",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Funcs: map[types.PkgName][]symbols.FuncSet{
					"myapp": {
						{
							Name:        "NewClient",
							Description: "Create new client",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Client"), TypePkgName: "myapp"},
								{TypeName: types.TypeName("error"), TypePkgName: ""},
							},
						},
					},
				},
				Methods: map[types.PkgName][]symbols.MethodSet{
					"myapp": {
						{
							ReceiverTypeName: "Client",
							Name:             "Do",
							Description:      "Do something",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Result"), TypePkgName: "myapp"},
							},
						},
					},
				},
			},
			setupRegistry: declregistry.NewRegistry(),
			expected:      nil,
		},
		{
			name:      "Method chain after function returning interface",
			inputText: "myapp.NewReader().",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Funcs: map[types.PkgName][]symbols.FuncSet{
					"myapp": {
						{
							Name:        "NewReader",
							Description: "Create new reader",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Reader"), TypePkgName: "myapp"},
							},
						},
					},
				},
				Interfaces: map[types.PkgName][]symbols.InterfaceSet{
					"myapp": {
						{
							Name:         "Reader",
							Methods:      []types.DeclName{"Read", "Close"},
							Descriptions: []string{"Read reads data", "Close closes reader"},
						},
					},
				},
			},
			setupRegistry: declregistry.NewRegistry(),
			expected: []prompt.Suggest{
				{
					Text:        "myapp.NewReader().Read()",
					DisplayText: "Read",
					Description: "Method: Read reads data",
				},
				{
					Text:        "myapp.NewReader().Close()",
					DisplayText: "Close",
					Description: "Method: Close closes reader",
				},
			},
		},
		{
			name:      "Method chain after method with single return value",
			inputText: "client.GetResource().",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Methods: map[types.PkgName][]symbols.MethodSet{
					"myapp": {
						{
							ReceiverTypeName: "Client",
							Name:             "GetResource",
							Description:      "Get resource",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Resource"), TypePkgName: types.PkgName("myapp")},
							},
						},
						{
							ReceiverTypeName: "Resource",
							Name:             "Open",
							Description:      "Open resource",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("error"), TypePkgName: types.PkgName("")},
							},
						},
					},
				},
			},
			setupRegistry: func() *declregistry.DeclRegistry {
				registry := declregistry.NewRegistry()
				registry.Decls = append(registry.Decls, declregistry.Decl{
					Name:        "client",
					TypeName:    "Client",
					TypePkgName: "myapp",
				})
				return registry
			}(),
			expected: []prompt.Suggest{
				{
					Text:        "client.GetResource().Open()",
					DisplayText: "Open",
					Description: "Method: Open resource",
				},
			},
		},
		{
			name:      "No method chain after method with multiple return values",
			inputText: "client.GetResource().",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Methods: map[types.PkgName][]symbols.MethodSet{
					"myapp": {
						{
							ReceiverTypeName: "Client",
							Name:             "GetResource",
							Description:      "Get resource",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Resource"), TypePkgName: types.PkgName("myapp")},
								{TypeName: types.TypeName("error"), TypePkgName: types.PkgName("")},
							},
						},
						{
							ReceiverTypeName: "Resource",
							Name:             "Open",
							Description:      "Open resource",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("error"), TypePkgName: types.PkgName("")},
							},
						},
					},
				},
			},
			setupRegistry: func() *declregistry.DeclRegistry {
				registry := declregistry.NewRegistry()
				registry.Decls = append(registry.Decls, declregistry.Decl{
					Name:        "client",
					TypeName:    "Client",
					TypePkgName: "myapp",
				})
				return registry
			}(),
			expected: nil,
		},
		{
			name:      "Method chain after interface-returning method chain",
			inputText: "reader.Read().",
			setupSymbolIndex: &symbols.SymbolIndex{
				Pkgs: []types.PkgName{"myapp"},
				Funcs: map[types.PkgName][]symbols.FuncSet{
					"myapp": {
						{
							Name:        "NewReader",
							Description: "NewReader creates a new reader",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Reader"), TypePkgName: types.PkgName("myapp")},
							},
						},
					},
				},
				Methods: map[types.PkgName][]symbols.MethodSet{
					"myapp": {
						{
							ReceiverTypeName: "Reader",
							Name:             "Read",
							Description:      "Read reads data from the reader",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("Reader"), TypePkgName: types.PkgName("myapp")},
							},
						},
						{
							ReceiverTypeName: "Reader",
							Name:             "Close",
							Description:      "Close closes the reader",
							Returns: []symbols.ReturnSet{
								{TypeName: types.TypeName("error"), TypePkgName: types.PkgName("")},
							},
						},
					},
				},
				Interfaces: map[types.PkgName][]symbols.InterfaceSet{
					"myapp": {
						{
							Name:         "Reader",
							Methods:      []types.DeclName{"Read", "Close"},
							Descriptions: []string{"Read reads data from the reader", "Close closes the reader"},
						},
					},
				},
			},
			setupRegistry: func() *declregistry.DeclRegistry {
				registry := declregistry.NewRegistry()
				registry.Decls = append(registry.Decls, declregistry.Decl{
					Name:        "reader",
					TypeName:    "Reader",
					TypePkgName: "myapp",
				})
				return registry
			}(),
			expected: []prompt.Suggest{
				{
					Text:        "reader.Read().Read()",
					DisplayText: "Read",
					Description: "Method: Read reads data from the reader",
				},
				{
					Text:        "reader.Read().Close()",
					DisplayText: "Close",
					Description: "Method: Close closes the reader",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completer := Completer{
				symbolIndex:  tt.setupSymbolIndex,
				declRegistry: tt.setupRegistry,
			}
			doc := prompt.Document{
				Text: tt.inputText,
			}

			got := completer.Complete(doc)

			// 結果を比較（順序は考慮しない）
			opts := []cmp.Option{
				cmp.AllowUnexported(prompt.Suggest{}),
				// prompt.SuggestのTextで順序を無視して比較
				cmpopts.SortSlices(func(a, b prompt.Suggest) bool {
					return a.Text < b.Text
				}),
			}

			if diff := cmp.Diff(tt.expected, got, opts...); diff != "" {
				t.Errorf("Complete() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
