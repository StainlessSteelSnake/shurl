// Пакет main
package main

import (
	"go/ast"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/analysis/passes/assign"
	"golang.org/x/tools/go/analysis/passes/bools"
	"golang.org/x/tools/go/analysis/passes/copylock"
	"golang.org/x/tools/go/analysis/passes/defers"
	"golang.org/x/tools/go/analysis/passes/httpresponse"
	"golang.org/x/tools/go/analysis/passes/loopclosure"
	"golang.org/x/tools/go/analysis/passes/lostcancel"
	"golang.org/x/tools/go/analysis/passes/nilfunc"
	"golang.org/x/tools/go/analysis/passes/printf"
	"golang.org/x/tools/go/analysis/passes/shadow"
	"golang.org/x/tools/go/analysis/passes/structtag"
	"golang.org/x/tools/go/analysis/passes/tests"
	"golang.org/x/tools/go/analysis/passes/timeformat"
	"golang.org/x/tools/go/analysis/passes/unmarshal"
	"golang.org/x/tools/go/analysis/passes/unreachable"
	"golang.org/x/tools/go/analysis/passes/unusedresult"
	"golang.org/x/tools/go/analysis/passes/unusedwrite"
	"honnef.co/go/tools/staticcheck"
)

var osExitAnalyzer = &analysis.Analyzer{
	Name: "osexitcheck",
	Doc:  "запрет на использование прямого вызова os.Exit",
	Run:  run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	for _, file := range pass.Files {
		if file.Name.Name != "main" {
			continue
		}

		mainFunctionIsFound := false

		// функцией ast.Inspect проходим по всем узлам AST
		ast.Inspect(file, func(node ast.Node) bool {
			if x, ok := node.(*ast.FuncDecl); ok {
				if x.Name.Name == "main" {
					mainFunctionIsFound = true
					return true
				} else if mainFunctionIsFound {
					return false
				} else {
					return true
				}
			}

			if x, ok := node.(*ast.SelectorExpr); ok && mainFunctionIsFound {
				if ident, ok := x.X.(*ast.Ident); ok {
					if ident.Name == "os" && x.Sel.Name == "Exit" {
						pass.Reportf(x.Sel.Pos(), "в теле функции main нельзя вызывать os.Exit")
						return false
					}
				}
			}

			return true
		})
	}
	return nil, nil
}

func main() {
	checkers := []*analysis.Analyzer{
		assign.Analyzer,
		bools.Analyzer,
		copylock.Analyzer,
		defers.Analyzer,
		httpresponse.Analyzer,
		loopclosure.Analyzer,
		lostcancel.Analyzer,
		nilfunc.Analyzer,
		structtag.Analyzer,
		tests.Analyzer,
		timeformat.Analyzer,
		unmarshal.Analyzer,
		unreachable.Analyzer,
		unusedresult.Analyzer,
		unusedwrite.Analyzer,
		printf.Analyzer,
		shadow.Analyzer,
		osExitAnalyzer,
	}

	staticcheckers := map[string]bool{
		"S1":  false,
		"ST1": false,
		"QF":  false,
	}

	for _, a := range staticcheck.Analyzers {
		if strings.HasSuffix(a.Analyzer.Name, "SA") {
			//checkers = append(checkers, a.Analyzer)
		}
		if strings.HasSuffix(a.Analyzer.Name, "S1") && !staticcheckers["S1"] {
			//checkers = append(checkers, a.Analyzer)
			staticcheckers["S1"] = true
		}
		if strings.HasSuffix(a.Analyzer.Name, "ST1") && !staticcheckers["ST1"] {
			//checkers = append(checkers, a.Analyzer)
			staticcheckers["ST1"] = true
		}
		if strings.HasSuffix(a.Analyzer.Name, "QF") && !staticcheckers["QF"] {
			//checkers = append(checkers, a.Analyzer)
			staticcheckers["QF"] = true
		}
	}

	multichecker.Main(
		checkers...,
	)
}
