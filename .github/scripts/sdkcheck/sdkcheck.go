// Command sdkcheck reports universal Go safety, naming, and test rules that
// golangci-lint (with the repository .golangci.yml) cannot express. It is
// syntactic (no type checking), so its findings ask for judgment, not blind
// fixes.
//
// Usage:
//
//	GOWORK=off go run sdkcheck.go [path ...]
//
// Each path is a .go file or a directory (walked recursively; a trailing
// "/..." is accepted). The default is ".". Findings print as
//
//	file:line:col: rule: message (guide link)
//
// The exit status is 1 when anything is reported.
//
// Silence a deliberate deviation with a comment on the same line or the line
// above:
//
//	// sdkcheck:ignore constructor-stutter keeps legacy constructor name
package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// ruleDocs maps each rule to its reference guide URL.
var ruleDocs = map[string]string{
	"error-concrete-return": "https://go.dev/doc/faq#nil_error",
	"error-log-and-return":  "https://go.dev/wiki/CodeReviewComments#handle-errors",
	"library-panic":         "https://go.dev/doc/effective_go#panic",
	"constructor-stutter":   "https://go.dev/doc/effective_go#package-names",
	"canonical-method":      "https://go.dev/doc/effective_go#interface-names",
	"test-message":          "https://go.dev/wiki/TestComments#useful-test-failures",
}

type finding struct {
	pos  token.Position
	rule string
	msg  string
}

type checker struct {
	fset     *token.FileSet
	findings []finding
	// ignores maps each file to its line -> suppressed rules.
	ignores map[string]map[int][]string
}

type fileCtx struct {
	*checker
	file   *ast.File
	pkg    string
	isTest bool
	isMain bool
}

var logMethods = map[string]bool{
	"Print": true, "Printf": true, "Println": true,
	"Error": true, "Errorf": true, "ErrorContext": true,
	"Warn": true, "Warnf": true, "WarnContext": true,
	"Info": true, "Infof": true, "Log": true, "Logf": true,
}

func main() {
	flag.Parse()
	paths := flag.Args()
	if len(paths) == 0 {
		paths = []string{"."}
	}
	c, err := check(paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sdkcheck: %v\n", err)
		os.Exit(2)
	}
	sort.Slice(c.findings, func(i, j int) bool {
		a, b := c.findings[i].pos, c.findings[j].pos
		if a.Filename != b.Filename {
			return a.Filename < b.Filename
		}
		return a.Line < b.Line
	})
	for _, f := range c.findings {
		fmt.Printf("%s: %s: %s (%s)\n", f.pos, f.rule, f.msg, ruleDocs[f.rule])
	}
	fmt.Fprintf(os.Stderr, "sdkcheck: %d finding(s)\n", len(c.findings))
	if len(c.findings) > 0 {
		os.Exit(1)
	}
}

// check runs every rule over paths.
func check(paths []string) (*checker, error) {
	c := &checker{
		fset:    token.NewFileSet(),
		ignores: map[string]map[int][]string{},
	}
	for _, p := range paths {
		if err := c.walk(strings.TrimSuffix(p, "/...")); err != nil {
			return nil, err
		}
	}
	return c, nil
}

func (c *checker) walk(root string) error {
	info, err := os.Stat(root)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return c.checkFile(root)
	}
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		name := d.Name()
		if d.IsDir() {
			if path != root && (strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "vendor" || name == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(name, ".go") {
			return c.checkFile(path)
		}
		return nil
	})
}

func (c *checker) checkFile(path string) error {
	f, err := parser.ParseFile(c.fset, path, nil, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return err
	}
	if ast.IsGenerated(f) {
		return nil
	}
	ignores := map[int][]string{}
	for _, cg := range f.Comments {
		for _, cm := range cg.List {
			rules := parseIgnore(cm.Text)
			if len(rules) == 0 {
				continue
			}
			line := c.fset.Position(cm.Pos()).Line
			ignores[line] = append(ignores[line], rules...)
			ignores[line+1] = append(ignores[line+1], rules...)
		}
	}
	c.ignores[path] = ignores
	fc := &fileCtx{
		checker: c,
		file:    f,
		pkg:     f.Name.Name,
		isTest:  strings.HasSuffix(path, "_test.go"),
		isMain:  f.Name.Name == "main",
	}
	fc.run()
	return nil
}

// isIgnored reports whether an sdkcheck:ignore comment suppresses rule at p.
func (c *checker) isIgnored(p token.Pos, rule string) bool {
	pos := c.fset.Position(p)
	return slices.Contains(c.ignores[pos.Filename][pos.Line], rule)
}

// reportAt records a finding at p unless an sdkcheck:ignore comment for rule is
// on its line or the line above.
func (c *checker) reportAt(p token.Pos, rule, format string, args ...any) {
	if c.isIgnored(p, rule) {
		return
	}
	pos := c.fset.Position(p)
	c.findings = append(c.findings, finding{pos, rule, fmt.Sprintf(format, args...)})
}

func (fc *fileCtx) run() {
	for _, decl := range fc.file.Decls {
		if fd, ok := decl.(*ast.FuncDecl); ok {
			fc.checkFuncDecl(fd)
		}
	}
	ast.Inspect(fc.file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.IfStmt:
			fc.checkLogAndReturn(n)
		case *ast.CallExpr:
			fc.checkTestMessage(n)
		}
		return true
	})
}

func (fc *fileCtx) report(n ast.Node, rule, format string, args ...any) {
	fc.reportAt(n.Pos(), rule, format, args...)
}

func (fc *fileCtx) checkFuncDecl(d *ast.FuncDecl) {
	name := d.Name.Name

	if d.Recv != nil && (name == "ToString" || name == "toString") {
		fc.report(d.Name, "canonical-method", "name the string conversion String() so fmt and %%v use it")
	}

	// widget.NewWidget should be widget.New. revive's exported rule only
	// catches names that start with the package name.
	if d.Recv == nil && !fc.isMain && !fc.isTest && len(fc.pkg) > 1 && strings.HasPrefix(name, "New") &&
		strings.HasPrefix(strings.ToLower(name[len("New"):]), fc.pkg) {
		fc.report(d.Name, "constructor-stutter", "%s.%s repeats the package name; use %s.New (single primary type) or %s.New<Type>", fc.pkg, name, fc.pkg, fc.pkg)
	}

	// A function that can fail returns error, not a concrete error type: a nil
	// *T stored in an error interface is non-nil.
	if res := d.Type.Results; res != nil && fieldCount(res) >= 2 {
		last := res.List[len(res.List)-1].Type
		if star, ok := last.(*ast.StarExpr); ok && strings.HasSuffix(typeName(star.X), "Error") {
			fc.report(last, "error-concrete-return", "return the error interface, not *%s; a nil concrete pointer becomes a non-nil error", typeName(star.X))
		}
	}

	if d.Body == nil || fc.isMain || fc.isTest || name == "init" || strings.HasPrefix(name, "Must") || strings.HasPrefix(name, "must") {
		return
	}
	ast.Inspect(d.Body, func(n ast.Node) bool {
		if _, ok := n.(*ast.FuncLit); ok {
			return false
		}
		if call, ok := n.(*ast.CallExpr); ok {
			if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "panic" {
				fc.report(call, "library-panic", "library code returns errors; panic only for API misuse or broken invariants, and never across the package boundary")
			}
		}
		return true
	})
}

// checkLogAndReturn reports an if block that both logs err and returns it,
// which reports the same failure twice.
func (fc *fileCtx) checkLogAndReturn(n *ast.IfStmt) {
	var logged, returned bool
	for _, st := range n.Body.List {
		switch st := st.(type) {
		case *ast.ExprStmt:
			if call, ok := st.X.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && logMethods[sel.Sel.Name] && strings.Contains(strings.ToLower(exprString(sel.X)), "log") && mentionsErr(call) {
					logged = true
				}
			}
		case *ast.ReturnStmt:
			if mentionsErr(st) {
				returned = true
			}
		}
	}
	if logged && returned {
		fc.report(n, "error-log-and-return", "error is both logged and returned; return it with context and let the caller decide whether to log")
	}
}

// checkTestMessage reports test failures that print want before got.
// revive's string-format rule already bans expected/actual wording.
func (fc *fileCtx) checkTestMessage(call *ast.CallExpr) {
	if !fc.isTest || len(call.Args) == 0 {
		return
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || (sel.Sel.Name != "Errorf" && sel.Sel.Name != "Fatalf") {
		return
	}
	bl, ok := call.Args[0].(*ast.BasicLit)
	if !ok || bl.Kind != token.STRING {
		return
	}
	s, err := strconv.Unquote(bl.Value)
	if err != nil {
		return
	}
	lower := strings.ToLower(s)
	w, g := strings.Index(lower, "want"), strings.Index(lower, "got")
	if w >= 0 && g > w && !strings.Contains(lower, "-want") && !strings.Contains(lower, "want)") {
		fc.report(bl, "test-message", "print got before want: \"Func(%%v) = %%v, want %%v\"")
	}
}

// parseIgnore returns the suppressed rule names when text is a line comment of
// the form "// sdkcheck:ignore <rule>[,<rule>...] <reason>". Directives without
// known rule names or a reason, and prose comments that merely mention
// sdkcheck:ignore, return nil.
func parseIgnore(text string) []string {
	if !strings.HasPrefix(text, "//") {
		return nil
	}
	body := strings.TrimSpace(strings.TrimPrefix(text, "//"))
	if !strings.HasPrefix(body, "sdkcheck:ignore ") && !strings.HasPrefix(body, "sdkcheck:ignore\t") {
		return nil
	}
	fields := strings.Fields(strings.TrimPrefix(body, "sdkcheck:ignore"))
	if len(fields) < 2 {
		return nil
	}
	var rules []string
	for _, r := range strings.Split(fields[0], ",") {
		if r == "" || ruleDocs[r] == "" {
			return nil
		}
		rules = append(rules, r)
	}
	return rules
}

func mentionsErr(n ast.Node) bool {
	found := false
	ast.Inspect(n, func(n ast.Node) bool {
		if id, ok := n.(*ast.Ident); ok && id.Name == "err" {
			found = true
		}
		return !found
	})
	return found
}

func typeName(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return e.Sel.Name
	case *ast.IndexExpr:
		return typeName(e.X)
	case *ast.IndexListExpr:
		return typeName(e.X)
	case *ast.StarExpr:
		return typeName(e.X)
	}
	return ""
}

func fieldCount(fl *ast.FieldList) int {
	n := 0
	for _, f := range fl.List {
		n += max(len(f.Names), 1)
	}
	return n
}

func exprString(e ast.Expr) string {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.SelectorExpr:
		return exprString(e.X) + "." + e.Sel.Name
	case *ast.CallExpr:
		return exprString(e.Fun) + "()"
	}
	return ""
}
