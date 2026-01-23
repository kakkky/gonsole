package completer

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kakkky/go-prompt"
	"github.com/kakkky/gonsole/declregistry"
	"github.com/kakkky/gonsole/types"
)

func TestCompleter_Complete(t *testing.T) {
	tests := []struct {
		name            string
		inputText       string
		setupCandidates *candidates
		setupRegistry   *declregistry.DeclRegistry
		expected        []prompt.Suggest
	}{
		{
			name:      "Complete package name",
			inputText: "myapp",
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp", "mylib", "myutil"},
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp", "mylib", "myutil"},
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp", "mylib", "myutil"},
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				funcs: map[types.PkgName][]funcSet{
					"myapp": {
						{name: "Print", description: "Print outputs a message"},
						{name: "Printf", description: "Printf formats a message"},
						{name: "Println", description: "Println outputs a message with newline"},
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				vars: map[types.PkgName][]varSet{
					"myapp": {
						{name: "StdIn", description: "Standard input", typeName: types.TypeName("Stream"), pkgName: "myapp"},
						{name: "StdOut", description: "Standard output", typeName: types.TypeName("Stream"), pkgName: "myapp"},
						{name: "StdErr", description: "Standard error", typeName: types.TypeName("Stream"), pkgName: "myapp"},
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"mylib"},
				consts: map[types.PkgName][]constSet{
					"mylib": {
						{name: "MaxRetries", description: "Maximum number of retries"},
						{name: "MinBufferSize", description: "Minimum buffer size"},
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				structs: map[types.PkgName][]structSet{
					"myapp": {
						{name: "Client", description: "A client for API calls", fields: []types.StructFieldName{"Timeout", "BaseURL"}},
						{name: "Config", description: "A configuration structure", fields: []types.StructFieldName{"Name", "Value"}},
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				structs: map[types.PkgName][]structSet{
					"myapp": {
						{name: "Client", description: "A client for API calls", fields: []types.StructFieldName{"Timeout", "BaseURL"}},
						{name: "Config", description: "A configuration structure", fields: []types.StructFieldName{"Name", "Value"}},
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
			name:      "Complete after variable declaration with =",
			inputText: "var client = myapp.C",
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				structs: map[types.PkgName][]structSet{
					"myapp": {
						{name: "Client", description: "A client for API calls", fields: []types.StructFieldName{"Timeout", "BaseURL"}},
						{name: "Config", description: "A configuration structure", fields: []types.StructFieldName{"Name", "Value"}},
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				methods: map[types.PkgName][]methodSet{
					"myapp": {
						{
							receiverTypeName: "Client",
							name:             "Do",
							description:      "Do executes a request",
							returns: []returnSet{
								{typeName: types.TypeName("Response"), pkgName: types.PkgName("myapp")},
								{typeName: types.TypeName("error"), pkgName: types.PkgName("")},
							},
						},
						{
							receiverTypeName: "Client",
							name:             "Get",
							description:      "Get sends a GET request",
							returns: []returnSet{
								{typeName: types.TypeName("Response"), pkgName: types.PkgName("myapp")},
								{typeName: types.TypeName("error"), pkgName: types.PkgName("")},
							},
						},
					},
				},
			},
			setupRegistry: func() *declregistry.DeclRegistry {
				registry := declregistry.NewRegistry()
				if err := registry.Register("client := myapp.Client{}"); err != nil {
					t.Fatalf("failed to register client variable: %v", err)
				}
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				vars: map[types.PkgName][]varSet{
					"myapp": {
						{name: "StdOut", description: "Standard output", typeName: types.TypeName("Stream"), pkgName: "myapp"},
					},
				},
				methods: map[types.PkgName][]methodSet{
					"myapp": {
						{
							receiverTypeName: "Stream",
							name:             "Write",
							description:      "Write writes data to the stream",
							returns: []returnSet{
								{typeName: types.TypeName("int"), pkgName: types.PkgName("")},
								{typeName: types.TypeName("error"), pkgName: types.PkgName("")},
							},
						},
						{
							receiverTypeName: "Stream",
							name:             "Close",
							description:      "Close closes the stream",
							returns: []returnSet{
								{typeName: types.TypeName("error"), pkgName: types.PkgName("")},
							},
						},
					},
				},
			},
			setupRegistry: func() *declregistry.DeclRegistry {
				registry := declregistry.NewRegistry()
				if err := registry.Register("stream := myapp.StdOut"); err != nil {
					t.Fatalf("failed to register stream variable: %v", err)
				}
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				funcs: map[types.PkgName][]funcSet{
					"myapp": {
						{
							name:        "FetchData",
							description: "FetchData retrieves data from a source",
							returns: []returnSet{
								{typeName: types.TypeName("Response"), pkgName: types.PkgName("myapp")},
								{typeName: types.TypeName("error"), pkgName: types.PkgName("")},
							},
						},
					},
				},
				methods: map[types.PkgName][]methodSet{
					"myapp": {
						{
							receiverTypeName: "Response",
							name:             "GetContent",
							description:      "GetContent returns the response content",
							returns: []returnSet{
								{typeName: types.TypeName("Content"), pkgName: types.PkgName("myapp")},
							},
						},
					},
				},
			},
			setupRegistry: func() *declregistry.DeclRegistry {
				registry := declregistry.NewRegistry()
				if err := registry.Register("response, _ := myapp.FetchData()"); err != nil {
					t.Fatalf("failed to register response variable: %v", err)
				}
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				funcs: map[types.PkgName][]funcSet{
					"myapp": {
						{
							name:        "FetchData",
							description: "FetchData retrieves data from a source",
							returns: []returnSet{
								{typeName: types.TypeName("Response"), pkgName: types.PkgName("myapp")},
								{typeName: types.TypeName("error"), pkgName: types.PkgName("")},
							},
						},
					},
				},
				methods: map[types.PkgName][]methodSet{
					"myapp": {
						{
							receiverTypeName: "Response",
							name:             "GetContent",
							description:      "GetContent returns the response content",
							returns: []returnSet{
								{typeName: types.TypeName("Content"), pkgName: types.PkgName("myapp")},
							},
						},
						{
							receiverTypeName: "Content",
							name:             "Read",
							description:      "Read reads data from the content",
							returns: []returnSet{
								{typeName: types.TypeName("int"), pkgName: types.PkgName("")},
								{typeName: types.TypeName("error"), pkgName: types.PkgName("")},
							},
						},
						{
							receiverTypeName: "Content",
							name:             "Type",
							description:      "Type returns the content type",
							returns: []returnSet{
								{typeName: types.TypeName("string"), pkgName: types.PkgName("")},
							},
						},
					},
				},
			},
			setupRegistry: func() *declregistry.DeclRegistry {
				registry := declregistry.NewRegistry()
				// まず response 変数を作成
				if err := registry.Register("response, _ := myapp.FetchData()"); err != nil {
					t.Fatalf("failed to register response variable: %v", err)
				}
				// 次に content 変数を response.GetContent() から作成
				if err := registry.Register("content := response.GetContent()"); err != nil {
					t.Fatalf("failed to register content variable: %v", err)
				}
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				funcs: map[types.PkgName][]funcSet{
					"myapp": {
						{
							name:        "NewReader",
							description: "NewReader creates a new reader",
							returns: []returnSet{
								{typeName: types.TypeName("Reader"), pkgName: types.PkgName("myapp")},
								{typeName: types.TypeName("error"), pkgName: types.PkgName("")},
							},
						},
					},
				},
				methods: map[types.PkgName][]methodSet{
					"myapp": {
						{
							receiverTypeName: "MyReader",
							name:             "Read",
							description:      "Read reads data from the reader",
							returns: []returnSet{
								{typeName: types.TypeName("int"), pkgName: types.PkgName("")},
								{typeName: types.TypeName("error"), pkgName: types.PkgName("")},
							},
						},
						{
							receiverTypeName: "MyReader",
							name:             "Close",
							description:      "Close closes the reader",
							returns: []returnSet{
								{typeName: types.TypeName("error"), pkgName: types.PkgName("")},
							},
						},
					},
				},
				interfaces: map[types.PkgName][]interfaceSet{
					"myapp": {
						{
							name:         "Reader",
							methods:      []types.DeclName{"Read", "Close"},
							descriptions: []string{"Read reads data from the reader", "Close closes the reader"},
						},
					},
				},
			},
			setupRegistry: func() *declregistry.DeclRegistry {
				registry := declregistry.NewRegistry()
				if err := registry.Register("reader, _ := myapp.NewReader()"); err != nil {
					t.Fatalf("failed to register reader variable: %v", err)
				}
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				funcs: map[types.PkgName][]funcSet{
					"myapp": {
						{
							name:        "CreateClient",
							description: "CreateClient creates a new client",
							returns: []returnSet{
								{typeName: types.TypeName("Client"), pkgName: types.PkgName("myapp")},
								{typeName: types.TypeName("error"), pkgName: types.PkgName("")},
							},
						},
					},
				},
				methods: map[types.PkgName][]methodSet{
					"myapp": {
						{
							receiverTypeName: "Client",
							name:             "GetResource",
							description:      "GetResource returns a resource interface",
							returns: []returnSet{
								{typeName: types.TypeName("Resource"), pkgName: types.PkgName("myapp")},
							},
						},
					},
				},
				interfaces: map[types.PkgName][]interfaceSet{
					"myapp": {
						{
							name:    "Resource",
							methods: []types.DeclName{"Open", "Save", "Delete"},
							descriptions: []string{
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
				// まずクライアント変数を作成
				if err := registry.Register("client, _ := myapp.CreateClient()"); err != nil {
					t.Fatalf("failed to register client variable: %v", err)
				}
				// 次にリソース変数をクライアントのメソッドから作成
				if err := registry.Register("resource := client.GetResource()"); err != nil {
					t.Fatalf("failed to register resource variable: %v", err)
				}
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				funcs: map[types.PkgName][]funcSet{
					"myapp": {
						{name: "Print", description: "Print outputs a message"},
						{name: "printf", description: "Internal printing function"},
					},
				},
				vars: map[types.PkgName][]varSet{
					"myapp": {
						{name: "Version", description: "Package:  version"},
						{name: "privateVar", description: "Internal variable"},
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				funcs: map[types.PkgName][]funcSet{
					"myapp": {
						{
							name:        "NewClient",
							description: "Create new client",
							returns: []returnSet{
								{typeName: types.TypeName("Client"), pkgName: "myapp"},
							},
						},
					},
				},
				methods: map[types.PkgName][]methodSet{
					"myapp": {
						{
							receiverTypeName: "Client",
							name:             "Do",
							description:      "Do something",
							returns: []returnSet{
								{typeName: types.TypeName("Result"), pkgName: "myapp"},
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				funcs: map[types.PkgName][]funcSet{
					"myapp": {
						{
							name:        "NewClient",
							description: "Create new client",
							returns: []returnSet{
								{typeName: types.TypeName("Client"), pkgName: "myapp"},
								{typeName: types.TypeName("error"), pkgName: ""},
							},
						},
					},
				},
				methods: map[types.PkgName][]methodSet{
					"myapp": {
						{
							receiverTypeName: "Client",
							name:             "Do",
							description:      "Do something",
							returns: []returnSet{
								{typeName: types.TypeName("Result"), pkgName: "myapp"},
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				funcs: map[types.PkgName][]funcSet{
					"myapp": {
						{
							name:        "NewReader",
							description: "Create new reader",
							returns: []returnSet{
								{typeName: types.TypeName("Reader"), pkgName: "myapp"},
							},
						},
					},
				},
				interfaces: map[types.PkgName][]interfaceSet{
					"myapp": {
						{
							name:         "Reader",
							methods:      []types.DeclName{"Read", "Close"},
							descriptions: []string{"Read reads data", "Close closes reader"},
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				methods: map[types.PkgName][]methodSet{
					"myapp": {
						{
							receiverTypeName: "Client",
							name:             "GetResource",
							description:      "Get resource",
							returns: []returnSet{
								{typeName: types.TypeName("Resource"), pkgName: types.PkgName("myapp")},
							},
						},
						{
							receiverTypeName: "Resource",
							name:             "Open",
							description:      "Open resource",
							returns: []returnSet{
								{typeName: types.TypeName("error"), pkgName: types.PkgName("")},
							},
						},
					},
				},
			},
			setupRegistry: func() *declregistry.DeclRegistry {
				registry := declregistry.NewRegistry()
				_ = registry.Register("client := myapp.Client{}")
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
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				methods: map[types.PkgName][]methodSet{
					"myapp": {
						{
							receiverTypeName: "Client",
							name:             "GetResource",
							description:      "Get resource",
							returns: []returnSet{
								{typeName: types.TypeName("Resource"), pkgName: types.PkgName("myapp")},
								{typeName: types.TypeName("error"), pkgName: types.PkgName("")},
							},
						},
						{
							receiverTypeName: "Resource",
							name:             "Open",
							description:      "Open resource",
							returns: []returnSet{
								{typeName: types.TypeName("error"), pkgName: types.PkgName("")},
							},
						},
					},
				},
			},
			setupRegistry: func() *declregistry.DeclRegistry {
				registry := declregistry.NewRegistry()
				_ = registry.Register("client := myapp.Client{}")
				return registry
			}(),
			expected: nil,
		},
		{
			name:      "Method chain after interface-returning method chain",
			inputText: "reader.Read().",
			setupCandidates: &candidates{
				pkgs: []types.PkgName{"myapp"},
				funcs: map[types.PkgName][]funcSet{
					"myapp": {
						{
							name:        "NewReader",
							description: "NewReader creates a new reader",
							returns: []returnSet{
								{typeName: types.TypeName("Reader"), pkgName: types.PkgName("myapp")},
							},
						},
					},
				},
				methods: map[types.PkgName][]methodSet{
					"myapp": {
						{
							receiverTypeName: "Reader",
							name:             "Read",
							description:      "Read reads data from the reader",
							returns: []returnSet{
								{typeName: types.TypeName("Reader"), pkgName: types.PkgName("myapp")},
							},
						},
						{
							receiverTypeName: "Reader",
							name:             "Close",
							description:      "Close closes the reader",
							returns: []returnSet{
								{typeName: types.TypeName("error"), pkgName: types.PkgName("")},
							},
						},
					},
				},
				interfaces: map[types.PkgName][]interfaceSet{
					"myapp": {
						{
							name:         "Reader",
							methods:      []types.DeclName{"Read", "Close"},
							descriptions: []string{"Read reads data from the reader", "Close closes the reader"},
						},
					},
				},
			},
			setupRegistry: func() *declregistry.DeclRegistry {
				registry := declregistry.NewRegistry()
				_ = registry.Register("reader := myapp.NewReader()")
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
				candidates:   tt.setupCandidates,
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
