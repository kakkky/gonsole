package declregistry

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kakkky/gonsole/types"
)

func TestDeclRegistry_Register(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		importPath    types.ImportPath
		expected      []Decl
		existingDecls []Decl
		wantErr       bool
	}{
		{
			name:       "selector expression assignment",
			input:      "a := testdata.Var",
			importPath: `"github.com/kakkky/gonsole/declregistry/testdata"`,
			expected: []Decl{
				{
					Name:        "a",
					TypeName:    "int",
					TypePkgName: "",
				},
			},
			wantErr: false,
		},
		{
			name:       "composite literal assignment",
			input:      "s := testdata.Struct{}",
			importPath: `"github.com/kakkky/gonsole/declregistry/testdata"`,
			expected: []Decl{
				{
					Name:        "s",
					TypeName:    "Struct",
					TypePkgName: "testdata",
				},
			},
			wantErr: false,
		},
		{
			name:       "pointer to struct assignment",
			input:      "p := &testdata.Struct{}",
			importPath: `"github.com/kakkky/gonsole/declregistry/testdata"`,
			expected: []Decl{
				{
					Name:        "p",
					Pointered:   true,
					TypeName:    "Struct",
					TypePkgName: "testdata",
				},
			},
			wantErr: false,
		},
		{
			name:       "function call assignment",
			input:      "f := testdata.Func()",
			importPath: `"github.com/kakkky/gonsole/declregistry/testdata"`,
			expected: []Decl{
				{
					Name:        "f",
					TypeName:    "int",
					TypePkgName: "",
				},
			},
			wantErr: false,
		},
		{
			name:       "multiple return values from function",
			input:      "a, b := testdata.MultiReturn()",
			importPath: `"github.com/kakkky/gonsole/declregistry/testdata"`,
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
			wantErr: false,
		},
		{
			name:       "var declaration with selector",
			input:      "var v = testdata.Var",
			importPath: `"github.com/kakkky/gonsole/declregistry/testdata"`,
			expected: []Decl{
				{
					Name:        "v",
					TypeName:    "int",
					TypePkgName: "",
				},
			},
			wantErr: false,
		},
		{
			name:       "var declaration with composite literal",
			input:      "var s = testdata.Struct{}",
			importPath: `"github.com/kakkky/gonsole/declregistry/testdata"`,
			expected: []Decl{
				{
					Name:        "s",
					TypeName:    "Struct",
					TypePkgName: "testdata",
				},
			},
			wantErr: false,
		},
		{
			name:       "var declaration with pointer to struct",
			input:      "var p = &testdata.Struct{}",
			importPath: `"github.com/kakkky/gonsole/declregistry/testdata"`,
			expected: []Decl{
				{
					Name:        "p",
					Pointered:   true,
					TypeName:    "Struct",
					TypePkgName: "testdata",
				},
			},
			wantErr: false,
		},
		{
			name:       "var declaration with function call",
			input:      "var f = testdata.Func()",
			importPath: `"github.com/kakkky/gonsole/declregistry/testdata"`,
			expected: []Decl{
				{
					Name:        "f",
					TypeName:    "int",
					TypePkgName: "",
				},
			},
			wantErr: false,
		},
		{
			name:     "invalid syntax",
			input:    "a := 1 +",
			expected: []Decl{},
			wantErr:  true,
		},
		{
			name:       "method assignment",
			input:      "b := a.Method1()",
			importPath: `"github.com/kakkky/gonsole/declregistry/testdata"`,
			existingDecls: []Decl{
				{
					Name:        "a",
					TypeName:    "Struct",
					TypePkgName: "testdata",
				},
			},
			expected: []Decl{
				{
					Name:        "b",
					TypeName:    "Struct",
					TypePkgName: "testdata",
				},
			},
			wantErr: false,
		},
		{
			name:       "method chain assignment",
			input:      "b := a.Method1().Method2()",
			importPath: `"github.com/kakkky/gonsole/declregistry/testdata"`,
			existingDecls: []Decl{
				{
					Name:        "a",
					TypeName:    "Struct",
					TypePkgName: "testdata",
				},
			},
			expected: []Decl{
				{
					Name:        "a",
					TypeName:    "Struct",
					TypePkgName: "testdata",
				},
				{
					Name:        "b",
					TypeName:    "string",
					TypePkgName: "",
				},
			},
			wantErr: false,
		},
		{
			name:       "var declaration with method chain",
			input:      "var b = a.Method1().Method2()",
			importPath: `"github.com/kakkky/gonsole/declregistry/testdata"`,
			existingDecls: []Decl{
				{
					Name:        "a",
					TypeName:    "Struct",
					TypePkgName: "testdata",
				},
			},
			expected: []Decl{
				{
					Name:        "a",
					TypeName:    "Struct",
					TypePkgName: "testdata",
				},
				{
					Name:        "b",
					TypeName:    "string",
					TypePkgName: "",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sut := NewRegistry()
			if len(tt.existingDecls) > 0 {
				sut.Decls = append(sut.Decls, tt.existingDecls...)
			}

			err := sut.Register(tt.input, tt.importPath)

			// エラーステータスの確認
			if (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// エラーが期待される場合は結果チェック不要
			if tt.wantErr {
				return
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
