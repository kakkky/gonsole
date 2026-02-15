package declregistry

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kakkky/gonsole/types"
)

func TestDeclRegistry_Register(t *testing.T) {
	tests := []struct {
		name                string
		existingTmpFileName string
		existingDecls       []Decl
		expected            []Decl
	}{
		{
			name:                "selector expression assignment",
			existingTmpFileName: "./testdata/selector_expression_assignment/00000_gonsole_tmp.go",
			expected: []Decl{
				{
					Name:        "a",
					TypeName:    "int",
					TypePkgName: "",
				},
			},
		},
		{
			name:                "composite literal assignment",
			existingTmpFileName: "./testdata/composite_literal_assignment/00000_gonsole_tmp.go",
			expected: []Decl{
				{
					Name:        "s",
					TypeName:    "Struct",
					TypePkgName: "sample",
				},
			},
		},
		{
			name:                "pointer to struct assignment",
			existingTmpFileName: "./testdata/pointer_to_struct_assignment/00000_gonsole_tmp.go",
			expected: []Decl{
				{
					Name:        "p",
					Pointered:   true,
					TypeName:    "Struct",
					TypePkgName: "sample",
				},
			},
		},
		{
			name:                "function call assignment",
			existingTmpFileName: "./testdata/function_call_assignment/00000_gonsole_tmp.go",
			expected: []Decl{
				{
					Name:        "f",
					TypeName:    "int",
					TypePkgName: "",
				},
			},
		},
		{
			name:                "multiple return values from function",
			existingTmpFileName: "./testdata/multiple_return_values_from_function/00000_gonsole_tmp.go",
			expected: []Decl{
				{
					Name:        "a",
					TypeName:    "int",
					TypePkgName: "",
				},
				{
					Name:        "b",
					TypeName:    "string",
					TypePkgName: "",
				},
			},
		},
		{
			name:                "var declaration with selector",
			existingTmpFileName: "./testdata/var_declaration_with_selector/00000_gonsole_tmp.go",
			expected: []Decl{
				{
					Name:        "v",
					TypeName:    "int",
					TypePkgName: "",
				},
			},
		},
		{
			name:                "var declaration with composite literal",
			existingTmpFileName: "./testdata/var_declaration_with_composite_literal/00000_gonsole_tmp.go",
			expected: []Decl{
				{
					Name:        "s",
					TypeName:    "Struct",
					TypePkgName: "sample",
				},
			},
		},
		{
			name:                "var declaration with pointer to struct",
			existingTmpFileName: "./testdata/var_declaration_with_pointer_to_struct/00000_gonsole_tmp.go",
			expected: []Decl{
				{
					Name:        "p",
					Pointered:   true,
					TypeName:    "Struct",
					TypePkgName: "sample",
				},
			},
		},
		{
			name:                "var declaration with function call",
			existingTmpFileName: "./testdata/var_declaration_with_function_call/00000_gonsole_tmp.go",
			expected: []Decl{
				{
					Name:        "f",
					TypeName:    "int",
					TypePkgName: "",
				},
			},
		},
		{
			name:                "method assignment",
			existingTmpFileName: "./testdata/method_assignment/00000_gonsole_tmp.go",
			existingDecls: []Decl{
				{
					Name:        "a",
					TypeName:    "Struct",
					TypePkgName: "sample",
				},
			},
			expected: []Decl{
				{
					Name:        "a",
					TypeName:    "Struct",
					TypePkgName: "sample",
				},
				{
					Name:        "b",
					TypeName:    "Struct",
					TypePkgName: "sample",
				},
			},
		},
		{
			name:                "method chain assignment",
			existingTmpFileName: "./testdata/method_chain_assignment/00000_gonsole_tmp.go",
			existingDecls: []Decl{
				{
					Name:        "a",
					TypeName:    "Struct",
					TypePkgName: "sample",
				},
			},
			expected: []Decl{
				{
					Name:        "a",
					TypeName:    "Struct",
					TypePkgName: "sample",
				},
				{
					Name:        "b",
					TypeName:    "string",
					TypePkgName: "",
				},
			},
		},
		{
			name:                "var declaration with method chain",
			existingTmpFileName: "./testdata/var_declaration_with_method_chain/00000_gonsole_tmp.go",
			existingDecls: []Decl{
				{
					Name:        "a",
					TypeName:    "Struct",
					TypePkgName: "sample",
				},
			},
			expected: []Decl{
				{
					Name:        "a",
					TypeName:    "Struct",
					TypePkgName: "sample",
				},
				{
					Name:        "b",
					TypeName:    "string",
					TypePkgName: "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sut := NewRegistry()

			if tt.existingDecls != nil {
				sut.Decls = tt.existingDecls
			}

			err := sut.Register(tt.existingTmpFileName)

			// エラーステータスの確認
			if err != nil {
				t.Fatalf("Register() returned an error: %v", err)
			}

			// cmp.Diffを使って結果を比較
			if diff := cmp.Diff(tt.expected, sut.Decls); diff != "" {
				t.Errorf("Register() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestRegistry_IsRegisteredDecl(t *testing.T) {
	tests := []struct {
		name          string
		existingDecls []Decl
		checkName     string
		expectedFound bool
	}{
		{
			name: "found registered declaration",
			existingDecls: []Decl{
				{
					Name:        "var1",
					TypeName:    "int",
					TypePkgName: "",
				},
				{
					Name:        "struct1",
					TypeName:    "Struct",
					TypePkgName: "declregistry",
				},
			},
			checkName:     "var1",
			expectedFound: true,
		},
		{
			name: "found registered declaration (from multiple declarations)",
			existingDecls: []Decl{
				{
					Name:        "var1",
					TypeName:    "int",
					TypePkgName: "",
				},
				{
					Name:        "struct1",
					TypeName:    "Struct",
					TypePkgName: "declregistry",
				},
			},
			checkName:     "struct1",
			expectedFound: true,
		},
		{
			name: "not found registered declaration",
			existingDecls: []Decl{
				{
					Name:        "var1",
					TypeName:    "int",
					TypePkgName: "",
				},
			},
			checkName:     "unknown",
			expectedFound: false,
		},
		{
			name:          "not found registered declaration (empty list)",
			existingDecls: []Decl{},
			checkName:     "var1",
			expectedFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Registryの準備
			dr := &DeclRegistry{
				Decls: tt.existingDecls,
			}

			// テスト対象メソッドの実行
			result := dr.IsRegisteredDecl(types.DeclName(tt.checkName))

			// 結果の検証
			if result != tt.expectedFound {
				t.Errorf("IsRegisteredDecl(%s) = %v, want %v",
					tt.checkName, result, tt.expectedFound)
			}
		})
	}
}
