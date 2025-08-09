// Package staticlint provides a custom analyzer that prohibits direct calls to os.Exit in main function of main package.
package main

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// OsExitAnalyzer is an analyzer that prohibits direct calls to os.Exit in main function of main package.
var OsExitAnalyzer = &analysis.Analyzer{
	Name:     "osexit",
	Doc:      "prohibits direct calls to os.Exit in main function of main package",
	Run:      runOsExitCheck,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

// runOsExitCheck performs the analysis to detect os.Exit calls in main function of main package.
func runOsExitCheck(pass *analysis.Pass) (interface{}, error) {
	// Проверяем, что это пакет main
	if pass.Pkg.Name() != "main" {
		return nil, nil
	}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// Ищем функции с именем main
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	inspect.Preorder(nodeFilter, func(node ast.Node) {
		funcDecl := node.(*ast.FuncDecl)

		// Проверяем, что это функция main
		if funcDecl.Name.Name != "main" {
			return
		}

		// Обходим тело функции main в поисках вызовов os.Exit
		ast.Inspect(funcDecl.Body, func(n ast.Node) bool {
			callExpr, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}

			// Проверяем, является ли это селектором (package.Function)
			selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}

			// Проверяем, что это вызов метода Exit
			if selExpr.Sel.Name != "Exit" {
				return true
			}

			// Проверяем, что это из пакета os
			if ident, ok := selExpr.X.(*ast.Ident); ok {
				// Получаем информацию о типе пакета
				if obj := pass.TypesInfo.Uses[ident]; obj != nil {
					if pkgName, ok := obj.(*types.PkgName); ok {
						if pkgName.Imported().Path() == "os" {
							pass.Reportf(
								callExpr.Pos(),
								"avoid direct os.Exit call in main function of main package",
							)
						}
					}
				}
			}

			return true
		})
	})

	return nil, nil
}
