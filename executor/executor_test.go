package executor

import (
	"go/ast"
	"go/token"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/kakkky/gonsole/declregistry"
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
				// filer
				mockFiler.EXPECT().createTmpFile().DoAndReturn(func() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
					r, w, _ := os.Pipe()
					return w, "test.go", func() { r.Close() }, nil
				}).Times(1)
				mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1) // 呼ばれていることが確認できればいいのでgomock.Any()で対応

				// commander
				mockCommander.EXPECT().execGoRun("test.go").Return([]byte{}, nil).Times(1)
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
				// filer
				mockFiler.EXPECT().createTmpFile().DoAndReturn(func() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
					r, w, _ := os.Pipe()
					return w, "test.go", func() { r.Close() }, nil
				}).Times(1)
				mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1) // 呼ばれていることが確認できればいいのでgomock.Any()で対応

				// commander
				mockCommander.EXPECT().execGoRun("test.go").Return([]byte{}, nil).Times(1)

				// importPathResolver
				mockImportPathResolver.EXPECT().resolve(types.PkgName("pkg")).Return(types.ImportPath(`"github.com/test/pkg"`), nil).Times(1)
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
				// filer
				mockFiler.EXPECT().createTmpFile().DoAndReturn(func() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
					r, w, _ := os.Pipe()
					return w, "test.go", func() { r.Close() }, nil
				}).Times(1)
				mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1) // 呼ばれていることが確認できればいいのでgomock.Any()で対応

				// commander
				mockCommander.EXPECT().execGoRun("test.go").Return([]byte{}, nil).Times(1)

				// importPathResolver
				mockImportPathResolver.EXPECT().resolve(types.PkgName("pkg")).Return(types.ImportPath(`"github.com/test/pkg"`), nil).Times(1)
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
				// filer
				mockFiler.EXPECT().createTmpFile().DoAndReturn(func() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
					r, w, _ := os.Pipe()
					return w, "test.go", func() { r.Close() }, nil
				}).Times(1)
				mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1) // 呼ばれていることが確認できればいいのでgomock.Any()で対応

				// commander
				mockCommander.EXPECT().execGoRun("test.go").Return([]byte{}, nil).Times(1)

				// importPathResolver
				mockImportPathResolver.EXPECT().resolve(types.PkgName("pkg")).Return(types.ImportPath(`"github.com/test/pkg"`), nil).Times(1)
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
				declRegistry.Register("var obj = pkg.NewObject()") // 事前にobjを登録しておく。右辺は適当
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				// filer
				mockFiler.EXPECT().createTmpFile().DoAndReturn(func() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
					r, w, _ := os.Pipe()
					return w, "test.go", func() { r.Close() }, nil
				}).Times(1)
				mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1) // 呼ばれていることが確認できればいいのでgomock.Any()で対応

				// commander
				mockCommander.EXPECT().execGoRun("test.go").Return([]byte{}, nil).Times(1)
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
												Names: []*ast.Ident{{Name: "x"}},
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
				declRegistry.Register("var obj = pkg.NewObject()") // 事前にobjを登録しておく。右辺は適当
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				// filer
				mockFiler.EXPECT().createTmpFile().DoAndReturn(func() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
					r, w, _ := os.Pipe()
					return w, "test.go", func() { r.Close() }, nil
				}).Times(1)
				mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1) // 呼ばれていることが確認できればいいのでgomock.Any()で対応

				// commander
				mockCommander.EXPECT().execGoRun("test.go").Return([]byte{}, nil).Times(1)
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
				declRegistry.Register("var obj = pkg.NewObject()") // 事前にobjを登録しておく。右辺は適当
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				// filer
				mockFiler.EXPECT().createTmpFile().DoAndReturn(func() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
					r, w, _ := os.Pipe()
					return w, "test.go", func() { r.Close() }, nil
				}).Times(1)
				mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1) // 呼ばれていることが確認できればいいのでgomock.Any()で対応

				// commander
				mockCommander.EXPECT().execGoRun("test.go").Return([]byte{}, nil).Times(1)
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
				// filer
				mockFiler.EXPECT().createTmpFile().DoAndReturn(func() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
					r, w, _ := os.Pipe()
					return w, "test.go", func() { r.Close() }, nil
				}).Times(1)
				mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1) // 呼ばれていることが確認できればいいのでgomock.Any()で対応

				// commander
				mockCommander.EXPECT().execGoRun("test.go").Return([]byte{}, nil).Times(1)
				// importPathResolver
				mockImportPathResolver.EXPECT().resolve(types.PkgName("pkg")).Return(types.ImportPath(`"github.com/test/pkg"`), nil).Times(1)
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
												Names: []*ast.Ident{{Name: "x"}},
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
				// filer
				mockFiler.EXPECT().createTmpFile().DoAndReturn(func() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
					r, w, _ := os.Pipe()
					return w, "test.go", func() { r.Close() }, nil
				}).Times(1)
				mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1) // 呼ばれていることが確認できればいいのでgomock.Any()で対応

				// commander
				mockCommander.EXPECT().execGoRun("test.go").Return([]byte{}, nil).Times(1)

				// importPathResolver
				mockImportPathResolver.EXPECT().resolve(types.PkgName("pkg")).Return(types.ImportPath(`"github.com/test/pkg"`), nil).Times(1)
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
												Names: []*ast.Ident{{Name: "x"}},
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
				// filer
				mockFiler.EXPECT().createTmpFile().DoAndReturn(func() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
					r, w, _ := os.Pipe()
					return w, "test.go", func() { r.Close() }, nil
				}).Times(1)
				mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1) // 呼ばれていることが確認できればいいのでgomock.Any()で対応

				// commander
				mockCommander.EXPECT().execGoRun("test.go").Return([]byte{}, nil).Times(1)

				// importPathResolver
				mockImportPathResolver.EXPECT().resolve(types.PkgName("pkg")).Return(types.ImportPath(`"github.com/test/pkg"`), nil).Times(1)

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
				// filer
				mockFiler.EXPECT().createTmpFile().DoAndReturn(func() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
					r, w, _ := os.Pipe()
					return w, "test.go", func() { r.Close() }, nil
				}).Times(1)
				mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1) // 呼ばれていることが確認できればいいのでgomock.Any()で対応

				// commander
				mockCommander.EXPECT().execGoRun("test.go").Return([]byte{}, nil).Times(1)
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
			name:  "call function of the package",
			input: "pkg.Function()",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				// 初期状態のセットアップが不要な場合は空の関数を指定
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				// filer
				mockFiler.EXPECT().createTmpFile().DoAndReturn(func() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
					r, w, _ := os.Pipe()
					return w, "test.go", func() { r.Close() }, nil
				}).Times(1)

				gomock.InOrder(
					// 関数呼び出しを追加してflushする時
					mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(sessionSrcAddedCallExpr *ast.File, targetFile *os.File, fset *token.FileSet) error {
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
												Value: `"fmt"`,
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
													Fun: &ast.Ident{Name: "fmt.Println"},
													Args: []ast.Expr{&ast.CallExpr{
														Fun: &ast.SelectorExpr{
															X:   &ast.Ident{Name: "pkg"},
															Sel: &ast.Ident{Name: "Function"},
														},
														Args: nil,
													}},
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
										Value: `"fmt"`,
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
					}).Times(1),
					// 関数を削除してflushする時
					mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1),
				)

				// commander
				mockCommander.EXPECT().execGoRun("test.go").Return([]byte{}, nil).Times(1)

				// importPathResolver
				gomock.InOrder(
					mockImportPathResolver.EXPECT().resolve(types.PkgName("pkg")).Return(types.ImportPath(`"github.com/test/pkg"`), nil).Times(1),
					mockImportPathResolver.EXPECT().resolve(types.PkgName("fmt")).Return(types.ImportPath(`"fmt"`), nil).Times(1),
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
			name:  "call function of the package when package is used by other declaration",
			input: "pkg.Function()",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				declRegistry.Register("var x = pkg.Variable") // 事前にpkgを使用する宣言を登録しておく。右辺は適当
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				// filer
				mockFiler.EXPECT().createTmpFile().DoAndReturn(func() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
					r, w, _ := os.Pipe()
					return w, "test.go", func() { r.Close() }, nil
				}).Times(1)
				gomock.InOrder(
					// 関数呼び出しを追加してflushする時
					mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(sessionSrcAddedCallExpr *ast.File, targetFile *os.File, fset *token.FileSet) error {
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
												Value: `"fmt"`,
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
													Fun: &ast.Ident{Name: "fmt.Println"},
													Args: []ast.Expr{&ast.CallExpr{
														Fun: &ast.SelectorExpr{
															X:   &ast.Ident{Name: "pkg"},
															Sel: &ast.Ident{Name: "Function"},
														},
														Args: nil,
													}},
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
										Value: `"fmt"`,
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
					}).Times(1),
					// 関数呼び出しを削除してflushする時
					mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1),
				)

				// commander
				mockCommander.EXPECT().execGoRun("test.go").Return([]byte{}, nil).Times(1)

				// importPathResolver
				gomock.InOrder(
					mockImportPathResolver.EXPECT().resolve(types.PkgName("pkg")).Return(types.ImportPath(`"github.com/test/pkg"`), nil).Times(1),
					mockImportPathResolver.EXPECT().resolve(types.PkgName("fmt")).Return(types.ImportPath(`"fmt"`), nil).Times(1),
				)
			},
			// 実際は"var x = pkg.Variable"のASTも含まれるが、ここでは省略
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
				declRegistry.Register("var obj = pkg.NewObject()") // 事前にobjを登録しておく。右辺は適当
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				// filer
				mockFiler.EXPECT().createTmpFile().DoAndReturn(func() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
					r, w, _ := os.Pipe()
					return w, "test.go", func() { r.Close() }, nil
				}).Times(1)
				gomock.InOrder(
					// 関数呼び出しを追加してflushする時
					mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(sessionSrcAddedCallExpr *ast.File, targetFile *os.File, fset *token.FileSet) error {
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
												Value: `"fmt"`,
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
													Fun: &ast.Ident{Name: "fmt.Println"},
													Args: []ast.Expr{&ast.CallExpr{
														Fun: &ast.SelectorExpr{
															X:   &ast.Ident{Name: "obj"},
															Sel: &ast.Ident{Name: "Method"},
														},
														Args: nil,
													}},
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
										Value: `"fmt"`,
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
					}).Times(1),
					// メソッド呼び出しを削除してflushする時
					mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1),
				)

				// commander
				mockCommander.EXPECT().execGoRun("test.go").Return([]byte{}, nil).Times(1)

				// importPathResolver
				mockImportPathResolver.EXPECT().resolve(types.PkgName("fmt")).Return(types.ImportPath(`"fmt"`), nil).Times(1)
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
			name:  "call method chain",
			input: "obj.Method1().Method2()",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				declRegistry.Register("var obj = pkg.NewObject()") // 事前にobjを登録しておく。右辺は適当
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				// filer
				mockFiler.EXPECT().createTmpFile().DoAndReturn(func() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
					r, w, _ := os.Pipe()
					return w, "test.go", func() { r.Close() }, nil
				}).Times(1)
				gomock.InOrder(
					// 関数呼び出しを追加してflushする時
					mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(sessionSrcAddedCallExpr *ast.File, targetFile *os.File, fset *token.FileSet) error {
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
												Value: `"fmt"`,
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
													Fun: &ast.Ident{Name: "fmt.Println"},
													Args: []ast.Expr{&ast.CallExpr{
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
													}},
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
										Value: `"fmt"`,
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
					}).Times(1),
					// メソッド呼び出しを削除してflushする時
					mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1),
				)

				// commander
				mockCommander.EXPECT().execGoRun("test.go").Return([]byte{}, nil).Times(1)

				// importPathResolver
				mockImportPathResolver.EXPECT().resolve(types.PkgName("fmt")).Return(types.ImportPath(`"fmt"`), nil).Times(1)
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
			name:  "defined variable",
			input: "x",
			setupDeclRegistry: func(declRegistry *declregistry.DeclRegistry) {
				declRegistry.Register("var x = 10") // 事前にxを登録しておく。右辺は適当
			},
			setupMocks: func(mockFiler *Mockfiler, mockCommander *Mockcommander, mockImportPathResolver *MockimportPathResolver) {
				// filer
				mockFiler.EXPECT().createTmpFile().DoAndReturn(func() (tmpFile *os.File, tmpFileName string, cleanup func(), err error) {
					r, w, _ := os.Pipe()
					return w, "test.go", func() { r.Close() }, nil
				}).Times(1)
				gomock.InOrder(
					// 関数呼び出しを追加してflushする時
					mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(func(sessionSrcAddedCallExpr *ast.File, targetFile *os.File, fset *token.FileSet) error {
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
												Value: `"fmt"`,
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
													Fun: &ast.Ident{Name: "fmt.Println"},
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
										Value: `"fmt"`,
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
					}).Times(1),
					// 変数参照を削除してflushする時
					mockFiler.EXPECT().flush(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).Times(1),
				)

				// commander
				mockCommander.EXPECT().execGoRun("test.go").Return([]byte{}, nil).Times(1)
				mockImportPathResolver.EXPECT().resolve(types.PkgName("fmt")).Return(types.ImportPath(`"fmt"`), nil).Times(1)
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
			declregistry := declregistry.NewRegistry()
			tt.setupDeclRegistry(declregistry)

			sut, err := NewExecutor(declregistry)
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

			sut.Execute(tt.input)

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
