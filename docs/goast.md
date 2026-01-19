# Go AST (Abstract Syntax Tree) リファレンス

このドキュメントは、`go/ast`パッケージの型、インターフェース、構造体について解説します。

## 目次

- [AST構造の概要](#ast構造の概要)
- [主要な型とインターフェース](#主要な型とインターフェース)
- [Tree構造図解](#tree構造図解)
- [実践例](#実践例)

---

## AST構造の概要

Go言語のソースコードは、パース後に抽象構文木（AST）として表現されます。ASTは階層構造を持ち、以下のような階層で構成されています：

```
Package → File → Decl → Stmt/Expr
```

### 基本階層

1. **Package** (`*ast.Package`): パッケージ全体
2. **File** (`*ast.File`): 個々のGoファイル
3. **Decl** (`ast.Decl`): 宣言（関数、型、変数など）
4. **Stmt** (`ast.Stmt`): 文（代入、if、forなど）
5. **Expr** (`ast.Expr`): 式（識別子、リテラル、演算子など）

---

## 主要な型とインターフェース

### 1. `*ast.Package`

パッケージ全体を表す構造体。

```go
type Package struct {
    Name    string             // パッケージ名
    Scope   *Scope
    Imports map[string]*Object
    Files   map[string]*File   // ファイル名 → Fileのマップ
}
```

**対応するコード構造:**
```go
// ソースコード（example.go）
package main

import "fmt"

func main() {
    fmt.Println("Hello")
}
```

**AST表現:**
```
*ast.Package {
    Name: "main"
    Files: map[string]*ast.File {
        "example.go": &ast.File{...}
    }
}
```

---

### 2. `*ast.File`

単一のGoファイルを表す構造体。

```go
type File struct {
    Doc        *CommentGroup   // ファイルのドキュメントコメント
    Package    token.Pos       // "package"キーワードの位置
    Name       *Ident          // パッケージ名
    Decls      []Decl          // トップレベルの宣言
    Scope      *Scope
    Imports    []*ImportSpec   // インポート宣言
    Unresolved []*Ident
    Comments   []*CommentGroup // 全コメント
}
```

**重要なフィールド:**
- `Decls`: ファイル内の全ての宣言（関数、型、変数など）

**対応するコード構造:**
```go
// ソースコード
package main

import "fmt"

func Add(a, b int) int {
    return a + b
}
```

**AST表現:**
```
*ast.File {
    Name: &ast.Ident{Name: "main"}
    Decls: []ast.Decl{
        &ast.GenDecl{...},      // import宣言
        &ast.FuncDecl{...},     // Add関数
    }
}
```

---

### 3. `ast.Decl` インターフェース

宣言を表すインターフェース。実装型は以下の2つ：

#### 3-1. `*ast.FuncDecl` - 関数/メソッド宣言

```go
type FuncDecl struct {
    Doc  *CommentGroup // ドキュメントコメント
    Recv *FieldList    // レシーバ（メソッドの場合のみ）
    Name *Ident        // 関数名
    Type *FuncType     // 関数のシグネチャ
    Body *BlockStmt    // 関数本体
}
```

**重要なフィールド:**
- `Recv`: nil → 関数、非nil → メソッド
- `Doc`: ドキュメントコメント
- `Type`: 関数の型情報（引数、戻り値）

**コードとAST対応:**

```go
// ソースコード（関数）
// Add adds two numbers.
func Add(a, b int) int {
    return a + b
}
```

**AST表現:**
```
*ast.FuncDecl {
    Doc: &ast.CommentGroup{...}    // "Add adds two numbers."
    Recv: nil                       // レシーバなし（関数）
    Name: &ast.Ident{Name: "Add"}
    Type: &ast.FuncType{
        Params: &ast.FieldList{...}   // (a, b int)
        Results: &ast.FieldList{...}  // int
    }
}
```

```go
// ソースコード（メソッド）
// Bark makes the dog bark.
func (d *Dog) Bark() string {
    return "Woof!"
}
```

**AST表現:**
```
*ast.FuncDecl {
    Doc: &ast.CommentGroup{...}
    Recv: &ast.FieldList{            // レシーバあり（メソッド）
        List: []*ast.Field{
            {
                Names: []*ast.Ident{{Name: "d"}}
                Type: &ast.StarExpr{X: &ast.Ident{Name: "Dog"}}
            }
        }
    }
    Name: &ast.Ident{Name: "Bark"}
    Type: &ast.FuncType{
        Results: &ast.FieldList{
            List: []*ast.Field{
                {Type: &ast.Ident{Name: "string"}}
            }
        }
    }
}
```

#### 3-2. `*ast.GenDecl` - 汎用宣言（import, const, var, type）

```go
type GenDecl struct {
    Doc    *CommentGroup // ドキュメントコメント
    TokPos token.Pos     // Tokの位置
    Tok    token.Token   // IMPORT, CONST, TYPE, VAR
    Lparen token.Pos     // '('の位置（グループ宣言の場合）
    Specs  []Spec        // 具体的な宣言内容
    Rparen token.Pos     // ')'の位置
}
```

**重要なフィールド:**
- `Tok`: 宣言の種類を示すトークン
  - `token.VAR`: 変数宣言
  - `token.CONST`: 定数宣言
  - `token.TYPE`: 型宣言
  - `token.IMPORT`: インポート宣言
- `Specs`: 宣言の詳細

**コードとAST対応:**

```go
// ソースコード
var x = 42
```

**AST表現:**
```
*ast.GenDecl {
    Tok: token.VAR
    Specs: []ast.Spec{
        &ast.ValueSpec{
            Names: []*ast.Ident{{Name: "x"}}
            Values: []ast.Expr{&ast.BasicLit{Value: "42"}}
        }
    }
}
```

```go
// ソースコード
type Dog struct {
    Name string
}
```

**AST表現:**
```
*ast.GenDecl {
    Tok: token.TYPE
    Specs: []ast.Spec{
        &ast.TypeSpec{
            Name: &ast.Ident{Name: "Dog"}
            Type: &ast.StructType{...}
        }
    }
}
```

---

### 4. `*ast.FuncType` - 関数の型情報

```go
type FuncType struct {
    Func    token.Pos  // "func"キーワードの位置
    Params  *FieldList // 引数リスト
    Results *FieldList // 戻り値リスト
}
```

**コードとAST対応:**

```go
// ソースコード
func GetUser() (*User, error) {
    return nil, nil
}
```

**AST表現:**
```
*ast.FuncType {
    Params: nil  // 引数なし
    Results: &ast.FieldList{
        List: []*ast.Field{
            {Type: &ast.StarExpr{X: &ast.Ident{Name: "User"}}},  // *User
            {Type: &ast.Ident{Name: "error"}},                    // error
        }
    }
}
```

---

### 5. `ast.Spec` インターフェース

`GenDecl`の具体的な内容を表すインターフェース。

#### 5-1. `*ast.ValueSpec` - 変数/定数の宣言

```go
type ValueSpec struct {
    Doc     *CommentGroup // ドキュメントコメント
    Names   []*Ident      // 変数/定数名のリスト
    Type    Expr          // 型（省略可）
    Values  []Expr        // 初期値のリスト
    Comment *CommentGroup // 行コメント
}
```

**コードとAST対応:**

```go
// ソースコード
var x = 42
var s = "hello"
var p = pkg.NewDog("Max", 3)
```

**AST表現:**
```
*ast.ValueSpec {
    Names: []*ast.Ident{{Name: "x"}}
    Values: []ast.Expr{
        &ast.BasicLit{Kind: token.INT, Value: "42"}
    }
}

*ast.ValueSpec {
    Names: []*ast.Ident{{Name: "s"}}
    Values: []ast.Expr{
        &ast.BasicLit{Kind: token.STRING, Value: `"hello"`}
    }
}

*ast.ValueSpec {
    Names: []*ast.Ident{{Name: "p"}}
    Values: []ast.Expr{
        &ast.CallExpr{
            Fun: &ast.SelectorExpr{
                X: &ast.Ident{Name: "pkg"}
                Sel: &ast.Ident{Name: "NewDog"}
            }
            Args: []ast.Expr{...}
        }
    }
}
```

#### 5-2. `*ast.TypeSpec` - 型宣言

```go
type TypeSpec struct {
    Doc     *CommentGroup // ドキュメントコメント
    Name    *Ident        // 型名
    Assign  token.Pos     // '='の位置（型エイリアスの場合）
    Type    Expr          // 型定義
    Comment *CommentGroup // 行コメント
}
```

**コードとAST対応:**

```go
// ソースコード
type Dog struct {
    Name string
    Age  int
}
```

**AST表現:**
```
*ast.TypeSpec {
    Name: &ast.Ident{Name: "Dog"}
    Type: &ast.StructType{...}
}
```

```go
// ソースコード
type Animal interface {
    Eat()
    Sleep()
}
```

**AST表現:**
```
*ast.TypeSpec {
    Name: &ast.Ident{Name: "Animal"}
    Type: &ast.InterfaceType{...}
}
```

---

### 6. `*ast.StructType` - 構造体定義

```go
type StructType struct {
    Struct     token.Pos  // "struct"キーワードの位置
    Fields     *FieldList // フィールドリスト
    Incomplete bool       // 不完全な構造体か
}
```

**コードとAST対応:**

```go
// ソースコード
type Dog struct {
    Name string
    Age  int
    BaseAnimal  // 埋め込み型
}
```

**AST表現:**
```
*ast.StructType {
    Fields: &ast.FieldList{
        List: []*ast.Field{
            {
                Names: []*ast.Ident{{Name: "Name"}}
                Type: &ast.Ident{Name: "string"}
            },
            {
                Names: []*ast.Ident{{Name: "Age"}}
                Type: &ast.Ident{Name: "int"}
            },
            {
                Names: nil  // 埋め込み型（匿名フィールド）
                Type: &ast.Ident{Name: "BaseAnimal"}
            },
        }
    }
}
```

---

### 7. `*ast.InterfaceType` - インターフェース定義

```go
type InterfaceType struct {
    Interface  token.Pos  // "interface"キーワードの位置
    Methods    *FieldList // メソッドリスト
    Incomplete bool       // 不完全なインターフェースか
}
```

**コードとAST対応:**

```go
// ソースコード
type Animal interface {
    Eat()
    Sleep()
}
```

**AST表現:**
```
*ast.InterfaceType {
    Methods: &ast.FieldList{
        List: []*ast.Field{
            {
                Names: []*ast.Ident{{Name: "Eat"}}
                Type: &ast.FuncType{...}
            },
            {
                Names: []*ast.Ident{{Name: "Sleep"}}
                Type: &ast.FuncType{...}
            },
        }
    }
}
```

---

### 8. `ast.Expr` インターフェース

式を表すインターフェース。多数の実装型があります。

#### 8-1. `*ast.Ident` - 識別子

```go
type Ident struct {
    NamePos token.Pos // 識別子の位置
    Name    string    // 識別子の名前
    Obj     *Object   // オブジェクト情報
}
```

**用途:**
- 変数名、関数名、型名など全ての名前を表す
- 最も基本的なAST要素

**コードとAST対応:**
```go
// ソースコード
var x int
func Foo() {}
```

**AST表現:**
```
// "x" → &ast.Ident{Name: "x"}
// "int" → &ast.Ident{Name: "int"}
// "Foo" → &ast.Ident{Name: "Foo"}
```

#### 8-2. `*ast.SelectorExpr` - セレクタ式

```go
type SelectorExpr struct {
    X   Expr   // セレクタの左側
    Sel *Ident // セレクタの右側
}
```

**用途:**
- `pkg.Function()`: パッケージの関数呼び出し
- `obj.Method()`: メソッド呼び出し
- `pkg.Type`: パッケージの型参照

**コードとAST対応:**
```go
// ソースコード
fmt.Println("hello")
http.Server
dog.Name
```

**AST表現:**
```
// fmt.Println
&ast.SelectorExpr{
    X: &ast.Ident{Name: "fmt"}
    Sel: &ast.Ident{Name: "Println"}
}

// http.Server
&ast.SelectorExpr{
    X: &ast.Ident{Name: "http"}
    Sel: &ast.Ident{Name: "Server"}
}

// dog.Name
&ast.SelectorExpr{
    X: &ast.Ident{Name: "dog"}
    Sel: &ast.Ident{Name: "Name"}
}
```

**重要:** `X`は必ずしも`*ast.Ident`とは限らない（メソッドチェーン等）。

#### 8-3. `*ast.CompositeLit` - 複合リテラル

```go
type CompositeLit struct {
    Type       Expr      // リテラルの型
    Lbrace     token.Pos // '{'の位置
    Elts       []Expr    // 要素のリスト
    Rbrace     token.Pos // '}'の位置
    Incomplete bool
}
```

**用途:**
構造体、配列、スライス、マップのリテラル。

**コードとAST対応:**
```go
// ソースコード
var server = http.Server{}
var person = Person{Name: "Alice", Age: 30}
var nums = []int{1, 2, 3}
```

**AST表現:**
```
// http.Server{}
&ast.CompositeLit{
    Type: &ast.SelectorExpr{
        X: &ast.Ident{Name: "http"}
        Sel: &ast.Ident{Name: "Server"}
    }
    Elts: []ast.Expr{}  // 空
}

// Person{Name: "Alice", Age: 30}
&ast.CompositeLit{
    Type: &ast.Ident{Name: "Person"}
    Elts: []ast.Expr{
        &ast.KeyValueExpr{...},  // Name: "Alice"
        &ast.KeyValueExpr{...},  // Age: 30
    }
}

// []int{1, 2, 3}
&ast.CompositeLit{
    Type: &ast.ArrayType{...}
    Elts: []ast.Expr{
        &ast.BasicLit{Value: "1"},
        &ast.BasicLit{Value: "2"},
        &ast.BasicLit{Value: "3"},
    }
}
```

#### 8-4. `*ast.CallExpr` - 関数/メソッド呼び出し

```go
type CallExpr struct {
    Fun      Expr      // 呼び出す関数
    Lparen   token.Pos // '('の位置
    Args     []Expr    // 引数リスト
    Ellipsis token.Pos // '...'の位置（可変長引数）
    Rparen   token.Pos // ')'の位置
}
```

**用途:**
関数やメソッドの呼び出し。

**コードとAST対応:**
```go
// ソースコード
fmt.Println("Hello", "World")
dog.Bark()
len(slice)
```

**AST表現:**
```
// fmt.Println("Hello", "World")
&ast.CallExpr{
    Fun: &ast.SelectorExpr{
        X: &ast.Ident{Name: "fmt"}
        Sel: &ast.Ident{Name: "Println"}
    }
    Args: []ast.Expr{
        &ast.BasicLit{Value: `"Hello"`},
        &ast.BasicLit{Value: `"World"`},
    }
}

// dog.Bark()
&ast.CallExpr{
    Fun: &ast.SelectorExpr{
        X: &ast.Ident{Name: "dog"}
        Sel: &ast.Ident{Name: "Bark"}
    }
    Args: []ast.Expr{}
}

// len(slice)
&ast.CallExpr{
    Fun: &ast.Ident{Name: "len"}
    Args: []ast.Expr{
        &ast.Ident{Name: "slice"}
    }
}
```

#### 8-5. `*ast.UnaryExpr` - 単項演算式

```go
type UnaryExpr struct {
    OpPos token.Pos   // 演算子の位置
    Op    token.Token // 演算子（AND, NOT, など）
    X     Expr        // オペランド
}
```

**用途:**
単項演算子（`&`, `*`, `!`, `-`, `+`, など）を伴う式。

**コードとAST対応:**
```go
// ソースコード
var p = &Dog{Name: "Max"}
var v = *ptr
var neg = -42
var not = !flag
```

**AST表現:**
```
// &Dog{Name: "Max"}
&ast.UnaryExpr{
    Op: token.AND
    X: &ast.CompositeLit{
        Type: &ast.Ident{Name: "Dog"}
        ...
    }
}

// *ptr
&ast.UnaryExpr{
    Op: token.MUL  // ポインタの参照外し
    X: &ast.Ident{Name: "ptr"}
}

// -42
&ast.UnaryExpr{
    Op: token.SUB
    X: &ast.BasicLit{Value: "42"}
}

// !flag
&ast.UnaryExpr{
    Op: token.NOT
    X: &ast.Ident{Name: "flag"}
}
```

#### 8-6. `*ast.StarExpr` - ポインタ型/間接参照

```go
type StarExpr struct {
    Star token.Pos // '*'の位置
    X    Expr      // ポインタが指す型
}
```

**用途:**
- 型宣言: `*Type`（ポインタ型）
- 式: `*ptr`（ポインタの参照外し）

**コードとAST対応:**
```go
// ソースコード
func (d *Dog) Bark() {}
func GetUser() *User {}
var ptr *int
```

**AST表現:**
```
// *Dog （レシーバの型）
&ast.StarExpr{
    X: &ast.Ident{Name: "Dog"}
}

// *User （戻り値の型）
&ast.StarExpr{
    X: &ast.Ident{Name: "User"}
}

// *int （変数の型）
&ast.StarExpr{
    X: &ast.Ident{Name: "int"}
}
```

**注意:** 式としての`*ptr`（参照外し）は`*ast.UnaryExpr`で表現されます。

#### 8-7. `*ast.BasicLit` - 基本リテラル

```go
type BasicLit struct {
    ValuePos token.Pos   // リテラルの位置
    Kind     token.Token // リテラルの種類（INT, FLOAT, STRING, など）
    Value    string      // リテラル値
}
```

**用途:**
整数、浮動小数点数、文字列、文字などのリテラル。

**コードとAST対応:**
```go
// ソースコード
var x = 42
var f = 3.14
var s = "hello"
var r = 'A'
```

**AST表現:**
```
// 42
&ast.BasicLit{
    Kind: token.INT
    Value: "42"
}

// 3.14
&ast.BasicLit{
    Kind: token.FLOAT
    Value: "3.14"
}

// "hello"
&ast.BasicLit{
    Kind: token.STRING
    Value: `"hello"`  // ダブルクォート含む
}

// 'A'
&ast.BasicLit{
    Kind: token.CHAR
    Value: "'A'"  // シングルクォート含む
}
```

---

### 9. `ast.Stmt` インターフェース

文を表すインターフェース。

#### 9-1. `*ast.AssignStmt` - 代入文

```go
type AssignStmt struct {
    Lhs    []Expr      // 左辺のリスト
    TokPos token.Pos   // 演算子の位置
    Tok    token.Token // 代入演算子（ASSIGN, DEFINE, など）
    Rhs    []Expr      // 右辺のリスト
}
```

**重要なフィールド:**
- `Tok`: `token.DEFINE` (`:=`) または `token.ASSIGN` (`=`)

**コードとAST対応:**
```go
// ソースコード
x := 10
y, z := foo(), bar()
a = b + c
```

**AST表現:**
```
// x := 10
&ast.AssignStmt{
    Lhs: []ast.Expr{&ast.Ident{Name: "x"}}
    Tok: token.DEFINE  // :=
    Rhs: []ast.Expr{&ast.BasicLit{Value: "10"}}
}

// y, z := foo(), bar()
&ast.AssignStmt{
    Lhs: []ast.Expr{
        &ast.Ident{Name: "y"},
        &ast.Ident{Name: "z"},
    }
    Tok: token.DEFINE
    Rhs: []ast.Expr{
        &ast.CallExpr{Fun: &ast.Ident{Name: "foo"}},
        &ast.CallExpr{Fun: &ast.Ident{Name: "bar"}},
    }
}

// a = b + c
&ast.AssignStmt{
    Lhs: []ast.Expr{&ast.Ident{Name: "a"}}
    Tok: token.ASSIGN  // =
    Rhs: []ast.Expr{&ast.BinaryExpr{...}}  // b + c
}
```

#### 9-2. `*ast.DeclStmt` - 宣言文

```go
type DeclStmt struct {
    Decl Decl // 宣言
}
```

**用途:**
関数内での宣言（`var`, `const`, `type`）。

**コードとAST対応:**
```go
// ソースコード
func main() {
    var x = 10
    const Pi = 3.14
}
```

**AST表現:**
```
// var x = 10
&ast.DeclStmt{
    Decl: &ast.GenDecl{
        Tok: token.VAR
        Specs: []ast.Spec{
            &ast.ValueSpec{...}
        }
    }
}

// const Pi = 3.14
&ast.DeclStmt{
    Decl: &ast.GenDecl{
        Tok: token.CONST
        Specs: []ast.Spec{
            &ast.ValueSpec{...}
        }
    }
}
```

---

### 10. `*ast.FieldList` - フィールドリスト

```go
type FieldList struct {
    Opening token.Pos // '('または'{'の位置
    List    []*Field  // フィールドのリスト
    Closing token.Pos // ')'または'}'の位置
}
```

**用途:**
- 関数の引数リスト
- 関数の戻り値リスト
- 構造体のフィールドリスト
- インターフェースのメソッドリスト

#### `*ast.Field`

```go
type Field struct {
    Doc     *CommentGroup // ドキュメントコメント
    Names   []*Ident      // フィールド名のリスト（省略可）
    Type    Expr          // フィールドの型
    Tag     *BasicLit     // フィールドタグ（構造体のみ）
    Comment *CommentGroup // 行コメント
}
```

**注意:** `Names`が空の場合は埋め込み型（匿名フィールド）。

---

## Tree構造図解

### 全体的なAST階層構造

```
ast.Package (パッケージ全体)
└── Files (map[string]*ast.File)
    └── ast.File (Goファイル)
        └── Decls ([]ast.Decl)
            │
            ├── ast.FuncDecl (関数/メソッド宣言)
            │   ├── Doc (*ast.CommentGroup)
            │   ├── Recv (*ast.FieldList) - レシーバ
            │   ├── Name (*ast.Ident) - 関数名
            │   ├── Type (*ast.FuncType)
            │   │   ├── Params (*ast.FieldList) - 引数
            │   │   └── Results (*ast.FieldList) - 戻り値
            │   │       └── List ([]*ast.Field)
            │   │           └── Type (ast.Expr)
            │   └── Body (*ast.BlockStmt)
            │
            └── ast.GenDecl (汎用宣言: import, const, var, type)
                ├── Tok (token.Token) - IMPORT/CONST/VAR/TYPE
                └── Specs ([]ast.Spec)
                    │
                    ├── ast.ValueSpec (変数/定数)
                    │   ├── Names ([]*ast.Ident)
                    │   ├── Type (ast.Expr)
                    │   └── Values ([]ast.Expr)
                    │
                    ├── ast.TypeSpec (型定義)
                    │   ├── Name (*ast.Ident)
                    │   └── Type (ast.Expr)
                    │       ├── ast.StructType (構造体)
                    │       │   └── Fields (*ast.FieldList)
                    │       │
                    │       └── ast.InterfaceType (インターフェース)
                    │           └── Methods (*ast.FieldList)
                    │
                    └── ast.ImportSpec (インポート)
                        ├── Name (*ast.Ident) - エイリアス
                        └── Path (*ast.BasicLit) - パス

ast.Expr (式インターフェース) - 実装型:
├── *ast.Ident (識別子: x, foo, int)
├── *ast.SelectorExpr (セレクタ: pkg.Type, obj.Field)
│   ├── X (ast.Expr) - 左側
│   └── Sel (*ast.Ident) - 右側
├── *ast.CallExpr (関数呼び出し: fmt.Println())
│   ├── Fun (ast.Expr)
│   └── Args ([]ast.Expr)
├── *ast.CompositeLit (複合リテラル: Person{}, []int{1,2})
│   ├── Type (ast.Expr)
│   └── Elts ([]ast.Expr)
├── *ast.UnaryExpr (単項演算: &x, *ptr)
│   ├── Op (token.Token)
│   └── X (ast.Expr)
├── *ast.StarExpr (ポインタ型: *Type)
│   └── X (ast.Expr)
├── *ast.BasicLit (基本リテラル: 42, "hello")
│   ├── Kind (token.Token)
│   └── Value (string)
└── ... その他多数の実装型
```

### 関数宣言の詳細構造

**コード例:** `func NewDog(name string) *Dog`

```
*ast.FuncDecl
├── Doc (*ast.CommentGroup) - ドキュメントコメント
│   └── List ([]*ast.Comment)
│
├── Recv (*ast.FieldList) - レシーバ (関数の場合は nil)
│
├── Name (*ast.Ident)
│   └── Name: "NewDog"
│
├── Type (*ast.FuncType)
│   ├── Params (*ast.FieldList) - 引数リスト
│   │   └── List ([]*ast.Field)
│   │       └── [0] *ast.Field
│   │           ├── Names ([]*ast.Ident)
│   │           │   └── [0] *ast.Ident {Name: "name"}
│   │           └── Type (ast.Expr)
│   │               └── *ast.Ident {Name: "string"}
│   │
│   └── Results (*ast.FieldList) - 戻り値リスト
│       └── List ([]*ast.Field)
│           └── [0] *ast.Field
│               ├── Names ([]*ast.Ident) - nil (名前なし戻り値)
│               └── Type (ast.Expr)
│                   └── *ast.StarExpr
│                       └── X (*ast.Ident {Name: "Dog"})
│
└── Body (*ast.BlockStmt) - 関数本体
    └── List ([]ast.Stmt)
```

### メソッド宣言の詳細構造

**コード例:**
```go
func (d *Dog) Bark() string {
    return "Woof!"
}
```

**AST構造:**
```
*ast.FuncDecl
├── Doc (*ast.CommentGroup) - ドキュメントコメント
│
├── Recv (*ast.FieldList) - レシーバ (メソッドの場合は非nil)
│   └── List ([]*ast.Field)
│       └── [0] *ast.Field
│           ├── Names ([]*ast.Ident)
│           │   └── [0] *ast.Ident {Name: "d"}
│           └── Type (ast.Expr)
│               └── *ast.StarExpr
│                   └── X (*ast.Ident {Name: "Dog"})
│
├── Name (*ast.Ident)
│   └── Name: "Bark"
│
├── Type (*ast.FuncType)
│   ├── Params (*ast.FieldList) - nil (引数なし)
│   │
│   └── Results (*ast.FieldList)
│       └── List ([]*ast.Field)
│           └── [0] *ast.Field
│               └── Type (*ast.Ident {Name: "string"})
│
└── Body (*ast.BlockStmt) - 関数本体 {}
    └── List ([]ast.Stmt) - {} 内の文のリスト
        └── [0] *ast.ReturnStmt
            └── Results ([]ast.Expr)
                └── [0] *ast.BasicLit {Kind: STRING, Value: "\"Woof!\""}
```

### 構造体定義の詳細構造

**コード例:**
```go
type Dog struct {
    Name string
    Age  int
}
```

**AST構造:**
```
*ast.GenDecl
├── Tok: token.TYPE
└── Specs ([]ast.Spec)
    └── [0] *ast.TypeSpec
        ├── Name (*ast.Ident)
        │   └── Name: "Dog"
        │
        └── Type (ast.Expr)
            └── *ast.StructType
                └── Fields (*ast.FieldList)
                    └── List ([]*ast.Field)
                        ├── [0] *ast.Field
                        │   ├── Names ([]*ast.Ident)
                        │   │   └── [0] *ast.Ident {Name: "Name"}
                        │   └── Type (*ast.Ident {Name: "string"})
                        │
                        └── [1] *ast.Field
                            ├── Names ([]*ast.Ident)
                            │   └── [0] *ast.Ident {Name: "Age"}
                            └── Type (*ast.Ident {Name: "int"})
```

### 変数宣言の詳細構造（複数パターン）

#### パターン1: 構造体リテラル

**コード例:** `var x = pkg.Type{}`

```
*ast.GenDecl
├── Tok: token.VAR
└── Specs ([]ast.Spec)
    └── [0] *ast.ValueSpec
        ├── Names ([]*ast.Ident)
        │   └── [0] *ast.Ident {Name: "x"}
        │
        └── Values ([]ast.Expr)
            └── [0] *ast.CompositeLit
                ├── Type (ast.Expr)
                │   └── *ast.SelectorExpr
                │       ├── X (*ast.Ident {Name: "pkg"})
                │       └── Sel (*ast.Ident {Name: "Type"})
                └── Elts ([]ast.Expr) - 空
```

#### パターン2: 関数呼び出し

**コード例:** `var x = pkg.Func()`

```
*ast.GenDecl
├── Tok: token.VAR
└── Specs ([]ast.Spec)
    └── [0] *ast.ValueSpec
        ├── Names ([]*ast.Ident)
        │   └── [0] *ast.Ident {Name: "x"}
        │
        └── Values ([]ast.Expr)
            └── [0] *ast.CallExpr
                ├── Fun (ast.Expr)
                │   └── *ast.SelectorExpr
                │       ├── X (*ast.Ident {Name: "pkg"})
                │       └── Sel (*ast.Ident {Name: "Func"})
                └── Args ([]ast.Expr) - 空
```

#### パターン3: ポインタ型

**コード例:** `var x = &pkg.Type{}`

```
*ast.GenDecl
├── Tok: token.VAR
└── Specs ([]ast.Spec)
    └── [0] *ast.ValueSpec
        ├── Names ([]*ast.Ident)
        │   └── [0] *ast.Ident {Name: "x"}
        │
        └── Values ([]ast.Expr)
            └── [0] *ast.UnaryExpr
                ├── Op: token.AND (&)
                └── X (ast.Expr)
                    └── *ast.CompositeLit
                        ├── Type (*ast.SelectorExpr)
                        │   ├── X (*ast.Ident {Name: "pkg"})
                        │   └── Sel (*ast.Ident {Name: "Type"})
                        └── Elts ([]ast.Expr) - 空
```

### セレクタ式のパターン

#### パターン1: パッケージ.関数
**コード例:** `pkg.Func`
```
*ast.SelectorExpr
├── X (ast.Expr)
│   └── *ast.Ident {Name: "pkg"}
└── Sel (*ast.Ident {Name: "Func"})
```

#### パターン2: 変数.メソッド
**コード例:** `obj.Method`
```
*ast.SelectorExpr
├── X (ast.Expr)
│   └── *ast.Ident {Name: "obj"}
└── Sel (*ast.Ident {Name: "Method"})
```

#### パターン3: パッケージ.型
**コード例:** `pkg.Type`
```
*ast.SelectorExpr
├── X (ast.Expr)
│   └── *ast.Ident {Name: "pkg"}
└── Sel (*ast.Ident {Name: "Type"})
```

#### パターン4: メソッドチェーン
**コード例:** `obj.GetData().Process()`
```
*ast.CallExpr (外側)
├── Fun (ast.Expr)
│   └── *ast.SelectorExpr
│       ├── X (ast.Expr)
│       │   └── *ast.CallExpr (内側)
│       │       ├── Fun (*ast.SelectorExpr)
│       │       │   ├── X (*ast.Ident {Name: "obj"})
│       │       │   └── Sel (*ast.Ident {Name: "GetData"})
│       │       └── Args ([]ast.Expr) - 空
│       └── Sel (*ast.Ident {Name: "Process"})
└── Args ([]ast.Expr) - 空
```

### gonsoleでのAST処理フロー

```
1. プロジェクト起動
   │
   └─> utils.AnalyzeGoAst() - 全Goファイルをパース
       │
       └─> map[string]*ast.Package を生成
           │
           └─> completer.NewCandidates() - 補完候補の構築開始
               │
               └─> パッケージごとに処理
                   │
                   └─> processPackageAst()
                       │
                       └─> ファイルごとに処理
                           │
                           └─> processFileAst()
                               │
                               └─> file.Decls をイテレート
                                   │
                                   ├─> [ast.FuncDecl の場合]
                                   │   │
                                   │   ├─> Recv != nil → processMethodDecl()
                                   │   │                 ├─> メソッド候補に追加
                                   │   │                 └─> レシーバ型を記録
                                   │   │
                                   │   └─> Recv == nil → processFuncDecl()
                                   │                     └─> 関数候補に追加
                                   │
                                   └─> [ast.GenDecl の場合]
                                       │
                                       ├─> Tok == token.VAR
                                       │   └─> processVarDecl()
                                       │       └─> 変数候補に追加
                                       │
                                       ├─> Tok == token.CONST
                                       │   └─> processConstDecl()
                                       │       └─> 定数候補に追加
                                       │
                                       └─> Tok == token.TYPE
                                           └─> processTypeDecl()
                                               │
                                               ├─> Type == *ast.StructType
                                               │   └─> 構造体候補に追加
                                               │       └─> フィールド名を記録
                                               │
                                               └─> Type == *ast.InterfaceType
                                                   └─> インターフェース候補に追加
                                                       └─> メソッド名を記録

2. 補完候補の構築完了
   └─> candidates 構造体に以下を格納:
       ├─> 関数リスト
       ├─> メソッドリスト (型ごと)
       ├─> 変数リスト
       ├─> 定数リスト
       ├─> 型リスト (構造体/インターフェース)
       └─> パッケージ情報
```

---

## まとめ

gonsoleで使用している主要なAST型：

| 型 | 用途 | 抽出する情報 |
|---|---|---|
| `*ast.Package` | パッケージ全体 | パッケージ名、ファイルリスト |
| `*ast.File` | Goファイル | 宣言リスト |
| `*ast.FuncDecl` | 関数/メソッド宣言 | 関数名、レシーバ、戻り値型、ドキュメント |
| `*ast.GenDecl` | 汎用宣言 | 変数、定数、型の宣言 |
| `*ast.ValueSpec` | 変数/定数の詳細 | 名前、型、初期値 |
| `*ast.TypeSpec` | 型定義の詳細 | 型名、構造体/インターフェース |
| `*ast.StructType` | 構造体定義 | フィールド名リスト |
| `*ast.InterfaceType` | インターフェース定義 | メソッド名リスト |
| `*ast.SelectorExpr` | セレクタ式 | パッケージ名、型名、関数名 |
| `*ast.CallExpr` | 関数呼び出し | 呼び出し元、戻り値の型推論 |
| `*ast.CompositeLit` | 複合リテラル | 構造体リテラルの型 |
| `*ast.UnaryExpr` | 単項演算 | ポインタ型の検出 |
| `*ast.Ident` | 識別子 | 名前 |

これらの型を組み合わせることで、Goコードの構造を完全に解析し、補完候補の構築や型推論を実現しています。
