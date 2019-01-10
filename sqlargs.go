package sqlargs

import (
	"errors"
	"go/ast"
	"go/constant"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const Doc = `check sql query strings for correctness

The sqlargs analyser checks the parameters passed to sql queries
and the actual number of parameters written in the query string
and reports any mismatches.

This is a common occurence when updating a sql query to add/remove
a column.`

var Analyzer = &analysis.Analyzer{
	Name:             "sqlargs",
	Doc:              Doc,
	Run:              run,
	Requires:         []*analysis.Analyzer{inspect.Analyzer},
	RunDespiteErrors: true,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// We ignore packages that do not import database/sql.
	hasImport := false
	for _, imp := range pass.Pkg.Imports() {
		if imp.Path() == "database/sql" {
			hasImport = true
			break
		}
	}
	if !hasImport {
		return nil, nil
	}

	inspect := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	// We filter only function calls.
	nodeFilter := []ast.Node{
		(*ast.CallExpr)(nil),
	}

	inspect.Preorder(nodeFilter, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		// Now we need to find expressions like these in the source code.
		// db.Exec(`INSERT INTO <> (foo, bar) VALUES ($1, $2)`, param1, param2)

		// A CallExpr has 2 parts - Fun and Args.
		// A Fun can either be an Ident (Fun()) or a SelectorExpr (foo.Fun()).
		// Since we are looking for patterns like db.Exec, we need to filter only SelectorExpr
		// We will ignore dot imported functions.
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}

		// A SelectorExpr(db.Exec) has 2 parts - X (db) and Sel (Exec/Query/QueryRow).
		// Now that we are inside the SelectorExpr, we need to verify 2 things -
		// 1. The function name is Exec, Query or QueryRow; because that is what we are interested in.
		// 2. The type of the selector is sql.DB, sql.Tx or sql.Stmt.
		if !isProperSelExpr(sel, pass.TypesInfo) {
			return
		}
		// Length of args has to be minimum of 1 because we only take Exec, Query or QueryRow;
		// all of which have atleast 1 argument. But still writing a sanity check.
		if len(call.Args) == 0 {
			return
		}

		sql, args, err := extractSQL(call.Args, pass.TypesInfo.Types)
		if err != nil {
			return
		}

		analyzeQuery(sql, args, call, pass)
	})

	return nil, nil
}

// extractSQL extracts the SQL statement and args
func extractSQL(args []ast.Expr, types map[ast.Expr]types.TypeAndValue) (string, []ast.Expr, error) {
	for i, arg := range args {
		tv, ok := types[arg]
		if !ok || tv.Value == nil {
			continue
		}
		// we assume first string type arg is going to be the sql statement
		// and everything after it as sql args
		if tv.Type.String() == "string" {
			return constant.StringVal(tv.Value), args[i+1:], nil
		}
	}
	return "", []ast.Expr{}, errors.New("No SQL statement found")
}

// contains checks if a string is in a string slice
func contains(needle string, hay []string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}

var validFuncs = []string{
	"Exec", "QueryRow", "Query", "ExecContext", "QueryRowContext", "QueryContext",
	"MustExec", "Queryx", "QueryRowx", "NamedQuery", "NamedExec", "QueryxContext",
	"QueryRowxContext", "NamedQueryContext", "NamedExecContext",
}

var validPackages = []string{"database/sql", "github.com/jmoiron/sqlx"}

var validTypeNames = []string{"DB", "Tx", "Stmt", "NamedStmt"}

func isProperSelExpr(sel *ast.SelectorExpr, typesInfo *types.Info) bool {
	// Only accept function calls for Exec, QueryRow and Query
	fnName := sel.Sel.Name
	if !contains(fnName, validFuncs) {
		return false
	}
	// Get the type info of X of the selector.
	typ, ok := typesInfo.Types[sel.X]
	if !ok {
		return false
	}
	ptr, ok := typ.Type.(*types.Pointer)
	if !ok {
		return false
	}
	n := ptr.Elem().(*types.Named)
	// remove vendor path prefix
	// eg. github.com/godwhoa/upboat/vendor/<pkg> -> <pkg>
	pkgPath := stripVendor(n.Obj().Pkg().Path())
	if !contains(pkgPath, validPackages) {
		return false
	}
	name := n.Obj().Name()
	// Only accept sql.DB, sql.Tx, sql.Stmt types.
	if !contains(name, validTypeNames) {
		return false
	}
	return true
}

// stripVendor strips out the vendor path prefix
func stripVendor(pkgPath string) string {
	idx := strings.LastIndex(pkgPath, "vendor/")
	if idx < 0 {
		return pkgPath
	}
	// len("vendor/") == 7
	return pkgPath[idx+7:]
}
