//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package parser

import (
	"bytes"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
        "fmt"
        "github.com/duzy/smart/token"
        "github.com/duzy/smart/ast"
)

type RuntimeSym interface{
}

type RuntimeContext interface {
        IsDialect(s string) bool
        IsModifier(s string) bool

        DeclareProject(name string) error

        OpenScope(as *ast.Scope, pos token.Pos, comment string) error
        CloseScope(as *ast.Scope) error

        Import(spec *ast.ImportSpec) error
        Include(spec *ast.IncludeSpec) error
        Use(spec *ast.UseSpec) error
        Eval(spec *ast.EvalSpec) error
        
        Define(clause *ast.DefineClause) (RuntimeSym, error)
        DeclareRule(clause *ast.RuleClause) (RuntimeSym, error)
        
        EvalExpr(x ast.Expr) (fmt.Stringer, error)
}

type Context struct {
        runtime  RuntimeContext
	universe *ast.Scope // builtin scope
}

func NewContext(runtime RuntimeContext) *Context {
        return &Context{
                runtime:  runtime,
                universe: ast.NewScope(nil),
        }
}

func (c *Context) Builtin(s string, i interface{}) {
        //fmt.Printf("builtin: %v\n", s)
        sym := ast.NewSym(ast.Bui, s)
        if alt := c.universe.Insert(sym); alt != nil {
                panic(fmt.Sprintf("duplicated builtin '%s'", s))
                // FIXME: unreachable
        }
}

// If src != nil, readSource converts src to a []byte if possible;
// otherwise it returns an error. If src == nil, readSource returns
// the result of reading the file specified by filename.
//
func readSource(filename string, src interface{}) ([]byte, error) {
	if src != nil {
		switch s := src.(type) {
		case string:
			return []byte(s), nil
		case []byte:
			return s, nil
		case *bytes.Buffer:
			// is io.Reader, but src is already available in []byte form
			if s != nil {
				return s.Bytes(), nil
			}
		case io.Reader:
			var buf bytes.Buffer
			if _, err := io.Copy(&buf, s); err != nil {
				return nil, err
			}
			return buf.Bytes(), nil
		}
		return nil, errors.New("invalid source")
	}
	return ioutil.ReadFile(filename)
}

// A Mode value is a set of flags (or 0).
// They control the amount of source code parsed and other optional
// parser functionality.
//
type Mode uint

const (
	ModuleClauseOnly Mode = 1 << iota // stop parsing after project or module clause
	ImportsOnly                       // stop parsing after import declarations
	ParseComments                     // parse comments and add them to AST
        Flat                              // parsing in flat mode (donot create a new module)
	Trace                             // print a trace of parsed productions
	DeclarationErrors                 // report declaration errors
	SpuriousErrors                    // same as AllErrors, for backward-compatibility
	AllErrors = SpuriousErrors        // report all errors (not just the first 10 on different lines)
        parsingDir
)

// ParseFile parses the source code of a single Go source file and returns
// the corresponding ast.File node. The source code may be provided via
// the filename of the source file, or via the src parameter.
//
// If src != nil, ParseFile parses the source from src and the filename is
// only used when recording position information. The type of the argument
// for the src parameter must be string, []byte, or io.Reader.
// If src == nil, ParseFile parses the file specified by filename.
//
// The mode parameter controls the amount of source text parsed and other
// optional parser functionality. Position information is recorded in the
// file set fset.
//
// If the source couldn't be read, the returned AST is nil and the error
// indicates the specific failure. If the source was read but syntax
// errors were found, the result is a partial AST (with ast.Bad* nodes
// representing the fragments of erroneous source code). Multiple errors
// are returned via a scanner.ErrorList which is sorted by file position.
//
func (c *Context) ParseFile(fset *token.FileSet, filename string, src interface{}, mode Mode) (f *ast.File, err error) {
	// get source
	text, err := readSource(filename, src)
	if err != nil {
		return nil, err
	}

	var p parser
	defer func() {
		if e := recover(); e != nil {
			// resume same panic if it's not a bailout
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}

		// set result values
		if f == nil {
			// source is not a valid source file - satisfy
			// ParseFile API and return a valid (but) empty
			// *ast.File
			f = &ast.File{
				Name:  new(ast.Ident),
				Scope: ast.NewScope(nil),
			}
		}

		p.errors.Sort()
		err = p.errors.Err()
	}()

	// parse source
	p.init(c, fset, filename, text, mode)
	f = p.parseFile()
	return
}

// ParseDir calls ParseFile for all files with names ending in ".go" in the
// directory specified by path and returns a map of package name -> package
// AST with all the packages found.
//
// If filter != nil, only the files with os.FileInfo entries passing through
// the filter (and ending in ".go") are considered. The mode bits are passed
// to ParseFile unchanged. Position information is recorded in fset.
//
// If the directory couldn't be read, a nil map and the respective error are
// returned. If a parse error occurred, a non-nil but incomplete map and the
// first error encountered are returned.
//
func (c *Context) ParseDir(fset *token.FileSet, path string, filter func(os.FileInfo) bool, mode Mode) (mods map[string]*ast.Project, first error) {
	fd, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer fd.Close()

	list, err := fd.Readdir(-1)
	if err != nil {
		return nil, err
	}

	mods = make(map[string]*ast.Project)
	for _, d := range list {
                sm := strings.HasSuffix(d.Name(), ".smart") || strings.HasSuffix(d.Name(), ".sm")
		if sm && (filter == nil || filter(d)) {
			filename := filepath.Join(path, d.Name())
			if src, err := c.ParseFile(fset, filename, nil, mode|parsingDir); err == nil {
				name := src.Name.Name
				mod, found := mods[name]
				if !found {
					mod = &ast.Project{
                                                Name:    name,
                                                Scope:   ast.NewScope(nil),
                                                Files:   make(map[string]*ast.File),
                                        }
					mods[name] = mod
				}
                                mod.Files[filename] = src
			} else if first == nil {
				first = err
			}
		}
	}

	return
}

// ParseExprFrom is a convenience function for parsing an expression.
// The arguments have the same meaning as for Parse, but the source must
// be a valid Go (type or value) expression.
//
/*
func ParseExprFrom(fset *token.FileSet, filename string, src interface{}, mode Mode) (ast.Expr, error) {
	// get source
	text, err := readSource(filename, src)
	if err != nil {
		return nil, err
	}

	var p parser
	defer func() {
		if e := recover(); e != nil {
			// resume same panic if it's not a bailout
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}
		p.errors.Sort()
		err = p.errors.Err()
	}()

	// parse expr
	p.init(fset, filename, text, mode)
	// Set up pkg-level scopes to avoid nil-pointer errors.
	// This is not needed for a correct expression x as the
	// parser will be ok with a nil topScope, but be cautious
	// in case of an erroneous x.
	p.openScope()
	p.pkgScope = p.topScope
	//e := p.parseRhsOrType()
	p.closeScope()

	assert(p.topScope == nil, "unbalanced scopes")

	// If a semicolon was inserted, consume it;
	// report an error if there's more tokens.
	if p.tok == token.LINEND && p.lit == "\n" {
		p.next()
	}
	p.expect(token.EOF)

	if p.errors.Len() > 0 {
		p.errors.Sort()
		return nil, p.errors.Err()
	}

	return e, nil
}

// ParseExpr is a convenience function for obtaining the AST of an expression x.
// The position information recorded in the AST is undefined. The filename used
// in error messages is the empty string.
//
func ParseExpr(x string) (ast.Expr, error) {
	return ParseExprFrom(token.NewFileSet(), "", []byte(x), 0)
}
*/
