// Command bcov measures branch coverage for Go packages, counting every outcome of every decision.
//
// A decision is an if or for condition, or a switch or type switch:
//   - if and for have two outcomes: the condition is false, or it is true.
//   - switch and type switch have one outcome per clause. A switch with no default gets a synthetic
//     default clause, so "no clause matched" is an outcome too.
//
// The instrumented copies are written under -out and applied with `go test -overlay`, so the
// repository tree is not modified. Each package gets a helper file that counts outcomes and a test
// hook that writes the counts to $BCOV_OUT when the tests finish. The evidence runner reads both
// the decision manifest and the counts.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const (
	helperName   = "zz_bcov_helper.go"
	testHookName = "zz_bcov_main_test.go"
)

type decision struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Kind     string `json:"kind"`
	Base     int    `json:"base"`
	Branches int    `json:"branches"`
}

type packageResult struct {
	Dir       string     `json:"dir"`
	Hits      string     `json:"hits"`
	Size      int        `json:"size"`
	Decisions []decision `json:"decisions"`
}

type manifest struct {
	Packages []packageResult `json:"packages"`
}

type overlayFile struct {
	Replace map[string]string `json:"Replace"`
}

const helperTemplate = `package %s

import "sync/atomic"

// bcovHits holds one counter per decision outcome. Indexes come from the decision manifest.
var bcovHits = make([]uint64, %d)

func bcovCond(base int, v bool) bool {
	if v {
		atomic.AddUint64(&bcovHits[base+1], 1)
	} else {
		atomic.AddUint64(&bcovHits[base], 1)
	}
	return v
}

func bcovCase(idx int) {
	atomic.AddUint64(&bcovHits[idx], 1)
}
`

const testHookTemplate = `package %s

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
)

func TestMain(m *testing.M) {
	code := m.Run()
	if dir := os.Getenv("BCOV_OUT"); dir != "" {
		if f, err := os.Create(filepath.Join(dir, %q)); err == nil {
			for i := range bcovHits {
				fmt.Fprintf(f, "%%d %%d\n", i, atomic.LoadUint64(&bcovHits[i]))
			}
			f.Close()
		}
	}
	os.Exit(code)
}
`

func main() {
	root := flag.String("root", ".", "repository root")
	out := flag.String("out", "reports/bcov", "output directory for instrumented files and the manifest")
	scope := flag.String("scope", "cmd,internal", "comma-separated directories under root to measure")
	flag.Parse()

	absRoot, err := filepath.Abs(*root)
	check(err)
	absOut, err := filepath.Abs(*out)
	check(err)
	check(os.RemoveAll(absOut))

	dirs := map[string][]string{} // package dir (repo-relative) -> files (repo-relative)
	for _, s := range strings.Split(*scope, ",") {
		base := filepath.Join(absRoot, strings.TrimSpace(s))
		check(filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() && d.Name() == "testdata" {
				return filepath.SkipDir
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			rel, err := filepath.Rel(absRoot, path)
			if err != nil {
				return err
			}
			pkgDir := filepath.ToSlash(filepath.Dir(rel))
			dirs[pkgDir] = append(dirs[pkgDir], filepath.ToSlash(rel))
			return nil
		}))
	}

	overlay := overlayFile{Replace: map[string]string{}}
	var result manifest
	pkgDirs := make([]string, 0, len(dirs))
	for d := range dirs {
		pkgDirs = append(pkgDirs, d)
	}
	sort.Strings(pkgDirs)

	for _, pkgDir := range pkgDirs {
		files := dirs[pkgDir]
		sort.Strings(files)
		counter := 0
		var decisions []decision
		pkgName := ""
		for _, rel := range files {
			fset := token.NewFileSet()
			src := filepath.Join(absRoot, rel)
			file, err := parser.ParseFile(fset, src, nil, parser.ParseComments)
			check(err)
			if pkgName == "" {
				pkgName = file.Name.Name
			}
			found, err := instrument(fset, file, rel, &counter)
			check(err)
			decisions = append(decisions, found...)

			instrumented := filepath.Join(absOut, "src", rel)
			check(os.MkdirAll(filepath.Dir(instrumented), 0o755))
			var buf strings.Builder
			check(printer.Fprint(&buf, fset, file))
			check(os.WriteFile(instrumented, []byte(buf.String()), 0o644))
			overlay.Replace[src] = instrumented
		}

		pkgAbs := filepath.Join(absRoot, pkgDir)
		helper := filepath.Join(absOut, "src", pkgDir, helperName)
		hook := filepath.Join(absOut, "src", pkgDir, testHookName)
		check(os.MkdirAll(filepath.Dir(helper), 0o755))
		check(os.WriteFile(helper, []byte(fmt.Sprintf(helperTemplate, pkgName, counter)), 0o644))
		tag := strings.ReplaceAll(pkgDir, "/", "_") + ".txt"
		check(os.WriteFile(hook, []byte(fmt.Sprintf(testHookTemplate, pkgName, tag)), 0o644))
		overlay.Replace[filepath.Join(pkgAbs, helperName)] = helper
		overlay.Replace[filepath.Join(pkgAbs, testHookName)] = hook

		result.Packages = append(result.Packages, packageResult{
			Dir: pkgDir, Hits: tag, Size: counter, Decisions: decisions,
		})
	}

	writeJSON(filepath.Join(absOut, "overlay.json"), overlay)
	writeJSON(filepath.Join(absOut, "decisions.json"), result)
	total := 0
	for _, p := range result.Packages {
		total += len(p.Decisions)
	}
	fmt.Printf("instrumented %d packages, %d decisions, %d outcomes\n", len(result.Packages), total, sumSizes(result))
}

// instrument rewrites the decisions in one file and returns their manifest entries.
func instrument(fset *token.FileSet, file *ast.File, rel string, counter *int) ([]decision, error) {
	var decisions []decision
	var unsupported error
	pos := func(p token.Pos) int { return fset.Position(p).Line }

	ast.Inspect(file, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.IfStmt:
			base := *counter
			*counter += 2
			decisions = append(decisions, decision{File: rel, Line: pos(s.If), Kind: "if", Base: base, Branches: 2})
			s.Cond = call("bcovCond", lit(base), s.Cond)
		case *ast.ForStmt:
			if s.Cond == nil {
				return true
			}
			base := *counter
			*counter += 2
			decisions = append(decisions, decision{File: rel, Line: pos(s.For), Kind: "for", Base: base, Branches: 2})
			s.Cond = call("bcovCond", lit(base), s.Cond)
		case *ast.SwitchStmt:
			instrumentSwitch(s.Body, s.Switch, fset, rel, counter, &decisions)
		case *ast.TypeSwitchStmt:
			instrumentSwitch(s.Body, s.Switch, fset, rel, counter, &decisions)
		case *ast.SelectStmt:
			unsupported = fmt.Errorf("%s:%d: select statements are not supported by bcov", rel, pos(s.Select))
		}
		return true
	})
	return decisions, unsupported
}

func instrumentSwitch(body *ast.BlockStmt, at token.Pos, fset *token.FileSet, rel string,
	counter *int, decisions *[]decision) {
	hasDefault := false
	for _, stmt := range body.List {
		if cc, ok := stmt.(*ast.CaseClause); ok && cc.List == nil {
			hasDefault = true
		}
	}
	if !hasDefault {
		// A synthetic default clause makes "no clause matched" a counted outcome. It is empty, so it changes nothing.
		body.List = append(body.List, &ast.CaseClause{})
	}
	// Outcomes are numbered in clause order.
	base := *counter
	*counter += len(body.List)
	*decisions = append(*decisions, decision{
		File: rel, Line: fset.Position(at).Line, Kind: "switch", Base: base, Branches: len(body.List),
	})
	for i, stmt := range body.List {
		cc := stmt.(*ast.CaseClause)
		cc.Body = append([]ast.Stmt{exprStmt(call("bcovCase", lit(base+i)))}, cc.Body...)
	}
}

func call(name string, args ...ast.Expr) *ast.CallExpr {
	return &ast.CallExpr{Fun: ast.NewIdent(name), Args: args}
}

func exprStmt(e ast.Expr) ast.Stmt { return &ast.ExprStmt{X: e} }

func lit(v int) ast.Expr {
	return &ast.BasicLit{Kind: token.INT, Value: strconv.Itoa(v)}
}

func sumSizes(m manifest) int {
	total := 0
	for _, p := range m.Packages {
		total += p.Size
	}
	return total
}

func writeJSON(path string, v any) {
	check(os.MkdirAll(filepath.Dir(path), 0o755))
	data, err := json.MarshalIndent(v, "", "  ")
	check(err)
	check(os.WriteFile(path, data, 0o644))
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "bcov:", err)
		os.Exit(1)
	}
}
