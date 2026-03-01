package executor

import (
	"bytes"
	"go/ast"
	"go/token"
	"os"
	"os/exec"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kakkky/gonsole/declregistry"
	"github.com/kakkky/gonsole/symbols"
	"github.com/kakkky/gonsole/types"
	gomock "go.uber.org/mock/gomock"
)

// MEMO: session srcのastが期待通りに組み立てられているかまでがExecutorの責務なので、実行結果やファイル生成はここでは検証しない。
func TestExecutor_Execute(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		setupDeclRegistry  func(*declregistry.DeclRegistry) // 必要に応じてDeclRegistryの初期状態をセットアップする
		setupMocks         func(*Mockfiler, *Mockcommander, *MockimportPathResolver)
		setupSymbolIndex   *symbols.SymbolIndex
		expectedSessionSrc *ast.File
	}{
		{
			name:  "empty input",
			input: "",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				// 初期状態のセットアップが不要な場合は空の関数を指定
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				// 何も呼ばれないことを期待
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{},
						},
					},
				},
			},
		},
		{
			name:  "define variable from basic lit",
			input: "var x = 10",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				// 初期状態のセットアップが不要な場合は空の関数を指定
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", declFlush(), []byte{}, nil)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.DeclStmt{
									Decl: &ast.GenDecl{
										Tok: token.VAR,
										Specs: []ast.Spec{
											&ast.ValueSpec{
												Names: []*ast.Ident{
													{Name: "x"},
												},
												Values: []ast.Expr{
													&ast.BasicLit{
														Kind:  token.INT,
														Value: "10",
													},
												},
											},
										},
									},
								},
								&ast.AssignStmt{
									Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
									Tok: token.ASSIGN,
									Rhs: []ast.Expr{&ast.Ident{Name: "x"}},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "define variable from variable of the package",
			input: "var x = pkg.Variable",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				// 初期状態のセットアップが不要な場合は空の関数を指定
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", declFlush(), []byte{}, nil,
					resolveExpect{types.PkgName("pkg"), types.ImportPath(`"github.com/test/pkg"`)},
				)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok: token.IMPORT,
						Specs: []ast.Spec{
							&ast.ImportSpec{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/test/pkg"`,
								},
							},
						},
					},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.DeclStmt{
									Decl: &ast.GenDecl{
										Tok: token.VAR,
										Specs: []ast.Spec{
											&ast.ValueSpec{
												Names: []*ast.Ident{
													{Name: "x"},
												},
												Values: []ast.Expr{
													&ast.SelectorExpr{
														X:   &ast.Ident{Name: "pkg"},
														Sel: &ast.Ident{Name: "Variable"},
													},
												},
											},
										},
									},
								},
								&ast.AssignStmt{
									Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
									Tok: token.ASSIGN,
									Rhs: []ast.Expr{&ast.Ident{Name: "x"}},
								},
							},
						},
					},
				},
				Imports: []*ast.ImportSpec{
					{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"github.com/test/pkg"`,
						},
					},
				},
			},
		},
		{
			name:  "define variable from function's return value of the package",
			input: "var x = pkg.Function()",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				// 初期状態のセットアップが不要な場合は空の関数を指定
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", declFlush(), []byte{}, nil,
					resolveExpect{types.PkgName("pkg"), types.ImportPath(`"github.com/test/pkg"`)},
				)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok: token.IMPORT,
						Specs: []ast.Spec{
							&ast.ImportSpec{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/test/pkg"`,
								},
							},
						},
					},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.DeclStmt{
									Decl: &ast.GenDecl{
										Tok: token.VAR,
										Specs: []ast.Spec{
											&ast.ValueSpec{
												Names: []*ast.Ident{
													{Name: "x"},
												},
												Values: []ast.Expr{
													&ast.CallExpr{
														Fun: &ast.SelectorExpr{
															X:   &ast.Ident{Name: "pkg"},
															Sel: &ast.Ident{Name: "Function"},
														},
														Args: nil,
													},
												},
											},
										},
									},
								},
								&ast.AssignStmt{
									Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
									Tok: token.ASSIGN,
									Rhs: []ast.Expr{&ast.Ident{Name: "x"}},
								},
							},
						},
					},
				},
				Imports: []*ast.ImportSpec{
					{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"github.com/test/pkg"`,
						},
					},
				},
			},
		},
		{
			name:  "define variable from function's return multiple values of the package",
			input: "var x, y = pkg.Function()",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				// 初期状態のセットアップが不要な場合は空の関数を指定
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", declFlush(), []byte{}, nil,
					resolveExpect{types.PkgName("pkg"), types.ImportPath(`"github.com/test/pkg"`)},
				)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok: token.IMPORT,
						Specs: []ast.Spec{
							&ast.ImportSpec{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/test/pkg"`,
								},
							},
						},
					},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.DeclStmt{
									Decl: &ast.GenDecl{
										Tok: token.VAR,
										Specs: []ast.Spec{
											&ast.ValueSpec{
												Names: []*ast.Ident{
													{Name: "x"},
													{Name: "y"},
												},
												Values: []ast.Expr{
													&ast.CallExpr{
														Fun: &ast.SelectorExpr{
															X:   &ast.Ident{Name: "pkg"},
															Sel: &ast.Ident{Name: "Function"},
														},
														Args: nil,
													},
												},
											},
										},
									},
								},
								&ast.AssignStmt{
									Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
									Tok: token.ASSIGN,
									Rhs: []ast.Expr{&ast.Ident{Name: "x"}},
								},
								&ast.AssignStmt{
									Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
									Tok: token.ASSIGN,
									Rhs: []ast.Expr{&ast.Ident{Name: "y"}},
								},
							},
						},
					},
				},
				Imports: []*ast.ImportSpec{
					{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"github.com/test/pkg"`,
						},
					},
				},
			},
		},
		{
			name:  "define variable from method's return value",
			input: "var x = obj.Method()",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				declRegistry.Decls = append(declRegistry.Decls, declregistry.Decl{
					Name:        "obj",
					TypeName:    "Object",
					TypePkgName: "pkg",
				})
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", declFlush(), []byte{}, nil)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.DeclStmt{
									Decl: &ast.GenDecl{
										Tok: token.VAR,
										Specs: []ast.Spec{
											&ast.ValueSpec{
												Names: []*ast.Ident{
													{Name: "x"},
												},
												Values: []ast.Expr{
													&ast.CallExpr{
														Fun: &ast.SelectorExpr{
															X:   &ast.Ident{Name: "obj"},
															Sel: &ast.Ident{Name: "Method"},
														},
														Args: nil,
													},
												},
											},
										},
									},
								},
								&ast.AssignStmt{
									Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
									Tok: token.ASSIGN,
									Rhs: []ast.Expr{&ast.Ident{Name: "x"}},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "define variable from method's return multiple values",
			input: "var x, y = obj.Method()",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				declRegistry.Decls = append(declRegistry.Decls, declregistry.Decl{
					Name:        "obj",
					TypeName:    "Object",
					TypePkgName: "pkg",
				})
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", declFlush(), []byte{}, nil)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.DeclStmt{
									Decl: &ast.GenDecl{
										Tok: token.VAR,
										Specs: []ast.Spec{
											&ast.ValueSpec{
												Names: []*ast.Ident{
													{Name: "x"},
													{Name: "y"},
												},
												Values: []ast.Expr{
													&ast.CallExpr{
														Fun: &ast.SelectorExpr{
															X:   &ast.Ident{Name: "obj"},
															Sel: &ast.Ident{Name: "Method"},
														},
														Args: nil,
													},
												},
											},
										},
									},
								},
								&ast.AssignStmt{
									Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
									Tok: token.ASSIGN,
									Rhs: []ast.Expr{&ast.Ident{Name: "x"}},
								},
								&ast.AssignStmt{
									Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
									Tok: token.ASSIGN,
									Rhs: []ast.Expr{&ast.Ident{Name: "y"}},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "define variable from method chain's return value",
			input: "var x = obj.Method1().Method2()",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				declRegistry.Decls = append(declRegistry.Decls, declregistry.Decl{
					Name:        "obj",
					TypeName:    "Object",
					TypePkgName: "pkg",
				})
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", declFlush(), []byte{}, nil)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.DeclStmt{
									Decl: &ast.GenDecl{
										Tok: token.VAR,
										Specs: []ast.Spec{
											&ast.ValueSpec{
												Names: []*ast.Ident{
													{Name: "x"},
												},
												Values: []ast.Expr{
													&ast.CallExpr{
														Fun: &ast.SelectorExpr{
															X: &ast.CallExpr{
																Fun: &ast.SelectorExpr{
																	X:   &ast.Ident{Name: "obj"},
																	Sel: &ast.Ident{Name: "Method1"},
																},
																Args: nil,
															},
															Sel: &ast.Ident{Name: "Method2"},
														},
														Args: nil,
													},
												},
											},
										},
									},
								},
								&ast.AssignStmt{
									Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
									Tok: token.ASSIGN,
									Rhs: []ast.Expr{&ast.Ident{Name: "x"}},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "define variable from struct of the package",
			input: "var x = pkg.Struct{field: y}",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				// 初期状態のセットアップが不要な場合は空の関数を指定
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", declFlush(), []byte{}, nil,
					resolveExpect{types.PkgName("pkg"), types.ImportPath(`"github.com/test/pkg"`)},
				)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok: token.IMPORT,
						Specs: []ast.Spec{
							&ast.ImportSpec{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/test/pkg"`,
								},
							},
						},
					},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.DeclStmt{
									Decl: &ast.GenDecl{
										Tok: token.VAR,
										Specs: []ast.Spec{
											&ast.ValueSpec{
												Names: []*ast.Ident{
													{Name: "x"},
												},
												Values: []ast.Expr{
													&ast.CompositeLit{
														Type: &ast.SelectorExpr{
															X:   &ast.Ident{Name: "pkg"},
															Sel: &ast.Ident{Name: "Struct"},
														},
														Elts: []ast.Expr{
															&ast.KeyValueExpr{
																Key:   &ast.Ident{Name: "field"},
																Value: &ast.Ident{Name: "y"},
															},
														},
													},
												},
											},
										},
									},
								},
								&ast.AssignStmt{
									Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
									Tok: token.ASSIGN,
									Rhs: []ast.Expr{&ast.Ident{Name: "x"}},
								},
							},
						},
					},
				},
				Imports: []*ast.ImportSpec{
					{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"github.com/test/pkg"`,
						},
					},
				},
			},
		},
		{
			name:  "define variable from pointer to struct of the package",
			input: "var x = &pkg.Struct{field: y}",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				// 初期状態のセットアップが不要な場合は空の関数を指定
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", declFlush(), []byte{}, nil,
					resolveExpect{types.PkgName("pkg"), types.ImportPath(`"github.com/test/pkg"`)},
				)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok: token.IMPORT,
						Specs: []ast.Spec{
							&ast.ImportSpec{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/test/pkg"`,
								},
							},
						},
					},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.DeclStmt{
									Decl: &ast.GenDecl{
										Tok: token.VAR,
										Specs: []ast.Spec{
											&ast.ValueSpec{
												Names: []*ast.Ident{
													{Name: "x"},
												},
												Values: []ast.Expr{
													&ast.UnaryExpr{
														Op: token.AND,
														X: &ast.CompositeLit{
															Type: &ast.SelectorExpr{
																X:   &ast.Ident{Name: "pkg"},
																Sel: &ast.Ident{Name: "Struct"},
															},
															Elts: []ast.Expr{
																&ast.KeyValueExpr{
																	Key:   &ast.Ident{Name: "field"},
																	Value: &ast.Ident{Name: "y"},
																},
															},
														},
													},
												},
											},
										},
									},
								},
								&ast.AssignStmt{
									Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
									Tok: token.ASSIGN,
									Rhs: []ast.Expr{&ast.Ident{Name: "x"}},
								},
							},
						},
					},
				},
				Imports: []*ast.ImportSpec{
					{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"github.com/test/pkg"`,
						},
					},
				},
			},
		},
		{
			name:  "define variable from variable of the package as short variable declaration",
			input: "x := pkg.Variable",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				// 初期状態のセットアップが不要な場合は空の関数を指定
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", declFlush(), []byte{}, nil,
					resolveExpect{types.PkgName("pkg"), types.ImportPath(`"github.com/test/pkg"`)},
				)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok: token.IMPORT,
						Specs: []ast.Spec{
							&ast.ImportSpec{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/test/pkg"`,
								},
							},
						},
					},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.AssignStmt{
									Lhs: []ast.Expr{&ast.Ident{Name: "x"}},
									Tok: token.DEFINE,
									Rhs: []ast.Expr{
										&ast.SelectorExpr{
											X:   &ast.Ident{Name: "pkg"},
											Sel: &ast.Ident{Name: "Variable"},
										},
									},
								},
								&ast.AssignStmt{
									Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
									Tok: token.ASSIGN,
									Rhs: []ast.Expr{&ast.Ident{Name: "x"}},
								},
							},
						},
					},
				},
				Imports: []*ast.ImportSpec{
					{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"github.com/test/pkg"`,
						},
					},
				},
			},
		},
		{
			name:  "define variable from basic lit as short variable declaration",
			input: `x := "y"`,
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				// 初期状態のセットアップが不要な場合は空の関数を指定
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", declFlush(), []byte{}, nil)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.AssignStmt{
									Lhs: []ast.Expr{&ast.Ident{Name: "x"}},
									Tok: token.DEFINE,
									Rhs: []ast.Expr{
										&ast.BasicLit{
											Kind:  token.STRING,
											Value: `"y"`,
										},
									},
								},
								&ast.AssignStmt{
									Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
									Tok: token.ASSIGN,
									Rhs: []ast.Expr{&ast.Ident{Name: "x"}},
								},
							},
						},
					},
				},
			},
		},
		{
			name:  "define variable from function's return multiple values as short variable declaration",
			input: "x, y := pkg.Function()",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				// 初期状態のセットアップが不要な場合は空の関数を指定
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", declFlush(), []byte{}, nil,
					resolveExpect{types.PkgName("pkg"), types.ImportPath(`"github.com/test/pkg"`)},
				)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok: token.IMPORT,
						Specs: []ast.Spec{
							&ast.ImportSpec{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/test/pkg"`,
								},
							},
						},
					},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{
								&ast.AssignStmt{
									Lhs: []ast.Expr{
										&ast.Ident{Name: "x"},
										&ast.Ident{Name: "y"},
									},
									Tok: token.DEFINE,
									Rhs: []ast.Expr{
										&ast.CallExpr{
											Fun: &ast.SelectorExpr{
												X:   &ast.Ident{Name: "pkg"},
												Sel: &ast.Ident{Name: "Function"},
											},
											Args: nil,
										},
									},
								},
								&ast.AssignStmt{
									Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
									Tok: token.ASSIGN,
									Rhs: []ast.Expr{&ast.Ident{Name: "x"}},
								},
								&ast.AssignStmt{
									Lhs: []ast.Expr{&ast.Ident{Name: "_"}},
									Tok: token.ASSIGN,
									Rhs: []ast.Expr{&ast.Ident{Name: "y"}},
								},
							},
						},
					},
				},
				Imports: []*ast.ImportSpec{
					{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"github.com/test/pkg"`,
						},
					},
				},
			},
		},
		{
			name:  "call function of the package",
			input: "pkg.Function()",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				// 初期状態のセットアップが不要な場合は空の関数を指定
			},
			setupSymbolIndex: &symbols.SymbolIndex{
				Funcs: map[types.PkgName][]symbols.FuncSet{
					"pkg": {
						{
							Name:    "Function",
							Returns: []symbols.ReturnSet{{TypeName: "SomeType", TypePkgName: "pkg"}},
						},
					},
				},
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", exprFlush(func(sessionSrcAddedCallExpr *ast.File, targetFile *os.File, fset *token.FileSet) error {
					// この時のsessionSrcの状態を確認する
					// この途中経過の状態をテストするやり方が微妙なので、要改善
					expectedSessionSrc := &ast.File{
						Name: &ast.Ident{Name: "main"},
						Decls: []ast.Decl{
							&ast.GenDecl{
								Tok: token.IMPORT,
								Specs: []ast.Spec{
									&ast.ImportSpec{
										Path: &ast.BasicLit{
											Kind:  token.STRING,
											Value: `"github.com/test/pkg"`,
										},
									},
									&ast.ImportSpec{
										Path: &ast.BasicLit{
											Kind:  token.STRING,
											Value: `"github.com/k0kubun/pp/v3"`,
										},
									},
								},
							},
							&ast.FuncDecl{
								Name: &ast.Ident{Name: "main"},
								Type: &ast.FuncType{
									Params:  &ast.FieldList{List: nil},
									Results: nil,
								},
								Body: &ast.BlockStmt{
									List: []ast.Stmt{
										&ast.ExprStmt{
											X: &ast.CallExpr{
												Fun: &ast.FuncLit{
													Type: &ast.FuncType{
														Params:  &ast.FieldList{List: nil},
														Results: nil,
													},
													Body: &ast.BlockStmt{
														List: []ast.Stmt{
															&ast.AssignStmt{
																Lhs: []ast.Expr{&ast.Ident{Name: "ret0"}},
																Tok: token.DEFINE,
																Rhs: []ast.Expr{&ast.CallExpr{
																	Fun: &ast.SelectorExpr{
																		X:   &ast.Ident{Name: "pkg"},
																		Sel: &ast.Ident{Name: "Function"},
																	},
																}},
															},
															&ast.ExprStmt{
																X: &ast.CallExpr{
																	Fun:  &ast.Ident{Name: "pp.Println"},
																	Args: []ast.Expr{&ast.Ident{Name: "ret0"}},
																},
															},
														},
													},
												},
												Args: []ast.Expr{},
											},
										},
									},
								},
							},
						},
						Imports: []*ast.ImportSpec{
							{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/test/pkg"`,
								},
							},
							{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/k0kubun/pp/v3"`,
								},
							},
						},
					}

					cmpOpts := []cmp.Option{
						cmpopts.IgnoreFields(ast.Ident{}, "Obj", "NamePos"),
						cmpopts.IgnoreFields(ast.CallExpr{}, "Lparen", "Rparen"),
						cmpopts.IgnoreFields(ast.FuncLit{}, "Type"),
						cmpopts.IgnoreFields(ast.FuncType{}, "Func"),
						cmpopts.IgnoreFields(ast.FieldList{}, "Opening", "Closing"),
						cmpopts.IgnoreFields(ast.BlockStmt{}, "Lbrace", "Rbrace"),
						cmpopts.IgnoreFields(ast.AssignStmt{}, "TokPos"),
					}

					if diff := cmp.Diff(expectedSessionSrc, sessionSrcAddedCallExpr, cmpOpts...); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}

					return nil
				}),
					[]byte{}, nil,
					resolveExpect{types.PkgName("pkg"), types.ImportPath(`"github.com/test/pkg"`)},
					resolveExpect{types.PkgName("pp"), types.ImportPath(`"github.com/k0kubun/pp/v3"`)},
				)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok:   token.IMPORT,
						Specs: []ast.Spec{},
					},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{},
						},
					},
				},
				Imports: []*ast.ImportSpec{},
			},
		},
		{
			name:  "call function of the package when package is used by other declaration",
			input: "pkg.Function()",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				declRegistry.Decls = append(declRegistry.Decls, declregistry.Decl{
					Name:        "x",
					TypeName:    "Variable",
					TypePkgName: "pkg",
				}) // 事前にpkg自体も登録しておく
			},
			setupSymbolIndex: &symbols.SymbolIndex{
				Funcs: map[types.PkgName][]symbols.FuncSet{
					"pkg": {
						{
							Name:    "Function",
							Returns: []symbols.ReturnSet{{TypeName: "SomeType", TypePkgName: "pkg"}},
						},
					},
				},
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", exprFlush(func(sessionSrcAddedCallExpr *ast.File, targetFile *os.File, fset *token.FileSet) error {
					// この時のsessionSrcの状態を確認する
					// この途中経過の状態をテストするやり方が微妙なので、要改善
					expectedSessionSrc := &ast.File{
						Name: &ast.Ident{Name: "main"},
						Decls: []ast.Decl{
							&ast.GenDecl{
								Tok: token.IMPORT,
								Specs: []ast.Spec{
									&ast.ImportSpec{
										Path: &ast.BasicLit{
											Kind:  token.STRING,
											Value: `"github.com/test/pkg"`,
										},
									},
									&ast.ImportSpec{
										Path: &ast.BasicLit{
											Kind:  token.STRING,
											Value: `"github.com/k0kubun/pp/v3"`,
										},
									},
								},
							},
							&ast.FuncDecl{
								Name: &ast.Ident{Name: "main"},
								Type: &ast.FuncType{
									Params:  &ast.FieldList{List: nil},
									Results: nil,
								},
								Body: &ast.BlockStmt{
									List: []ast.Stmt{
										&ast.ExprStmt{
											X: &ast.CallExpr{
												Fun: &ast.FuncLit{
													Type: &ast.FuncType{
														Params:  &ast.FieldList{List: nil},
														Results: nil,
													},
													Body: &ast.BlockStmt{
														List: []ast.Stmt{
															&ast.AssignStmt{
																Lhs: []ast.Expr{&ast.Ident{Name: "ret0"}},
																Tok: token.DEFINE,
																Rhs: []ast.Expr{&ast.CallExpr{
																	Fun: &ast.SelectorExpr{
																		X:   &ast.Ident{Name: "pkg"},
																		Sel: &ast.Ident{Name: "Function"},
																	},
																}},
															},
															&ast.ExprStmt{
																X: &ast.CallExpr{
																	Fun:  &ast.Ident{Name: "pp.Println"},
																	Args: []ast.Expr{&ast.Ident{Name: "ret0"}},
																},
															},
														},
													},
												},
												Args: []ast.Expr{},
											},
										},
									},
								},
							},
						},
						Imports: []*ast.ImportSpec{
							{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/test/pkg"`,
								},
							},
							{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/k0kubun/pp/v3"`,
								},
							},
						},
					}

					cmpOpts := []cmp.Option{
						cmpopts.IgnoreFields(ast.Ident{}, "Obj", "NamePos"),
						cmpopts.IgnoreFields(ast.CallExpr{}, "Lparen", "Rparen"),
						cmpopts.IgnoreFields(ast.FuncLit{}, "Type"),
						cmpopts.IgnoreFields(ast.FuncType{}, "Func"),
						cmpopts.IgnoreFields(ast.FieldList{}, "Opening", "Closing"),
						cmpopts.IgnoreFields(ast.BlockStmt{}, "Lbrace", "Rbrace"),
						cmpopts.IgnoreFields(ast.AssignStmt{}, "TokPos"),
					}

					if diff := cmp.Diff(expectedSessionSrc, sessionSrcAddedCallExpr, cmpOpts...); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}

					return nil
				}),
					[]byte{}, nil,
					resolveExpect{types.PkgName("pkg"), types.ImportPath(`"github.com/test/pkg"`)},
					resolveExpect{types.PkgName("pp"), types.ImportPath(`"github.com/k0kubun/pp/v3"`)},
				)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok: token.IMPORT,
						Specs: []ast.Spec{
							&ast.ImportSpec{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/test/pkg"`,
								},
							},
						},
					},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{},
						},
					},
				},
				Imports: []*ast.ImportSpec{
					{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"github.com/test/pkg"`,
						},
					},
				},
			},
		},
		{
			name:  "call method",
			input: "obj.Method()",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				declRegistry.Decls = append(declRegistry.Decls, declregistry.Decl{
					Name:        "obj",
					TypeName:    "Object",
					TypePkgName: "pkg",
				}) // 事前にpkg自体も登録しておく
			},
			setupSymbolIndex: &symbols.SymbolIndex{
				Methods: map[types.PkgName][]symbols.MethodSet{
					"pkg": {
						{
							Name:             "Method",
							ReceiverTypeName: "Object",
							Returns:          []symbols.ReturnSet{{TypeName: "SomeType", TypePkgName: "pkg"}},
						},
					},
				},
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", exprFlush(func(sessionSrcAddedCallExpr *ast.File, targetFile *os.File, fset *token.FileSet) error {
					// この時のsessionSrcの状態を確認する
					// この途中経過の状態をテストするやり方が微妙なので、要改善
					//
					// 実際には、"var obj = pkg.NewObject()""に関連するASTも含まれるが、ここでは省略
					expectedSessionSrc := &ast.File{
						Name: &ast.Ident{Name: "main"},
						Decls: []ast.Decl{
							&ast.GenDecl{
								Tok: token.IMPORT,
								Specs: []ast.Spec{
									&ast.ImportSpec{
										Path: &ast.BasicLit{
											Kind:  token.STRING,
											Value: `"github.com/k0kubun/pp/v3"`,
										},
									},
								},
							},
							&ast.FuncDecl{
								Name: &ast.Ident{Name: "main"},
								Type: &ast.FuncType{
									Params:  &ast.FieldList{List: nil},
									Results: nil,
								},
								Body: &ast.BlockStmt{
									List: []ast.Stmt{
										&ast.ExprStmt{
											X: &ast.CallExpr{
												Fun: &ast.FuncLit{
													Type: &ast.FuncType{
														Params:  &ast.FieldList{List: nil},
														Results: nil,
													},
													Body: &ast.BlockStmt{
														List: []ast.Stmt{
															&ast.AssignStmt{
																Lhs: []ast.Expr{&ast.Ident{Name: "ret0"}},
																Tok: token.DEFINE,
																Rhs: []ast.Expr{&ast.CallExpr{
																	Fun: &ast.SelectorExpr{
																		X:   &ast.Ident{Name: "obj"},
																		Sel: &ast.Ident{Name: "Method"},
																	},
																}},
															},
															&ast.ExprStmt{
																X: &ast.CallExpr{
																	Fun:  &ast.Ident{Name: "pp.Println"},
																	Args: []ast.Expr{&ast.Ident{Name: "ret0"}},
																},
															},
														},
													},
												},
												Args: []ast.Expr{},
											},
										},
									},
								},
							},
						},
						Imports: []*ast.ImportSpec{
							{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/k0kubun/pp/v3"`,
								},
							},
						},
					}

					cmpOpts := []cmp.Option{
						cmpopts.IgnoreFields(ast.Ident{}, "Obj", "NamePos"),
						cmpopts.IgnoreFields(ast.CallExpr{}, "Lparen", "Rparen"),
						cmpopts.IgnoreFields(ast.FuncLit{}, "Type"),
						cmpopts.IgnoreFields(ast.FuncType{}, "Func"),
						cmpopts.IgnoreFields(ast.FieldList{}, "Opening", "Closing"),
						cmpopts.IgnoreFields(ast.BlockStmt{}, "Lbrace", "Rbrace"),
						cmpopts.IgnoreFields(ast.AssignStmt{}, "TokPos"),
					}

					if diff := cmp.Diff(expectedSessionSrc, sessionSrcAddedCallExpr, cmpOpts...); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}

					return nil
				}),
					[]byte{}, nil,
					resolveExpect{types.PkgName("pp"), types.ImportPath(`"github.com/k0kubun/pp/v3"`)},
				)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok:   token.IMPORT,
						Specs: []ast.Spec{},
					},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{},
						},
					},
				},
				Imports: []*ast.ImportSpec{},
			},
		},
		{
			name:  "call method chain",
			input: "obj.Method1().Method2()",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				declRegistry.Decls = append(declRegistry.Decls, declregistry.Decl{
					Name:        "obj",
					TypeName:    "Object",
					TypePkgName: "pkg",
				})
			},
			setupSymbolIndex: &symbols.SymbolIndex{
				Methods: map[types.PkgName][]symbols.MethodSet{
					"pkg": {
						{
							Name:             "Method2",
							ReceiverTypeName: "Object",
							Returns:          []symbols.ReturnSet{{TypeName: "SomeType", TypePkgName: "pkg"}},
						},
					},
				},
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", exprFlush(func(sessionSrcAddedCallExpr *ast.File, targetFile *os.File, fset *token.FileSet) error {
					// この時のsessionSrcの状態を確認する
					// この途中経過の状態をテストするやり方が微妙なので、要改善
					//
					// 実際には、"var obj = pkg.NewObject()""に関連するASTも含まれるが、ここでは省略
					expectedSessionSrc := &ast.File{
						Name: &ast.Ident{Name: "main"},
						Decls: []ast.Decl{
							&ast.GenDecl{
								Tok: token.IMPORT,
								Specs: []ast.Spec{
									&ast.ImportSpec{
										Path: &ast.BasicLit{
											Kind:  token.STRING,
											Value: `"github.com/k0kubun/pp/v3"`,
										},
									},
								},
							},
							&ast.FuncDecl{
								Name: &ast.Ident{Name: "main"},
								Type: &ast.FuncType{
									Params:  &ast.FieldList{List: nil},
									Results: nil,
								},
								Body: &ast.BlockStmt{
									List: []ast.Stmt{
										&ast.ExprStmt{
											X: &ast.CallExpr{
												Fun: &ast.FuncLit{
													Type: &ast.FuncType{
														Params:  &ast.FieldList{List: nil},
														Results: nil,
													},
													Body: &ast.BlockStmt{
														List: []ast.Stmt{
															&ast.AssignStmt{
																Lhs: []ast.Expr{&ast.Ident{Name: "ret0"}},
																Tok: token.DEFINE,
																Rhs: []ast.Expr{&ast.CallExpr{
																	Fun: &ast.SelectorExpr{
																		X: &ast.CallExpr{
																			Fun: &ast.SelectorExpr{
																				X:   &ast.Ident{Name: "obj"},
																				Sel: &ast.Ident{Name: "Method1"},
																			},
																		},
																		Sel: &ast.Ident{Name: "Method2"},
																	},
																}},
															},
															&ast.ExprStmt{
																X: &ast.CallExpr{
																	Fun:  &ast.Ident{Name: "pp.Println"},
																	Args: []ast.Expr{&ast.Ident{Name: "ret0"}},
																},
															},
														},
													},
												},
												Args: []ast.Expr{},
											},
										},
									},
								},
							},
						},
						Imports: []*ast.ImportSpec{
							{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/k0kubun/pp/v3"`,
								},
							},
						},
					}

					cmpOpts := []cmp.Option{
						cmpopts.IgnoreFields(ast.Ident{}, "Obj", "NamePos"),
						cmpopts.IgnoreFields(ast.CallExpr{}, "Lparen", "Rparen"),
						cmpopts.IgnoreFields(ast.FuncLit{}, "Type"),
						cmpopts.IgnoreFields(ast.FuncType{}, "Func"),
						cmpopts.IgnoreFields(ast.FieldList{}, "Opening", "Closing"),
						cmpopts.IgnoreFields(ast.BlockStmt{}, "Lbrace", "Rbrace"),
						cmpopts.IgnoreFields(ast.AssignStmt{}, "TokPos"),
					}

					if diff := cmp.Diff(expectedSessionSrc, sessionSrcAddedCallExpr, cmpOpts...); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}

					return nil
				}),
					[]byte{}, nil,
					resolveExpect{types.PkgName("pp"), types.ImportPath(`"github.com/k0kubun/pp/v3"`)},
				)
			},
			// 実際は"var obj = pkg.NewObject()"に関連するASTも含まれるが、ここでは省略
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok:   token.IMPORT,
						Specs: []ast.Spec{},
					},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{},
						},
					},
				},
				Imports: []*ast.ImportSpec{},
			},
		},
		{
			name:  "call void function of the package",
			input: "pkg.VoidFunction()",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				// 初期状態のセットアップが不要な場合は空の関数を指定
			},
			setupSymbolIndex: &symbols.SymbolIndex{
				Funcs: map[types.PkgName][]symbols.FuncSet{
					"pkg": {
						{
							Name:    "VoidFunction",
							Returns: []symbols.ReturnSet{}, // void: 返り値なし
						},
					},
				},
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", exprFlush(func(sessionSrc *ast.File, targetFile *os.File, fset *token.FileSet) error {
					expectedSessionSrc := &ast.File{
						Name: &ast.Ident{Name: "main"},
						Decls: []ast.Decl{
							&ast.GenDecl{
								Tok: token.IMPORT,
								Specs: []ast.Spec{
									&ast.ImportSpec{
										Path: &ast.BasicLit{
											Kind:  token.STRING,
											Value: `"github.com/test/pkg"`,
										},
									},
								},
							},
							&ast.FuncDecl{
								Name: &ast.Ident{Name: "main"},
								Type: &ast.FuncType{
									Params:  &ast.FieldList{List: nil},
									Results: nil,
								},
								Body: &ast.BlockStmt{
									List: []ast.Stmt{
										&ast.ExprStmt{
											X: &ast.CallExpr{
												Fun: &ast.SelectorExpr{
													X:   &ast.Ident{Name: "pkg"},
													Sel: &ast.Ident{Name: "VoidFunction"},
												},
											},
										},
									},
								},
							},
						},
						Imports: []*ast.ImportSpec{
							{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/test/pkg"`,
								},
							},
						},
					}

					cmpOpts := []cmp.Option{
						cmpopts.IgnoreFields(ast.Ident{}, "Obj", "NamePos"),
						cmpopts.IgnoreFields(ast.CallExpr{}, "Lparen", "Rparen"),
						cmpopts.IgnoreFields(ast.FieldList{}, "Opening", "Closing"),
						cmpopts.IgnoreFields(ast.BlockStmt{}, "Lbrace", "Rbrace"),
					}
					if diff := cmp.Diff(expectedSessionSrc, sessionSrc, cmpOpts...); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}
					return nil
				}),
					[]byte{}, nil,
					resolveExpect{types.PkgName("pkg"), types.ImportPath(`"github.com/test/pkg"`)},
				)
			},
			// pkg の import が追加されてから cleanCallExprFromSessionSrc で削除されるため、空の GenDecl が残る
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok:   token.IMPORT,
						Specs: []ast.Spec{},
					},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{},
						},
					},
				},
				Imports: []*ast.ImportSpec{},
			},
		},
		{
			name:  "call void method",
			input: "obj.VoidMethod()",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				declRegistry.Decls = append(declRegistry.Decls, declregistry.Decl{
					Name:        "obj",
					TypeName:    "Object",
					TypePkgName: "pkg",
				})
			},
			setupSymbolIndex: &symbols.SymbolIndex{
				Methods: map[types.PkgName][]symbols.MethodSet{
					"pkg": {
						{
							Name:             "VoidMethod",
							ReceiverTypeName: "Object",
							Returns:          []symbols.ReturnSet{}, // void: 返り値なし
						},
					},
				},
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go",
					// void なので pp.Println ではラップせず、元の CallExpr がそのまま追加されてflushされる
					// メソッド呼び出しなのでimportは追加されない
					exprFlush(func(sessionSrc *ast.File, targetFile *os.File, fset *token.FileSet) error {
						expectedSessionSrc := &ast.File{
							Name: &ast.Ident{Name: "main"},
							Decls: []ast.Decl{
								&ast.FuncDecl{
									Name: &ast.Ident{Name: "main"},
									Type: &ast.FuncType{
										Params:  &ast.FieldList{List: nil},
										Results: nil,
									},
									Body: &ast.BlockStmt{
										List: []ast.Stmt{
											&ast.ExprStmt{
												X: &ast.CallExpr{
													Fun: &ast.SelectorExpr{
														X:   &ast.Ident{Name: "obj"},
														Sel: &ast.Ident{Name: "VoidMethod"},
													},
												},
											},
										},
									},
								},
							},
						}

						cmpOpts := []cmp.Option{
							cmpopts.IgnoreFields(ast.Ident{}, "Obj", "NamePos"),
							cmpopts.IgnoreFields(ast.CallExpr{}, "Lparen", "Rparen"),
							cmpopts.IgnoreFields(ast.FieldList{}, "Opening", "Closing"),
							cmpopts.IgnoreFields(ast.BlockStmt{}, "Lbrace", "Rbrace"),
							cmpopts.IgnoreFields(ast.FuncType{}, "Func"),
						}
						if diff := cmp.Diff(expectedSessionSrc, sessionSrc, cmpOpts...); diff != "" {
							t.Errorf("mismatch (-want +got):\n%s", diff)
						}
						return nil
					}),
					[]byte{}, nil,
				)
			},
			// import は一切追加されないため GenDecl も存在しない
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{},
						},
					},
				},
			},
		},
		{
			name:  "call selector expr of unregistered package",
			input: "pkg.Var",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				// 初期状態のセットアップが不要な場合は空の関数を指定
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", exprFlush(func(sessionSrc *ast.File, targetFile *os.File, fset *token.FileSet) error {
					expectedSessionSrc := &ast.File{
						Name: &ast.Ident{Name: "main"},
						Decls: []ast.Decl{
							&ast.GenDecl{
								Tok: token.IMPORT,
								Specs: []ast.Spec{
									&ast.ImportSpec{
										Path: &ast.BasicLit{
											Kind:  token.STRING,
											Value: `"github.com/test/pkg"`,
										},
									},
									&ast.ImportSpec{
										Path: &ast.BasicLit{
											Kind:  token.STRING,
											Value: `"github.com/k0kubun/pp/v3"`,
										},
									},
								},
							},
							&ast.FuncDecl{
								Name: &ast.Ident{Name: "main"},
								Type: &ast.FuncType{
									Params:  &ast.FieldList{List: nil},
									Results: nil,
								},
								Body: &ast.BlockStmt{
									List: []ast.Stmt{
										&ast.ExprStmt{
											X: &ast.CallExpr{
												Fun: &ast.Ident{Name: "pp.Println"},
												Args: []ast.Expr{
													&ast.SelectorExpr{
														X:   &ast.Ident{Name: "pkg"},
														Sel: &ast.Ident{Name: "Var"},
													},
												},
											},
										},
									},
								},
							},
						},
						Imports: []*ast.ImportSpec{
							{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/test/pkg"`,
								},
							},
							{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/k0kubun/pp/v3"`,
								},
							},
						},
					}

					cmpOpts := []cmp.Option{
						cmpopts.IgnoreFields(ast.Ident{}, "Obj", "NamePos"),
						cmpopts.IgnoreFields(ast.CallExpr{}, "Lparen", "Rparen"),
						cmpopts.IgnoreFields(ast.FieldList{}, "Opening", "Closing"),
						cmpopts.IgnoreFields(ast.BlockStmt{}, "Lbrace", "Rbrace"),
					}
					if diff := cmp.Diff(expectedSessionSrc, sessionSrc, cmpOpts...); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}
					return nil
				}),
					[]byte{}, nil, resolveExpect{types.PkgName("pkg"), types.ImportPath(`"github.com/test/pkg"`)},
					resolveExpect{types.PkgName("pp"), types.ImportPath(`"github.com/k0kubun/pp/v3"`)},
				)
			},
			// pkg と pp の import が追加されてから cleanCallExprFromSessionSrc で両方削除されるため、空の GenDecl が残る
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok:   token.IMPORT,
						Specs: []ast.Spec{},
					},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{},
						},
					},
				},
				Imports: []*ast.ImportSpec{},
			},
		},
		{
			name:  "call selector expr of registered decl",
			input: "obj.Field",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				declRegistry.Decls = append(declRegistry.Decls, declregistry.Decl{
					Name:        "obj",
					TypeName:    "Object",
					TypePkgName: "pkg",
				})
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", exprFlush(func(sessionSrc *ast.File, targetFile *os.File, fset *token.FileSet) error {
					expectedSessionSrc := &ast.File{
						Name: &ast.Ident{Name: "main"},
						Decls: []ast.Decl{
							&ast.GenDecl{
								Tok: token.IMPORT,
								Specs: []ast.Spec{
									&ast.ImportSpec{
										Path: &ast.BasicLit{
											Kind:  token.STRING,
											Value: `"github.com/k0kubun/pp/v3"`,
										},
									},
								},
							},
							&ast.FuncDecl{
								Name: &ast.Ident{Name: "main"},
								Type: &ast.FuncType{
									Params:  &ast.FieldList{List: nil},
									Results: nil,
								},
								Body: &ast.BlockStmt{
									List: []ast.Stmt{
										&ast.ExprStmt{
											X: &ast.CallExpr{
												Fun: &ast.Ident{Name: "pp.Println"},
												Args: []ast.Expr{
													&ast.SelectorExpr{
														X:   &ast.Ident{Name: "obj"},
														Sel: &ast.Ident{Name: "Field"},
													},
												},
											},
										},
									},
								},
							},
						},
						Imports: []*ast.ImportSpec{
							{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/k0kubun/pp/v3"`,
								},
							},
						},
					}

					cmpOpts := []cmp.Option{
						cmpopts.IgnoreFields(ast.Ident{}, "Obj", "NamePos"),
						cmpopts.IgnoreFields(ast.CallExpr{}, "Lparen", "Rparen"),
						cmpopts.IgnoreFields(ast.FieldList{}, "Opening", "Closing"),
						cmpopts.IgnoreFields(ast.BlockStmt{}, "Lbrace", "Rbrace"),
					}
					if diff := cmp.Diff(expectedSessionSrc, sessionSrc, cmpOpts...); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}
					return nil
				}),
					[]byte{}, nil,
					resolveExpect{types.PkgName("pp"), types.ImportPath(`"github.com/k0kubun/pp/v3"`)},
				)
			},
			// pp の import が追加されてから cleanCallExprFromSessionSrc で削除されるため、空の GenDecl が残る
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok:   token.IMPORT,
						Specs: []ast.Spec{},
					},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{},
						},
					},
				},
				Imports: []*ast.ImportSpec{},
			},
		},
		{
			name:  "defined variable",
			input: "x",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				declRegistry.Decls = append(declRegistry.Decls, declregistry.Decl{
					Name:        "x",
					TypeName:    "int",
					TypePkgName: "",
				}) // 事前にfmtも登録しておく
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", exprFlush(func(sessionSrcAddedCallExpr *ast.File, targetFile *os.File, fset *token.FileSet) error {
					// この時のsessionSrcの状態を確認する
					// この途中経過の状態をテストするやり方が微妙なので、要改善
					//
					// 実際には、"var x = 10""に関連するASTも含まれるが、ここでは省略
					expectedSessionSrc := &ast.File{
						Name: &ast.Ident{Name: "main"},
						Decls: []ast.Decl{
							&ast.GenDecl{
								Tok: token.IMPORT,
								Specs: []ast.Spec{
									&ast.ImportSpec{
										Path: &ast.BasicLit{
											Kind:  token.STRING,
											Value: `"github.com/k0kubun/pp/v3"`,
										},
									}},
							},
							&ast.FuncDecl{
								Name: &ast.Ident{Name: "main"},
								Type: &ast.FuncType{
									Params:  &ast.FieldList{List: nil},
									Results: nil,
								},
								Body: &ast.BlockStmt{
									List: []ast.Stmt{
										&ast.ExprStmt{
											X: &ast.CallExpr{
												Fun: &ast.Ident{Name: "pp.Println"},
												Args: []ast.Expr{
													&ast.Ident{Name: "x"},
												},
											},
										},
									},
								},
							},
						},
						Imports: []*ast.ImportSpec{
							{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/k0kubun/pp/v3"`,
								},
							},
						},
					}

					cmpOpts := []cmp.Option{
						cmpopts.IgnoreFields(ast.Ident{}, "Obj", "NamePos"),
						cmpopts.IgnoreFields(ast.CallExpr{}, "Lparen", "Rparen"),
					}

					if diff := cmp.Diff(expectedSessionSrc, sessionSrcAddedCallExpr, cmpOpts...); diff != "" {
						t.Errorf("mismatch (-want +got):\n%s", diff)
					}

					return nil
				}),
					[]byte{}, nil,
					resolveExpect{types.PkgName("pp"), types.ImportPath(`"github.com/k0kubun/pp/v3"`)},
				)
			},
			// 実際は"var x = 10"のASTも含まれるが、ここでは省略
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok:   token.IMPORT,
						Specs: []ast.Spec{},
					},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{},
						},
					},
				},
				Imports: []*ast.ImportSpec{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			registry := declregistry.NewRegistry()
			tt.setupDeclRegistry(registry)

			symbolIdx := tt.setupSymbolIndex

			sut, err := NewExecutor(registry, symbolIdx)
			if err != nil {
				t.Fatalf("failed to create Executor: %v", err)
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockFiler := NewMockfiler(ctrl)
			mockCommander := NewMockcommander(ctrl)
			mockImportPathResolver := NewMockimportPathResolver(ctrl)
			tt.setupMocks(mockFiler, mockCommander, mockImportPathResolver)

			sut.filer = mockFiler
			sut.commander = mockCommander
			sut.importPathResolver = mockImportPathResolver

			// loadsessionSrcFile 等が実際のファイルを触って出すノイズ出力を抑制する
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w
			sut.Execute(tt.input)
			w.Close()
			os.Stdout = oldStdout
			r.Close()

			// 位置情報等はここでは無視する
			cmpOpts := []cmp.Option{
				cmpopts.IgnoreFields(ast.Ident{}, "Obj", "NamePos"),
				cmpopts.IgnoreFields(ast.GenDecl{}, "TokPos", "Lparen", "Rparen"),
				cmpopts.IgnoreFields(ast.CallExpr{}, "Lparen", "Rparen"),
				cmpopts.IgnoreFields(ast.BasicLit{}, "ValuePos"),
				cmpopts.IgnoreFields(ast.UnaryExpr{}, "OpPos"),
				cmpopts.IgnoreFields(ast.CompositeLit{}, "Lbrace", "Rbrace"),
				cmpopts.IgnoreFields(ast.KeyValueExpr{}, "Colon"),
				cmpopts.IgnoreFields(ast.BlockStmt{}, "Lbrace", "Rbrace"),
				cmpopts.IgnoreFields(ast.AssignStmt{}, "TokPos"),
			}

			if diff := cmp.Diff(tt.expectedSessionSrc, sut.sessionSrc, cmpOpts...); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestExecutor_Execute_Error(t *testing.T) {
	tests := []struct {
		name               string
		input              string
		setupDeclRegistry  func(*declregistry.DeclRegistry)
		setupMocks         func(*Mockfiler, *Mockcommander, *MockimportPathResolver)
		expectedSessionSrc *ast.File
		expectedErrMsg     string
	}{
		{
			name:              "input invalid syntax",
			input:             "x := ",
			setupDeclRegistry: func(dr *declregistry.DeclRegistry) {},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{},
						},
					},
				},
			},
			expectedErrMsg: "\n\x1b[31m[BAD INPUT ERROR]\n invalid input syntax\x1b[0m\n\n",
		},
		{
			name:              "input unsupported statement type",
			input:             "if true {}",
			setupDeclRegistry: func(dr *declregistry.DeclRegistry) {},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{},
						},
					},
				},
			},
			expectedErrMsg: "\n\x1b[31m[BAD INPUT ERROR]\n unsupported statement type\x1b[0m\n\n",
		},
		{
			name:              "input unsupported expression type",
			input:             "x.(int)",
			setupDeclRegistry: func(dr *declregistry.DeclRegistry) {},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{},
						},
					},
				},
			},
			expectedErrMsg: "\n\x1b[31m[BAD INPUT ERROR]\n unsupported expression type\x1b[0m\n\n",
		},
		{
			name:              "clean err element of sessionSrc when commander returns error",
			input:             `x = "y"`, // y is undefined
			setupDeclRegistry: func(dr *declregistry.DeclRegistry) {},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				errMsg := "# command-line-arguments\n1769312920_gonsole_session_src.go:3:2: undefined: x"
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "1769312920_gonsole_session_src.go", declFlush(), []byte{}, &exec.ExitError{Stderr: []byte(errMsg)})
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{},
						},
					},
				},
			},
			expectedErrMsg: "\n\x1b[31m[BAD INPUT ERROR]\n \n1 errors found\n\nundefined: x\n\n\x1b[0m\n\n",
		},
		{
			name:  "when commander returns error, clean err element of sessionSrc but import remains if other declarations use it",
			input: "x := pkg.Variable", // x is already defined
			setupDeclRegistry: func(dr *declregistry.DeclRegistry) {
				dr.Decls = append(dr.Decls, declregistry.Decl{
					Name:        "x",
					TypeName:    "Variable",
					TypePkgName: "pkg",
				})
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				errMsg := "# command-line-arguments\n1769312920_gonsole_session_src.go:8:4: no new variables on left side of :="
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "1769312920_gonsole_session_src.go", declFlush(), []byte{}, &exec.ExitError{Stderr: []byte(errMsg)},
					resolveExpect{types.PkgName("pkg"), types.ImportPath(`"github.com/test/pkg"`)},
				)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok: token.IMPORT,
						Specs: []ast.Spec{
							&ast.ImportSpec{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/test/pkg"`,
								},
							},
						},
					},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							// 実際はすでに定義している"x := pkg.Variable"のASTも含まれるが、ここではセットアップしていないので省略
							List: []ast.Stmt{},
						},
					},
				},
				Imports: []*ast.ImportSpec{
					{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"github.com/test/pkg"`,
						},
					},
				},
			},
			expectedErrMsg: "\n\x1b[31m[BAD INPUT ERROR]\n \n1 errors found\n\nno new variables on left side of :=\n\n\x1b[0m\n\n",
		},
		{
			name:  "when commander returns error, clean err element of sessionSrc and import if no other declarations use it",
			input: "x := pkg.Variable", // x is already defined
			setupDeclRegistry: func(dr *declregistry.DeclRegistry) {
				dr.Decls = append(dr.Decls, declregistry.Decl{
					Name:        "x",
					TypeName:    "Variable",
					TypePkgName: "anotherpkg",
				}) // 事前にpkg自体も登録しておく
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				errMsg := "# command-line-arguments\n1769312920_gonsole_session_src.go:8:4: no new variables on left side of :="
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "1769312920_gonsole_session_src.go", declFlush(), []byte{}, &exec.ExitError{Stderr: []byte(errMsg)},
					resolveExpect{types.PkgName("pkg"), types.ImportPath(`"github.com/test/pkg"`)},
				)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{}},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							// 実際はすでに定義している"x := pkg.Variable"のASTも含まれるが、ここではセットアップしていないので省略
							List: []ast.Stmt{},
						},
					},
				},
				Imports: []*ast.ImportSpec{},
			},
			expectedErrMsg: "\n\x1b[31m[BAD INPUT ERROR]\n \n1 errors found\n\nno new variables on left side of :=\n\n\x1b[0m\n\n",
		},
		{
			name:              "clean err element of ExprStmt from sessionSrc when commander returns error",
			input:             "pkg.VoidFunction()", // void なので ExprStmt がそのまま追加される (returnValuesCnt==0)
			setupDeclRegistry: func(dr *declregistry.DeclRegistry) {},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				errMsg := "# command-line-arguments\ntest.go:3:2: undefined: pkg"
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", declFlush(), []byte{}, &exec.ExitError{Stderr: []byte(errMsg)},
					resolveExpect{types.PkgName("pkg"), types.ImportPath(`"github.com/test/pkg"`)},
				)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{Tok: token.IMPORT, Specs: []ast.Spec{}},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{},
						},
					},
				},
				Imports: []*ast.ImportSpec{},
			},
			expectedErrMsg: "\n\x1b[31m[BAD INPUT ERROR]\n \n1 errors found\n\nundefined: pkg\n\n\x1b[0m\n\n",
		},
		{
			name:  "when commander returns error, clean ExprStmt but import remains if other declarations use it",
			input: "pkg.VoidFunction()",
			setupDeclRegistry: func(dr *declregistry.DeclRegistry) {
				dr.Decls = append(dr.Decls, declregistry.Decl{
					Name:        "obj",
					TypeName:    "Object",
					TypePkgName: "pkg", // pkg を利用している宣言が存在する
				})
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				errMsg := "# command-line-arguments\ntest.go:3:2: undefined: VoidFunction"
				setupMocks(t, mockFiler, mockCommander, mockImportPathResolver, "test.go", declFlush(), []byte{}, &exec.ExitError{Stderr: []byte(errMsg)},
					resolveExpect{types.PkgName("pkg"), types.ImportPath(`"github.com/test/pkg"`)},
				)
			},
			expectedSessionSrc: &ast.File{
				Name: &ast.Ident{Name: "main"},
				Decls: []ast.Decl{
					&ast.GenDecl{
						Tok: token.IMPORT,
						Specs: []ast.Spec{
							&ast.ImportSpec{
								Path: &ast.BasicLit{
									Kind:  token.STRING,
									Value: `"github.com/test/pkg"`,
								},
							},
						},
					},
					&ast.FuncDecl{
						Name: &ast.Ident{Name: "main"},
						Type: &ast.FuncType{
							Params:  &ast.FieldList{List: nil},
							Results: nil,
						},
						Body: &ast.BlockStmt{
							List: []ast.Stmt{},
						},
					},
				},
				Imports: []*ast.ImportSpec{
					{
						Path: &ast.BasicLit{
							Kind:  token.STRING,
							Value: `"github.com/test/pkg"`,
						},
					},
				},
			},
			expectedErrMsg: "\n\x1b[31m[BAD INPUT ERROR]\n \n1 errors found\n\nundefined: VoidFunction\n\n\x1b[0m\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := declregistry.NewRegistry()

			sut, err := NewExecutor(registry, nil)
			if err != nil {
				t.Fatalf("failed to create Executor: %v", err)
			}

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockFiler := NewMockfiler(ctrl)
			mockCommander := NewMockcommander(ctrl)
			mockImportPathResolver := NewMockimportPathResolver(ctrl)
			tt.setupMocks(mockFiler, mockCommander, mockImportPathResolver)

			sut.filer = mockFiler
			sut.commander = mockCommander
			sut.importPathResolver = mockImportPathResolver

			tt.setupDeclRegistry(registry)

			// 標準出力を一時的に差し替え
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// テスト終了時に元に戻す
			defer func() {
				os.Stdout = oldStdout
			}()

			sut.Execute(tt.input)

			if importPathAddedInSession != "" {
				t.Fatalf("importPathAddedInSession should be empty, but got %q", importPathAddedInSession)
			}

			// パイプを閉じて出力を読み取る
			if err := w.Close(); err != nil {
				t.Fatalf("failed to close pipe writer: %v", err)
			}
			var buf bytes.Buffer
			if _, err := buf.ReadFrom(r); err != nil {
				t.Fatalf("failed to read from pipe: %v", err)
			}
			gotErrMsg := buf.String()
			if gotErrMsg != tt.expectedErrMsg {
				t.Errorf("expected error message %q, but got %q", tt.expectedErrMsg, gotErrMsg)
			}

			// 位置情報等はここでは無視する
			cmpOpts := []cmp.Option{
				cmpopts.IgnoreFields(ast.File{}, "Package", "FileStart", "FileEnd", "Scope"),
				cmpopts.IgnoreFields(ast.FuncType{}, "Func"),
				cmpopts.IgnoreFields(ast.FieldList{}, "Opening", "Closing"),
				cmpopts.IgnoreFields(ast.AssignStmt{}, "TokPos"),
				cmpopts.IgnoreFields(ast.BasicLit{}, "ValuePos"),
				cmpopts.IgnoreFields(ast.Ident{}, "Obj", "NamePos"),
				cmpopts.IgnoreFields(ast.GenDecl{}, "TokPos", "Lparen", "Rparen"),
				cmpopts.IgnoreFields(ast.CallExpr{}, "Lparen", "Rparen"),
				cmpopts.IgnoreFields(ast.BasicLit{}, "ValuePos"),
				cmpopts.IgnoreFields(ast.UnaryExpr{}, "OpPos"),
				cmpopts.IgnoreFields(ast.CompositeLit{}, "Lbrace", "Rbrace"),
				cmpopts.IgnoreFields(ast.KeyValueExpr{}, "Colon"),
				cmpopts.IgnoreFields(ast.BlockStmt{}, "Lbrace", "Rbrace"),
				cmpopts.IgnoreFields(ast.AssignStmt{}, "TokPos"),
			}

			if diff := cmp.Diff(tt.expectedSessionSrc, sut.sessionSrc, cmpOpts...); diff != "" {
				t.Errorf("mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
