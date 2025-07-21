package executor

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

type Executor struct {
	interpreter *interp.Interpreter
}

func NewExecutor(importPaths []string) *Executor {
	// 絶対パスをGoPathに設定
	interpreter := interp.New(interp.Options{
		GoPath: ".",
	})

	// 標準ライブラリをロード
	if err := interpreter.Use(stdlib.Symbols); err != nil {
		log.Fatalf("Failed to load stdlib: %v", err)
	}
	e := &Executor{
		interpreter: interpreter,
	}

	srcs, graph, err := e.CombineGoFilesByPackage()
	if err != nil {
		log.Fatalf("Failed to combine Go files: %v", err)
	}
	// import文がダブっている場合、１つにまとめる
	// 評価するファイル順を依存関係から制御するためにトポロジカルソ	ートを行う
	sorted := topologicalSortSrcs(srcs, graph)
	for _, src := range sorted {
		_, err := e.interpreter.Eval(src)
		if err != nil {
			fmt.Printf("Evaluating source (prefix) Error: %.100s...\n", err)
		}
	}

	return e
}

func (e *Executor) Execute(input string) {
	// 入力を実行する
	result, err := e.interpreter.Eval(input)
	if err != nil {
		log.Printf("Error executing input: %v", err)
		return
	}
	fmt.Println(result)
}

type pkgSrc map[string]string

// ディレクトリごとに全てのGoファイルを結合する関数
func (e *Executor) CombineGoFilesByPackage() (pkgSrc, pkgDependencyGraph, error) {
	pkgSources := make(map[string][]string)
	pkgImports := make(map[string][]*ast.ImportSpec)
	graph := make(pkgDependencyGraph)
	fset := token.NewFileSet()
	mod, err := getGoModPath("go.mod")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get go.mod path: %w", err)
	}

	err = filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}

		// Parse only the package clause
		file, err := parser.ParseFile(fset, path, src, parser.AllErrors)
		if err != nil {
			return fmt.Errorf("failed to parse package from %s: %w", path, err)
		}

		pkgName := fmt.Sprintf("%s/%s", mod, file.Name.Name)
		if strings.Contains(path, "vendor") {
			// vendor dirだったら相応の github.com/xxx/yyy/pkg 形式にする
			vendorPath := path[strings.Index(path, "vendor/")+len("vendor/"):]
			vendorPathParts := strings.Split(vendorPath, "/")
			// vendorディレクトリの最後の部分はパッケージ名ではないので除外
			vendorPath = strings.Join(vendorPathParts[:len(vendorPathParts)-1], "/")
			pkgName = fmt.Sprintf("%s/%s", vendorPath, file.Name.Name)
		}
		err = addPkgDependencyGraph(file, pkgName, graph)
		if err != nil {
			return fmt.Errorf("failed to add package dependency graph for %s: %w", path, err)
		}
		srcStr, imports, err := e.removePackageAndImport(file)
		if err != nil {
			return fmt.Errorf("failed to remove package and imports from %s: %w", path, err)
		}
		pkgSources[pkgName] = append(pkgSources[pkgName], srcStr)

		existImportName := make([]string, len(pkgImports[pkgName]))
		for _, imp := range pkgImports[pkgName] {
			existImportName = append(existImportName, imp.Path.Value)
		}
		for _, imp := range imports {
			if !slices.Contains(existImportName, imp.Path.Value) {
				pkgImports[pkgName] = append(pkgImports[pkgName], imp)
			}
		}

		return nil
	})

	if err != nil {
		return nil, nil, err
	}

	// 文字列で1パッケージ＝1エントリにして返す
	var combined = make(pkgSrc)
	for pkg, sources := range pkgSources {
		var combinedSrc strings.Builder
		for _, src := range sources {
			combinedSrc.WriteString(src)
			combinedSrc.WriteString("\n")
		}
		var src string
		if strings.Contains(pkg, mod) {
			src = fmt.Sprintf("package %s\n\n%s", strings.ReplaceAll(pkg, mod+"/", ""), combinedSrc.String())
			fmt.Println(src)
		} else {
			if strings.Contains(pkg, "github.com") || strings.Contains(pkg, "golang.org") {
				pkgParts := strings.Split(pkg, "/")
				fmt.Println(pkgParts[len(pkgParts)-1])
				src = fmt.Sprintf("package %s\n\n%s", pkgParts[len(pkgParts)-1], combinedSrc.String())
			}
		}

		file, err := parser.ParseFile(fset, pkg, src, parser.AllErrors)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse combined source for package %s: %w", pkg, err)
		}
		// インポート宣言を追加
		if imports, ok := pkgImports[pkg]; ok {
			genDecl := &ast.GenDecl{
				Tok:   token.IMPORT,
				Specs: make([]ast.Spec, len(imports)),
			}
			for i, imp := range imports {
				genDecl.Specs[i] = imp
			}
			file.Decls = append(
				[]ast.Decl{genDecl}, // インポート宣言を最初に追加
				file.Decls...,
			)
		}
		var buf bytes.Buffer
		err = format.Node(&buf, token.NewFileSet(), file)
		if err != nil {
			return nil, nil, err
		}

		combined[pkg] = buf.String()
	}
	return combined, graph, nil
}

// import宣言とpackage宣言を除去する関数
func (e *Executor) removePackageAndImport(file *ast.File) (string, []*ast.ImportSpec, error) {
	mod, err := getGoModPath("go.mod")
	imports := make([]*ast.ImportSpec, 0)
	if err != nil {
		return "", nil, fmt.Errorf("failed to get go.mod path: %w", err)
	}
	// 実際にインポート宣言を削除する
	var decls []ast.Decl
	for _, decl := range file.Decls {
		if genDecl, ok := decl.(*ast.GenDecl); ok && genDecl.Tok == token.IMPORT {
			for _, spec := range genDecl.Specs {
				if impSpec, ok := spec.(*ast.ImportSpec); ok {
					if !strings.Contains(impSpec.Path.Value, mod) {
						// decls = append(decls, decl) // modパスを含まないインポート宣言は残す
						imports = append(imports, impSpec)
					}
				}
			}
			continue
		}
		decls = append(decls, decl) // 他の宣言はそのまま残す
	}
	file.Decls = decls
	// fileを文字列に変換
	var buf bytes.Buffer
	err = format.Node(&buf, token.NewFileSet(), file)
	if err != nil {
		return "", nil, err
	}
	src := buf.String()
	// package宣言を除去
	src = strings.ReplaceAll(src, "package "+file.Name.Name+"\n", "")
	return src, imports, nil
}

type pkgDependencyGraph map[string][]string

func addPkgDependencyGraph(file *ast.File, pkgName string, graph pkgDependencyGraph) error {
	if _, exists := graph[pkgName]; !exists {
		graph[pkgName] = []string{}
	}

	for _, imp := range file.Imports {
		impName := strings.Trim(imp.Path.Value, `"`)
		if !slices.Contains(graph[pkgName], impName) {
			graph[pkgName] = append(graph[pkgName], impName)
		}
	}
	return nil
}

func topologicalSortSrcs(pkgSource pkgSrc, graph pkgDependencyGraph) []string {
	n := len(graph)
	indegree := make(map[string]int, n)
	for node := range graph {
		indegree[node] = 0
	}
	for _, node := range graph {
		for _, dep := range node {
			indegree[dep]++
		}
	}
	queue := make([]string, 0, n)
	for node, degree := range indegree {
		if degree == 0 {
			queue = append(queue, node)
		}
	}
	var result []string
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)
		for _, dep := range graph[node] {
			indegree[dep]--
			if indegree[dep] == 0 {
				queue = append(queue, dep)
			}
		}
	}

	var sortedSrcs []string
	slices.Reverse(result) // Reverse to get the correct order
	for _, node := range result {
		if src, exists := pkgSource[node]; exists {
			sortedSrcs = append(sortedSrcs, src)
		} else {
			log.Printf("Warning: No source found for package %s", node)
		}
	}

	return sortedSrcs
}
