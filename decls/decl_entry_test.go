package decls

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDeclEntry_Register(t *testing.T) {
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
					pkg:  "pkg",
					name: "a",
					rhs: declRhs{
						declVar: declVar{name: "Var"},
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
					pkg:  "pkg",
					name: "s",
					rhs: declRhs{
						declStruct: declStruct{typeName: "Struct"},
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
					pkg:  "pkg",
					name: "p",
					rhs: declRhs{
						declStruct: declStruct{typeName: "Struct"},
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
					pkg:  "pkg",
					name: "f",
					rhs: declRhs{
						declFunc: declFunc{name: "Func", order: 0},
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
					pkg:  "pkg",
					name: "a",
					rhs: declRhs{
						declFunc: declFunc{name: "Func", order: 0},
					},
				},
				{
					pkg:  "pkg",
					name: "b",
					rhs: declRhs{
						declFunc: declFunc{name: "Func", order: 1},
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
					pkg:  "pkg",
					name: "v",
					rhs: declRhs{
						declVar: declVar{name: "Var"},
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
					pkg:  "pkg",
					name: "s",
					rhs: declRhs{
						declStruct: declStruct{typeName: "Struct"},
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
					pkg:  "pkg",
					name: "p",
					rhs: declRhs{
						declStruct: declStruct{typeName: "Struct"},
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
					pkg:  "pkg",
					name: "f",
					rhs: declRhs{
						declFunc: declFunc{name: "Func", order: 0},
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
			sut := NewDeclEntry()
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
			if diff := cmp.Diff(tt.expected, sut.Decls(), cmp.AllowUnexported(Decl{}, declVar{}, declFunc{}, declMethod{}, declStruct{}, declRhs{})); diff != "" {
				t.Errorf("Register() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestDeclEntry_IsRegisteredDecl(t *testing.T) {
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
					pkg:  "pkg1",
					name: "var1",
					rhs: declRhs{
						declVar: declVar{name: "Var"},
					},
				},
				{
					pkg:  "pkg2",
					name: "struct1",
					rhs: declRhs{
						declStruct: declStruct{typeName: "Struct"},
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
					pkg:  "pkg1",
					name: "var1",
					rhs: declRhs{
						declVar: declVar{name: "Var"},
					},
				},
				{
					pkg:  "pkg2",
					name: "struct1",
					rhs: declRhs{
						declStruct: declStruct{typeName: "Struct"},
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
					pkg:  "pkg1",
					name: "var1",
					rhs: declRhs{
						declVar: declVar{name: "Var"},
					},
				},
				{
					pkg:  "pkg2",
					name: "struct1",
					rhs: declRhs{
						declStruct: declStruct{typeName: "Struct"},
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
			// DeclEntryの準備
			de := &DeclEntry{
				decls: &tt.existingDecls,
			}

			// テスト対象メソッドの実行
			result := de.IsRegisteredDecl(tt.checkName)

			// 結果の検証
			if result != tt.expectedFound {
				t.Errorf("IsRegisteredDecl(%s) = %v, want %v",
					tt.checkName, result, tt.expectedFound)
			}
		})
	}
}
