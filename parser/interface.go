//
//  Copyright (C) 2012-2017, Duzy Chan <code@duzy.info>, all rights reserverd.
//  Use of this source code is governed by a BSD-style license that can be
//  found in the LICENSE file.
//
package parser

import (
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
        "unicode/utf8"
        "fmt"
        "extbit.io/smart/ast"
        "extbit.io/smart/token"
        "extbit.io/smart/types"
        "extbit.io/smart/values"
)

const optSortErrors = false

type ResolveBits int
const (
        // If many bits are set, resolve in the listed priority.
        FromGlobe ResolveBits = 1<<iota
        FromBase
        FromProject
        FindDef
        FindRule

        FromHere

        // This is the default be
        anywhere = FromHere
        global = FromGlobe
        local = FromProject
        nonlocal = FromGlobe | FromBase | FromProject
)

type EvalBits int
const (
        KeepClosures EvalBits = 1<<iota
        KeepDelegates

        // Wants value for rule depends.
        DependValue

        // Wants v.Strval(), expends delegates and closures,
        // turn off KeepClosures, KeepDelegates.
        StringValue = 0
)

type RuntimeObj ast.Symbol

type RuntimeContext interface {
        Getwd() string
        
        File(name string) *types.File
        MapFile(pat string, paths []string)

        DeclareProject(name *ast.Bareword, params types.Value) error
        CloseCurrentProject(name *ast.Bareword) error

        OpenNamedScope(name, comment string) (ast.Scope, error)
        OpenScope(comment string) ast.Scope
        CloseScope(scope ast.Scope) error

        ClauseImport(spec *ast.ImportSpec) (error, int) // (error, wrong arg num)
        ClauseInclude(spec *ast.IncludeSpec) error
        ClauseUse(spec *ast.UseSpec) error
        ClauseEval(spec *ast.EvalSpec) error
        ClauseDock(spec *ast.DockSpec) error
        
        Rule(clause *ast.RuleClause) (RuntimeObj, error)
        
        Resolve(name string, bits ResolveBits) (obj RuntimeObj)
        Symbol(name string, t types.Type) (obj, alt RuntimeObj)

        Eval(x ast.Expr, bits EvalBits) (types.Value, error)
}

type Context struct {
        runtime  RuntimeContext
	universe ast.Scope // builtin scope
        p        *parser   // current parser (or nil)
}

func NewContext(runtime RuntimeContext, universe ast.Scope) *Context {
        return &Context{
                runtime:  runtime,
                universe: universe,
        }
}

func (c *Context) Position(pos token.Pos) (position token.Position) {
        if c.p != nil {
                position = c.p.file.Position(pos)
        }
        return
}

func (c *Context) ParseWarn(pos token.Pos, s string, a... interface{}) {
        if c.p != nil {
                c.p.warn(pos, s, a...)
        }
}

func (c *Context) ParseInfo(pos token.Pos, s string, a... interface{}) {
        if c.p != nil {
                c.p.info(pos, s, a...)
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
		return nil, fmt.Errorf("invalid source")
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

// ParseFile parses the source code of a single source file and returns
// the corresponding ast.File node. The source code may be provided via
// the filename of the source file, or via the src parameter.
func (c *Context) ParseFile(fset *token.FileSet, filename string, src interface{}, mode Mode) (f *ast.File, err error) {
	// get source
	text, err := readSource(filename, src)
	if err != nil {
		return nil, err
	}

        //fmt.Printf("ParseFile: %v\n", filename)
        
	var (
                oldp = c.p
                p parser
        )
	defer func() {
		if e := recover(); e != nil {
			// resume same panic if it's not a bailout
			if _, ok := e.(bailout); !ok {
				panic(e)
			}
		}

		// set result values
		if f == nil {
                        s := fmt.Sprintf("file %s", filename)

			// source is not a valid source file - satisfy
			// ParseFile API and return a valid (but) empty
			// *ast.File
			f = &ast.File{
				Name:  new(ast.Bareword),
				Scope: c.runtime.OpenScope(s),
			}
                        c.runtime.CloseScope(f.Scope)
		}

                if optSortErrors {
                        p.errors.Sort()
                }
		err = p.errors.Err()
                c.p = oldp
	}()

        // set the current parser
        c.p = &p

	// parse source
	p.init(c, fset, filename, text, mode)
	f = p.parseFile()
	return
}

// ParseConfigDir parses a configuration directory, where
//     * pathname - is the original pathname (symlink or 'configure' smart file)
//     * linked - is the destination directory pathname to be really iterated
func (c *Context) ParseConfigDir(pathname, linked string) (err error) {
        var fd *os.File // Directory of the destination.
	if fd, err = os.Open(linked); err != nil { return }
	defer fd.Close()

        var list []os.FileInfo
	if list, err = fd.Readdir(-1); err != nil || len(list) == 0 {
                return 
        }

        var (
                sym RuntimeObj
                wd = c.runtime.Getwd()
                rel , _ = filepath.Rel(wd, pathname)
                ident = filepath.Base(pathname)
        )
        if ident == "_" {
                return fmt.Errorf("invalid package name %s", ident)
        }

        scope, err := c.runtime.OpenNamedScope(ident, fmt.Sprintf("config %s", pathname))
        if err != nil {
                return
        }
        defer func() { err = c.runtime.CloseScope(scope) } ()

        sym, _ = c.runtime.Symbol("/", types.DefType)
        sym.(*types.Def).Assign(values.String(pathname))

        sym, _ = c.runtime.Symbol(".", types.DefType)
        sym.(*types.Def).Assign(values.String(rel))

	ListLoop: for _, d := range list {
                var name = d.Name()
                if strings.HasPrefix(name, ".#") || 
                   strings.HasSuffix(name, "~") || 
                   strings.HasSuffix(name, ".smart") ||
                   strings.HasSuffix(name, ".sm") {
                        continue ListLoop
                }

                var fullname = filepath.Join(linked, name)
                if d.Mode()&os.ModeSymlink != 0 {
                        var ( l string; t os.FileInfo )
                        if l, err = os.Readlink(fullname); err != nil { continue ListLoop }
                        if !filepath.IsAbs(l) { l = filepath.Join(linked, l) }
                        if t, err = os.Stat(l); err != nil { continue ListLoop }
                        if t.IsDir() { continue ListLoop }
                }

                if d.IsDir() {
                        if err = c.ParseConfigDir(filepath.Join(pathname, name), fullname); err != nil {
                                break ListLoop
                        }
                } else if s, a := c.runtime.Symbol(name, types.DefType); a != nil {
                        err = fmt.Errorf("declare project: %v", err)
                        break ListLoop
                } else if def, _ := s.(*types.Def); def != nil {
                        var ( v []byte; s string )
                        if v, err = ioutil.ReadFile(fullname); err != nil { break ListLoop }
                        if s = string(v); !utf8.ValidString(s) {
                                err = fmt.Errorf("%s: invalid UTF8 content", fullname)
                                break ListLoop
                        }
                        def.SetOrigin(types.ImmediateDef)
                        def.Assign(values.String(s))
                        //fmt.Printf("%s: %v = %v\n", ident, name, s)
                } else if s != nil {
                        err =  fmt.Errorf("Name `%s' already taken, not def (%T).", name, s)
                        break ListLoop
                }
        }
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

        //fmt.Printf("ParseDir: %v\n", path)
        //fmt.Printf("ParseDir: %v %v\n", path, list)
        //fmt.Printf("ParseDir: runtime: %p\n", c.runtime)
        //fmt.Printf("ParseDir: parser: %p\n", c.p)
        for i, a := range list {
                if i > 0 && a.Name() == "build.smart" {
                        first := list[0]
                        list[0] = a
                        list[i] = first
                        //fmt.Printf("ParseDir: %v <-> %v\n", a.Name(), first.Name())
                }
        }
        //fmt.Printf("ParseDir: %v %v %v\n", path, len(list), list[0].Name())

        scope := c.runtime.OpenScope(fmt.Sprintf("dir %s", path))
        defer func() {
                if err := c.runtime.CloseScope(scope); err != nil {
                        if first == nil {
                                first = err
                        }
                }
                //fmt.Printf("ParseDir: %v\n%v\n", scope, scope.(*types.Scope).Outer())
        }()

	mods = make(map[string]*ast.Project)
	ListLoop: for _, d := range list {
                var (
                        filename, mo = filepath.Join(path, d.Name()), d.Mode()
                        linked, linkPath = "", path
                )
                for fn := filename; mo&os.ModeSymlink != 0; {
                        if s, err := os.Readlink(fn); err != nil {
                                continue ListLoop
                        } else {
                                rel := !filepath.IsAbs(s)
                                if rel { s = filepath.Join(linkPath, s) }
                                if fi, err := os.Lstat(s); err != nil {
                                        continue ListLoop
                                } else {
                                        if rel { linkPath = filepath.Dir(s) }
                                        mo, fn = fi.Mode(), s
                                        linked = fn
                                }
                        }
                }

                if strings.HasPrefix(d.Name(), ".#") ||
                        (!strings.HasSuffix(d.Name(), ".smart") &&
                        !strings.HasSuffix(d.Name(), ".sm")) {
                        continue
                } else if s := d.Name(); (s == "configure.smart" || s == "configure.sm") && (len(linked) > 0 || mo.IsDir()) {
                        if err := c.ParseConfigDir(filepath.Dir(filename), linked); err != nil {
                                if first == nil {
                                        first = err
                                }
                                return
                        }
                        continue ListLoop
                } else if s == "config.smart" || s == "config.sm" {
                        err = fmt.Errorf("use configure.sm[art] instead of config.sm[art]")
                        break
                }

		if mo.IsRegular() && (filter == nil || filter(d)) {
			if src, err := c.ParseFile(fset, filename, nil, mode|parsingDir); err == nil {
                                if src.Name == nil {
                                        first = fmt.Errorf("module '%v' has no name", filename)
                                        return
                                }

				name := src.Name.Value
				mod, found := mods[name]
				if !found {
					mod = &ast.Project{
                                                Name:    name,
                                                Scope:   scope,
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
