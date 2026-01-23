package decl_registry

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kakkky/gonsole/types"
)

func TestDeclRegistry_Register(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Decl
		wantErr  bool
	}{
		{
			name:  "selector expression assignment",
			input: "a := pkg.Var",
			expected: []Decl{
				{
					name: "a",
					rhs: declRHS{
						name:    "Var",
						kind:    DeclRHSKindVar,
						pkgName: "pkg",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "composite literal assignment",
			input: "s := pkg.Struct{}",
			expected: []Decl{
				{
					name: "s",
					rhs: declRHS{
						name:    "Struct",
						kind:    DeclRHSKindStruct,
						pkgName: "pkg",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "pointer to struct assignment",
			input: "p := &pkg.Struct{}",
			expected: []Decl{
				{
					name: "p",
					rhs: declRHS{
						name:    "Struct",
						kind:    DeclRHSKindStruct,
						pkgName: "pkg",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "function call assignment",
			input: "f := pkg.Func()",
			expected: []Decl{
				{
					name:        "f",
					isReturnVal: true,
					returnedIdx: 0,
					rhs: declRHS{
						name:    "Func",
						kind:    DeclRHSKindFunc,
						pkgName: "pkg",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "multiple return values from function",
			input: "a, b := pkg.Func()",
			expected: []Decl{
				{
					name:        "a",
					isReturnVal: true,
					returnedIdx: 0,
					rhs: declRHS{
						name:    "Func",
						kind:    DeclRHSKindFunc,
						pkgName: "pkg",
					},
				},
				{
					name:        "b",
					isReturnVal: true,
					returnedIdx: 1,
					rhs: declRHS{
						name:    "Func",
						kind:    DeclRHSKindFunc,
						pkgName: "pkg",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "var declaration with selector",
			input: "var v = pkg.Var",
			expected: []Decl{
				{
					name: "v",
					rhs: declRHS{
						name:    "Var",
						kind:    DeclRHSKindVar,
						pkgName: "pkg",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "var declaration with composite literal",
			input: "var s = pkg.Struct{}",
			expected: []Decl{
				{
					name: "s",
					rhs: declRHS{
						name:    "Struct",
						kind:    DeclRHSKindStruct,
						pkgName: "pkg",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "var declaration with pointer to struct",
			input: "var p = &pkg.Struct{}",
			expected: []Decl{
				{
					name: "p",
					rhs: declRHS{
						name:    "Struct",
						kind:    DeclRHSKindStruct,
						pkgName: "pkg",
					},
				},
			},
			wantErr: false,
		},
		{
			name:  "var declaration with function call",
			input: "var f = pkg.Func()",
			expected: []Decl{
				{
					name:        "f",
					isReturnVal: true,
					returnedIdx: 0,
					rhs: declRHS{
						name:    "Func",
						kind:    DeclRHSKindFunc,
						pkgName: "pkg",
					},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sut := NewRegistry()
			err := sut.Register(tt.input)

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
			if diff := cmp.Diff(tt.expected, sut.Decls(), cmp.AllowUnexported(Decl{}, declRHS{})); diff != "" {
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
					name: "var1",
					rhs: declRHS{
						name:    "Var",
						kind:    DeclRHSKindVar,
						pkgName: "pkg1",
					},
				},
				{
					name: "struct1",
					rhs: declRHS{
						name:    "Struct",
						kind:    DeclRHSKindStruct,
						pkgName: "pkg2",
					},
				},
			},
			checkName:     "var1",
			expectedFound: true,
		},
		{
			name: "found registered declaration (from multiple declarations)",
			existingDecls: []Decl{
				{
					name: "var1",
					rhs: declRHS{
						name:    "Var",
						kind:    DeclRHSKindVar,
						pkgName: "pkg1",
					},
				},
				{
					name: "struct1",
					rhs: declRHS{
						name:    "Struct",
						kind:    DeclRHSKindStruct,
						pkgName: "pkg2",
					},
				},
			},
			checkName:     "struct1",
			expectedFound: true,
		},
		{
			name: "not found registered declaration",
			existingDecls: []Decl{
				{
					name: "var1",
					rhs: declRHS{
						name:    "Var",
						kind:    DeclRHSKindVar,
						pkgName: "pkg1",
					},
				},
				{
					name: "struct1",
					rhs: declRHS{
						name:    "Struct",
						kind:    DeclRHSKindStruct,
						pkgName: "pkg2",
					},
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
				decls: tt.existingDecls,
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
