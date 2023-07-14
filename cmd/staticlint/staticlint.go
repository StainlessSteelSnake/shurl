// Пакет main содержит набор статических анализаторов проекта (линтеров).
// Проверки запускаются командой go run ./cmd/staticlint [путь к файлам для проверки].
// Например, для проверки всего проекта следует указать команду "go run ./cmd/staticlint ./...".

//go:build ignore

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
	"honnef.co/go/tools/staticcheck"

	"github.com/kisielk/errcheck/errcheck"
	"github.com/timakin/bodyclose/passes/bodyclose"
)

// osExitAnalyzer представляет содержит ссылку на линтер, проверяющий наличие вызова os.Exit в теле функции main.
var osExitAnalyzer = &analysis.Analyzer{
	Name: "osexitcheck",
	Doc:  "запрет на использование прямого вызова os.Exit",
	Run:  osExitCheckRun,
}

// osExitAnalyzer проверяет наличие вызова os.Exit в теле функции main.
func osExitCheckRun(pass *analysis.Pass) (interface{}, error) {
	// Перебираем все файлы, для которых построено абстрактное синтаксическое дерево.
	for _, file := range pass.Files {

		// Игнорируем все файлы, кроме main.
		if file.Name.Name != "main" {
			continue
		}

		// mainFunctionIsFound содержит признак того, что мы нашли узел AST, отвечающий за функцию main.
		mainFunctionIsFound := false

		// Проходим по всем узлам построенного AST.
		ast.Inspect(file, func(node ast.Node) bool {

			// Проверяем что рассматриваемый узел - объявление функции main.
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

			//
			if x, ok := node.(*ast.SelectorExpr); ok && mainFunctionIsFound {
				if ident, ok := x.X.(*ast.Ident); ok {
					if ident.Name == "os" && x.Sel.Name == "Exit" {
						pass.Reportf(x.Pos(), "в теле функции main нельзя вызывать os.Exit")
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
		assign.Analyzer,       // Проверка на неиспользуемые присвоения значений
		bools.Analyzer,        // Проверка на частые ошибки при использовании булевых операций
		copylock.Analyzer,     // Проверка, что блокировки не передаются по значению
		defers.Analyzer,       // Проверка на частые ошибки при использовании оператора defer
		httpresponse.Analyzer, // Проверка на частые ошибки при формировании HTTP-ответа
		loopclosure.Analyzer,  // Проверка на ошибки при замыкании переменных из цикла во вложенных функциях
		lostcancel.Analyzer,   // Проверка на неиспользуемые вызовы функции cancel для созданных контекстов
		nilfunc.Analyzer,      // Проверка на бесполезные проверки на равенство nil
		structtag.Analyzer,    // Проверка на правильность указания тэгов для полей структур
		tests.Analyzer,        // Проверка на типовые ошибки в тестах и примерах
		timeformat.Analyzer,   // Проверка на ошибки в форматах дат при использовании функций time.Format и time.Parse
		unmarshal.Analyzer,    // Проверка на передачу не-указателей и не-интерфейсов в функции unmarshal и decode
		unreachable.Analyzer,  // Проверка на наличие фрагментов кода, которые никогда не будут выполнены
		unusedresult.Analyzer, // Проверка на наличие неиспользуемых значений, возвращаемых при вызове функций
		printf.Analyzer,       // Проверка соответствия формата переданных переменных шаблону, указанному при вызове функции Printf
		shadow.Analyzer,       // Проверка на затенение переменных во вложенных блоках

		bodyclose.Analyzer, // Проверка закрытия тела ответа на HTTP-запрос
		errcheck.Analyzer,  // Проверка обработки возвращаемых функциями ошибок

		osExitAnalyzer, // Проверка вызова функции os.Exit в функции main пакета main
	}

	// Формируем список префиксов для классов анализаторов из пакета staticcheck.io, кроме класса SA.
	staticcheckers := map[string]bool{
		"S1":  false,
		"ST1": false,
		"QF":  false,
	}

	// Добавляем все анализаторы класса SA и по одному анализатору остальных классов из пакета staticcheck.io
	// к списку используемых.
	for _, a := range staticcheck.Analyzers {
		if strings.HasSuffix(a.Analyzer.Name, "SA") {
			checkers = append(checkers, a.Analyzer)
		}
		if strings.HasSuffix(a.Analyzer.Name, "S1") && !staticcheckers["S1"] {
			checkers = append(checkers, a.Analyzer)
			staticcheckers["S1"] = true
		}
		if strings.HasSuffix(a.Analyzer.Name, "ST1") && !staticcheckers["ST1"] {
			checkers = append(checkers, a.Analyzer)
			staticcheckers["ST1"] = true
		}
		if strings.HasSuffix(a.Analyzer.Name, "QF") && !staticcheckers["QF"] {
			checkers = append(checkers, a.Analyzer)
			staticcheckers["QF"] = true
		}
	}

	// Выполняем запуск анализаторов из сформированного списка.
	multichecker.Main(
		checkers...,
	)
}
