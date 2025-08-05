package completer

import (
	"testing"

	"github.com/c-bata/go-prompt"
	"github.com/google/go-cmp/cmp"
	"github.com/kakkky/gonsole/decls"
)

func TestCompleter_Complete(t *testing.T) {
	tests := []struct {
		name            string
		inputText       string
		setupCandidates *candidates
		setupDeclEntry  *decls.DeclEntry
		expected        []prompt.Suggest
	}{
		{
			name:      "Complete package name",
			inputText: "myapp",
			setupCandidates: &candidates{
				pkgs: []pkgName{"myapp", "mylib", "myutil"},
			},
			setupDeclEntry: decls.NewDeclEntry(),
			expected: []prompt.Suggest{
				{
					Text:        "myapp",
					DisplayText: "myapp",
					Description: "Package",
				},
			},
		},
		{
			name:      "Complete package name with multiple candidates",
			inputText: "my",
			setupCandidates: &candidates{
				pkgs: []pkgName{"myapp", "mylib", "myutil"},
			},
			setupDeclEntry: decls.NewDeclEntry(),
			expected: []prompt.Suggest{
				{
					Text:        "myapp",
					DisplayText: "myapp",
					Description: "Package",
				},
				{
					Text:        "mylib",
					DisplayText: "mylib",
					Description: "Package",
				},
				{
					Text:        "myutil",
					DisplayText: "myutil",
					Description: "Package",
				},
			},
		},
		{
			name:      "Complete package name with & operator",
			inputText: "&myapp",
			setupCandidates: &candidates{
				pkgs: []pkgName{"myapp", "mylib", "myutil"},
			},
			setupDeclEntry: decls.NewDeclEntry(),
			expected: []prompt.Suggest{
				{
					Text:        "&myapp",
					DisplayText: "myapp",
					Description: "Package",
				},
			},
		},
		{
			name:      "Complete functions",
			inputText: "myapp.P",
			setupCandidates: &candidates{
				pkgs: []pkgName{"myapp"},
				funcs: map[pkgName][]funcSet{
					"myapp": {
						{name: "Print", description: "Print outputs a message"},
						{name: "Printf", description: "Printf formats a message"},
						{name: "Println", description: "Println outputs a message with newline"},
					},
				},
			},
			setupDeclEntry: decls.NewDeclEntry(),
			expected: []prompt.Suggest{
				{
					Text:        "myapp.Print()",
					DisplayText: "Print()",
					Description: "Function: Print outputs a message",
				},
				{
					Text:        "myapp.Printf()",
					DisplayText: "Printf()",
					Description: "Function: Printf formats a message",
				},
				{
					Text:        "myapp.Println()",
					DisplayText: "Println()",
					Description: "Function: Println outputs a message with newline",
				},
			},
		},
		{
			name:      "Complete variables",
			inputText: "myapp.S",
			setupCandidates: &candidates{
				pkgs: []pkgName{"myapp"},
				vars: map[pkgName][]varSet{
					"myapp": {
						{name: "StdIn", description: "Standard input", typeName: "Stream", typePkgName: "myapp"},
						{name: "StdOut", description: "Standard output", typeName: "Stream", typePkgName: "myapp"},
						{name: "StdErr", description: "Standard error", typeName: "Stream", typePkgName: "myapp"},
					},
				},
			},
			setupDeclEntry: decls.NewDeclEntry(),
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
				pkgs: []pkgName{"mylib"},
				consts: map[pkgName][]constSet{
					"mylib": {
						{name: "MaxRetries", description: "Maximum number of retries"},
						{name: "MinBufferSize", description: "Minimum buffer size"},
					},
				},
			},
			setupDeclEntry: decls.NewDeclEntry(),
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
				pkgs: []pkgName{"myapp"},
				structs: map[pkgName][]structSet{
					"myapp": {
						{name: "Client", description: "A client for API calls", fields: []string{"Timeout", "BaseURL"}},
						{name: "Config", description: "A configuration structure", fields: []string{"Name", "Value"}},
					},
				},
			},
			setupDeclEntry: decls.NewDeclEntry(),
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
				pkgs: []pkgName{"myapp"},
				structs: map[pkgName][]structSet{
					"myapp": {
						{name: "Client", description: "A client for API calls", fields: []string{"Timeout", "BaseURL"}},
						{name: "Config", description: "A configuration structure", fields: []string{"Name", "Value"}},
					},
				},
			},
			setupDeclEntry: decls.NewDeclEntry(),
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
				pkgs: []pkgName{"myapp"},
				structs: map[pkgName][]structSet{
					"myapp": {
						{name: "Client", description: "A client for API calls", fields: []string{"Timeout", "BaseURL"}},
						{name: "Config", description: "A configuration structure", fields: []string{"Name", "Value"}},
					},
				},
			},
			setupDeclEntry: decls.NewDeclEntry(),
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
				pkgs: []pkgName{"myapp"},
				methods: map[pkgName][]methodSet{
					"myapp": {
						{
							receiverTypeName: "Client",
							name:             "Do",
							description:      "Do executes a request",
							returnTypeNames:  []string{"Response", "error"},
						},
						{
							receiverTypeName: "Client",
							name:             "Get",
							description:      "Get sends a GET request",
							returnTypeNames:  []string{"Response", "error"},
						},
					},
				},
			},
			setupDeclEntry: func() *decls.DeclEntry {
				de := decls.NewDeclEntry()
				if err := de.Register("client := myapp.Client{}"); err != nil {
					t.Fatalf("failed to register client variable: %v", err)
				}
				return de
			}(),
			expected: []prompt.Suggest{
				{
					Text:        "client.Do()",
					DisplayText: "Do()",
					Description: "Method: Do executes a request",
				},
				{
					Text:        "client.Get()",
					DisplayText: "Get()",
					Description: "Method: Get sends a GET request",
				},
			},
		},
		{
			name:      "Complete methods of variable declared from another variable",
			inputText: "stream.",
			setupCandidates: &candidates{
				pkgs: []pkgName{"myapp"},
				vars: map[pkgName][]varSet{
					"myapp": {
						{name: "StdOut", description: "Standard output", typeName: "Stream", typePkgName: "myapp"},
					},
				},
				methods: map[pkgName][]methodSet{
					"myapp": {
						{
							receiverTypeName: "Stream",
							name:             "Write",
							description:      "Write writes data to the stream",
							returnTypeNames:  []string{"int", "error"},
						},
						{
							receiverTypeName: "Stream",
							name:             "Close",
							description:      "Close closes the stream",
							returnTypeNames:  []string{"error"},
						},
					},
				},
			},
			setupDeclEntry: func() *decls.DeclEntry {
				de := decls.NewDeclEntry()
				if err := de.Register("stream := myapp.StdOut"); err != nil {
					t.Fatalf("failed to register stream variable: %v", err)
				}
				return de
			}(),
			expected: []prompt.Suggest{
				{
					Text:        "stream.Write()",
					DisplayText: "Write()",
					Description: "Method: Write writes data to the stream",
				},
				{
					Text:        "stream.Close()",
					DisplayText: "Close()",
					Description: "Method: Close closes the stream",
				},
			},
		},
		{
			name:      "Complete methods of variable declared from function return",
			inputText: "response.",
			setupCandidates: &candidates{
				pkgs: []pkgName{"myapp"},
				funcs: map[pkgName][]funcSet{
					"myapp": {
						{
							name:               "FetchData",
							description:        "FetchData retrieves data from a source",
							returnTypeNames:    []string{"Response", "error"},
							returnTypePkgNames: []string{"myapp", ""},
						},
					},
				},
				methods: map[pkgName][]methodSet{
					"myapp": {
						{
							receiverTypeName:   "Response",
							name:               "GetContent",
							description:        "GetContent returns the response content",
							returnTypeNames:    []string{"Content"},
							returnTypePkgNames: []string{"myapp"},
						},
					},
				},
			},
			setupDeclEntry: func() *decls.DeclEntry {
				de := decls.NewDeclEntry()
				if err := de.Register("response, _ := myapp.FetchData()"); err != nil {
					t.Fatalf("failed to register response variable: %v", err)
				}
				return de
			}(),
			expected: []prompt.Suggest{
				{
					Text:        "response.GetContent()",
					DisplayText: "GetContent()",
					Description: "Method: GetContent returns the response content",
				},
			},
		},
		{
			name:      "Complete methods of variable declared from method return",
			inputText: "content.",
			setupCandidates: &candidates{
				pkgs: []pkgName{"myapp"},
				funcs: map[pkgName][]funcSet{
					"myapp": {
						{
							name:               "FetchData",
							description:        "FetchData retrieves data from a source",
							returnTypeNames:    []string{"Response", "error"},
							returnTypePkgNames: []string{"myapp", ""},
						},
					},
				},
				methods: map[pkgName][]methodSet{
					"myapp": {
						{
							receiverTypeName:   "Response",
							name:               "GetContent",
							description:        "GetContent returns the response content",
							returnTypeNames:    []string{"Content"},
							returnTypePkgNames: []string{"myapp"},
						},
						{
							receiverTypeName:   "Content",
							name:               "Read",
							description:        "Read reads data from the content",
							returnTypeNames:    []string{"int", "error"},
							returnTypePkgNames: []string{"", ""},
						},
						{
							receiverTypeName:   "Content",
							name:               "Type",
							description:        "Type returns the content type",
							returnTypeNames:    []string{"string"},
							returnTypePkgNames: []string{""},
						},
					},
				},
			},
			setupDeclEntry: func() *decls.DeclEntry {
				de := decls.NewDeclEntry()
				// まず response 変数を作成
				if err := de.Register("response, _ := myapp.FetchData()"); err != nil {
					t.Fatalf("failed to register response variable: %v", err)
				}
				// 次に content 変数を response.GetContent() から作成
				if err := de.Register("content := response.GetContent()"); err != nil {
					t.Fatalf("failed to register content variable: %v", err)
				}
				return de
			}(),
			expected: []prompt.Suggest{
				{
					Text:        "content.Read()",
					DisplayText: "Read()",
					Description: "Method: Read reads data from the content",
				},
				{
					Text:        "content.Type()",
					DisplayText: "Type()",
					Description: "Method: Type returns the content type",
				},
			},
		},
		{
			name:      "Do not complete private symbols",
			inputText: "myapp.",
			setupCandidates: &candidates{
				pkgs: []pkgName{"myapp"},
				funcs: map[pkgName][]funcSet{
					"myapp": {
						{name: "Print", description: "Print outputs a message"},
						{name: "printf", description: "Internal printing function"},
					},
				},
				vars: map[pkgName][]varSet{
					"myapp": {
						{name: "Version", description: "Package version"},
						{name: "privateVar", description: "Internal variable"},
					},
				},
			},
			setupDeclEntry: decls.NewDeclEntry(),
			expected: []prompt.Suggest{
				{
					Text:        "myapp.Print()",
					DisplayText: "Print()",
					Description: "Function: Print outputs a message",
				},
				{
					Text:        "myapp.Version",
					DisplayText: "Version",
					Description: "Variable: Package version",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completer := NewCompleter(tt.setupCandidates, tt.setupDeclEntry)
			doc := prompt.Document{
				Text: tt.inputText,
			}

			got := completer.Complete(doc)

			// 結果を比較（順序は考慮しない）
			cmpOpts := []cmp.Option{
				cmp.AllowUnexported(prompt.Suggest{}),
				// 順序を無視して比較
				cmp.Transformer("SortSuggestions", func(in []prompt.Suggest) []prompt.Suggest {
					result := append([]prompt.Suggest{}, in...) // コピーを作成
					return result
				}),
			}

			if diff := cmp.Diff(tt.expected, got, cmpOpts...); diff != "" {
				t.Errorf("Complete() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
