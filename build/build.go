//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "bytes"
        "errors"
        "fmt"
        "io"
        //"io/ioutil"
        "os"
        "os/exec"
        "path/filepath"
        "regexp"
        "runtime"
        "strings"
        "sort"
        "github.com/duzy/worker"
)

var (
        workdir, _ = os.Getwd()

        initScript []string
        hooksMap = HooksMap{
        }

        generalMetaFiles = []*FileMatcher{
                { "backup",     os.ModeDir | ^os.ModeType,      `[^~]*~$` },
                { "git",        os.ModeDir,                     `^\.git$` },
                { "git",       ^os.ModeType,                    `^\.gitignore$` },
                { "mercurial",  os.ModeDir,                     `^\.hg$` },
                { "subversion", os.ModeDir,                     `^\.svn$` },
                { "cvs",       ^os.ModeType,                    `^CVS$` },
        }

        onRecipeExecutionFailure = func(err error, c *exec.Cmd) {
                /*
                if s, e := ioutil.ReadAll(c.Stderr); e != nil && 0 < len(s) {
                        fmt.Printf(os.Stderr, "%s", string(s))
                } else {
                        fmt.Printf(os.Stderr, "fail to execute `%v'", c.Path)
                } */
                fmt.Fprintf(os.Stderr, "error: `%v` %v\n", c.Path, c.Args)
                fmt.Fprintf(os.Stderr, "error: %v\n", err)
                os.Exit(-1)
        }

        recipeStdout, recipeStderr, recipeEcho = io.Writer(os.Stdout), io.Writer(os.Stderr), io.Writer(os.Stdout)
)

// TODO: put this into namespace
// toolset represents a toolchain like gcc and related utilities.
type toolset interface {
        // DeclModule setup the current module being processed.
        // `args' and `vars' is passed in on the `$(module)' invocation.
        DeclModule(p *Context, args Items, vars map[string]string)

        // CommitModule setup the current module being committed.
        CommitModule(p *Context, args Items)

        // UseModule lets a toolset decides how to use a module.
        UseModule(p *Context, o *Module) bool

        // getNamespace returns toolset namespace (internal)
        getNamespace() namespace
}

type HooksMap map[string]HookTable
type HookTable map[string]Hook
type Hook func(ctx *Context, args Items) Items
type HookContext struct {
}

func (hc *HookContext) Unhook() {
        panic("todo: Unhook...")
}

func Hookup(hm HooksMap, script string) (hc *HookContext, err error) {
        if hm != nil {
                for k, _ := range hm {
                        if v, ok := hooksMap[k]; ok && v != nil { 
                                s := fmt.Sprintf("%v already taken", k)
                                err = errors.New(s)
                                return
                        }
                }
                for k, h := range hm {
                        hooksMap[k] = h
                }
        }
        if script != "" {
                initScript = append(initScript, script)
        }
        if err == nil {
                hc = new(HookContext)
        }
        return
}

func MustHookup(ht HooksMap, script string) *HookContext {
        hc, err := Hookup(ht, script)
        if err != nil {
                panic(err)
        }
        return hc
}

type SourceLangMap map[string][]string

func (m SourceLangMap) ExtLang(ext string) (lang string) {
        ext = strings.ToLower(ext)
source_lang_loop:
        for s, l := range m {
                for _, x := range l {
                        if strings.ToLower(x) == ext {
                                lang = s; break source_lang_loop
                        }
                }
        }
        return
}

func IsIA32Command(s string) bool {
        buf := new(bytes.Buffer)
        cmd := exec.Command("file", "-b", s)
        cmd.Stdout = buf
        if err := cmd.Run(); err != nil {
                message("error: %v", err)
        }
        //return strings.HasPrefix(buf.String(), "ELF 32-bit")
        return strings.Contains(buf.String(), "ELF 32-bit")
}

// A command executed by an action while updating a target.
type Command interface {
        Execute(targets []string, prerequisites []string) bool
}

type Excmd struct {
        path string
        dir string
        mkdir string
        precall func() bool
        postcall func(so, se *bytes.Buffer)
        cmd func() bool
        ia32 bool
        stdout, stderr *bytes.Buffer
        stdin io.Reader
}

func NewExcmd(s string) *Excmd {
        return &Excmd{ path:s }
}

func (c *Excmd) GetPath() string { return c.path }
func (c *Excmd) SetPath(s string) { c.path = s }
func (c *Excmd) GetIA32() bool { return c.ia32 }
func (c *Excmd) SetIA32(v bool) { c.ia32 = v }
func (c *Excmd) SetMkdir(s string) { c.mkdir = s }
func (c *Excmd) GetMkdir() string { return c.mkdir }
func (c *Excmd) SetDir(s string) { c.dir = s }
func (c *Excmd) GetDir() string { return c.dir }
func (c *Excmd) SetStderr(s *bytes.Buffer/*io.Writer*/) { c.stderr = s }
func (c *Excmd) GetStderr() *bytes.Buffer/*io.Writer*/ { return c.stderr }
func (c *Excmd) SetStdout(s *bytes.Buffer/*io.Writer*/) { c.stdout = s }
func (c *Excmd) GetStdout() *bytes.Buffer/*io.Writer*/ { return c.stdout }
func (c *Excmd) SetStdin(s io.Reader) { c.stdin = s }
func (c *Excmd) GetStdin() io.Reader { return c.stdin }
func (c *Excmd) Run(targetHint string, args ...string) bool {
        return c.run(targetHint, args...)
}

func (c *Excmd) run(targetHint string, args ...string) bool {
        if strings.HasPrefix(c.path, "~/") {
                c.path = os.Getenv("HOME") + c.path[1:]
        }
        if c.mkdir != "" {
                if e := os.MkdirAll(c.mkdir, 0755); e != nil {
                        return false
                }
        }

        if c.stdout == nil { c.stdout = new(bytes.Buffer) }
        if c.stderr == nil { c.stderr = new(bytes.Buffer) }
        c.stdout.Reset()
        c.stderr.Reset()

        updated := false
        if c.path == "" {
                if c.cmd != nil {
                        updated = c.cmd()
                } else {
                        errorf("can't update `%v'", targetHint)
                        return false
                }
        } else {
                cmd := exec.Command(c.path, args...)
                cmd.Stdout, cmd.Stderr = c.stdout, c.stderr
                if c.stdin != nil { cmd.Stdin = c.stdin }
                if c.dir != "" { cmd.Dir = c.dir }

                if *flagV {
                        if targetHint != "" {
                                message("%v -> %v", filepath.Base(c.path), targetHint)
                        }
                } else if *flagVV {
                        fmt.Printf("%v\n", strings.Join(cmd.Args, " "))
                }

                if c.precall != nil && c.precall() == false {
                        return false
                }

                if c.ia32 && runtime.GOOS == "linux" {
                        switch runtime.GOARCH {
                        case "amd64":
                                cmd = exec.Command("linux32", append([]string{ cmd.Path }, args...)...)
                                cmd.Stdout, cmd.Stderr = c.stdout, c.stderr
                                if c.stdin != nil { cmd.Stdin = c.stdin }
                                //fmt.Printf("%v\n", strings.Join(cmd.Args, " "))
                        }
                }

                err := cmd.Run()
                if err == nil {
                        updated = true
                }

                if (*flagL /**flagV && *flagVV*/) || err != nil {
                        if err != nil { message("%v (%v)", err, c.path) }
                        if c.path != "" {
                                fmt.Fprintf(os.Stderr, "--------------------------------------------------------------------------------\n")
                                fmt.Fprintf(os.Stderr, "%v %v\n", c.path, strings.Join(args, " "))
                        }
                        so, se := c.stdout.String(), c.stderr.String()
                        if so != "" {
                                fmt.Fprintf(os.Stderr, "------------------------------------------------------------------------- stdout\n")
                                fmt.Fprintf(os.Stderr, "%v", so)
                                if !strings.HasSuffix(so, "\n") { fmt.Fprintf(os.Stderr, "\n") }
                        }
                        if se != "" {
                                fmt.Fprintf(os.Stderr, "------------------------------------------------------------------------- stderr\n")
                                fmt.Fprintf(os.Stderr, "%v", se)
                                if !strings.HasSuffix(se, "\n") { fmt.Fprintf(os.Stderr, "\n") }
                        }
                        fmt.Fprintf(os.Stderr, "--------------------------------------------------------------------------------\n")
                        if err != nil { errorf(`failed executing "%v"`, c.path) }
                }

                if c.postcall != nil {
                        c.postcall(c.stdout, c.stderr)
                }
        }

        return updated
}

type template struct {
        *namespaceEmbed
        name string
        declNodes []*node
        postNodes []*node
        post, commit *node
}

type templateToolset struct {
        *template
}

func (tt *templateToolset) getNamespace() namespace {
        return tt.template.namespaceEmbed
}

func (tt *templateToolset) UseModule(p *Context, o *Module) bool {
        return false
}

func (tt *templateToolset) DeclModule(ctx *Context, args Items, vars map[string]string) {
        for _, n := range tt.declNodes {
                if e := ctx.processNode(n); e != nil {
                        //errorf("%v", e)
                        break
                }
        }
}

func (tt *templateToolset) CommitModule(ctx *Context, args Items) {
        for _, n := range tt.postNodes {
                if e := ctx.processNode(n); e != nil {
                        //errorf("%v", e)
                        break
                }
        }
}

// Module is defined by a $(module) invocation in .smart script.
type Module struct {
        *namespaceEmbed
        Parent *Module // upper module
        Toolset toolset
        Updating bool // marked as 'true' if module is updating
        Children map[string]*Module
        declareLoc, commitLoc location // where does it defined and commit (could be nil)
        l *lex // the lex scope where does it defined (could be nil)
        //x *Context // the context of the module
}

func (m *Module) getNamespace(name string) (ns namespace) {
        if c, ok := m.Children[name]; ok && c != nil {
                ns = c
        }
        return
}

func (m *Module) GetDeclareLocation() (s string, lineno, colno int) {
        if l := m.l; l != nil {
                lineno, colno = l.caculateLocationLineColumn(m.declareLoc)
                s = l.scope
        }
        return
}

func (m *Module) GetCommitLocation() (s string, lineno, colno int) {
        if l := m.l; l != nil {
                lineno, colno = l.caculateLocationLineColumn(m.commitLoc)
                s = l.scope
        }
        return
}

func (m *Module) Get(ctx *Context, name string) (s string) {
        if d, ok := m.defines[name]; ok && d != nil {
                s = d.value.Expand(ctx)
        }
        return
}

func (m *Module) GetName(ctx *Context) string { return m.Get(ctx, "name") }
func (m *Module) GetDir(ctx *Context) string { return m.Get(ctx, "dir") }
func (m *Module) GetSources(ctx *Context) (sources []string) {
        sources = Split(m.Get(ctx, "sources"))
        for i := range sources {
                if filepath.IsAbs(sources[i]) { continue }
                sources[i] = filepath.Join(m.GetDir(ctx), sources[i])
        }
        return
}

func (m *Module) update(ctx *Context) (updated bool) {
        //fmt.Printf("Module.update: %s\n", m.goal)
        if !m.Updating {
                if g, ok := m.files[m.goal]; ok && g != nil {
                        owd, err := os.Getwd()
                        if err != nil { errorf("get working directory: %v", err) }
                        
                        wd := m.Get(ctx, "workdir")
                        if wd != owd {
                                if err = os.Chdir(wd); err != nil {
                                        errorf("change working directory: %v", err)
                                }
                        }
                        
                        om := ctx.m
                        defer func(){ 
                                if wd != owd {
                                        if err = os.Chdir(owd); err != nil {
                                                errorf("change working directory: %v", err)
                                        }
                                }
                                ctx.m = om
                        }()
                        
                        ctx.m = m // change current working module 

                        m.Updating = true
                        updated = g.updateAll(ctx)
                        m.Updating = false
                }
        }
        return
}

func (ctx *Context) update(target string) (updated bool) {
        //fmt.Printf("Context.update: %s\n", target)
        if g, ok := ctx.g.files[target]; ok && g != nil {
                updated = g.updateAll(ctx)
        }
        if m, ok := ctx.modules[target]; ok && m != nil {
                updated = m.update(ctx) || updated
        }
        return
}

func (ctx *Context) targetFoundable(target string) bool {
        if g, ok := ctx.g.files[target]; ok && g != nil {
                return true
        }
        if m, ok := ctx.modules[target]; ok && m != nil {
                return true
        }
        if a := ctx.g.findMatchRules(ctx, target); 0 < len(a) {
                return true
        }
        if _, e := os.Stat(target); e == nil {
                return true
        }
        return false
}

type FileMatcher struct {
        Name string
        Mode os.FileMode
        Rule string // *regexp.Regexp
}

func (r *FileMatcher) match(fi os.FileInfo) bool {
        re := regexp.MustCompile(r.Rule)
        if fi.Mode() & r.Mode != 0 && re.MatchString(fi.Name()) {
                return true
        }
        return false
}

func (r *FileMatcher) matchName(fn string) bool {
        re := regexp.MustCompile(r.Rule)
        if re.MatchString(fn) {
                return true
        }
        return false
}

func ReadDirNames(dirname string) ([]string, error) {
        return readDirNames(dirname)
}

// readDirNames reads the directory named by dirname and returns
// a sorted list of directory entries.
func readDirNames(dirname string) ([]string, error) {
	fd, err := os.Open(dirname)
	if err != nil {
		return nil, err
	}

	defer fd.Close()

	names, err := fd.Readdirnames(-1)
	if err != nil {
		return nil, err
	}

	sort.Strings(names)
	return names, nil
}

func FindFiles(d string, sre string) ([]string, error) {
        return findFiles(d, sre)
}

func FindFile(d string, sre string) (string) {
        return findFile(d, sre)
}

func Traverse(d string, fun traverseFunc) (error) {
        return traverse(d, fun)
}

func CopyFile(s, d string) (err error) {
        return copyFile(s, d)
}

type traverseFunc func(dname string, fi os.FileInfo) bool
func traverse(d string, fun traverseFunc) (err error) {
        names, err := readDirNames(d)
        if err != nil {
                //errorf("readDirNames: %v, %v\n", err, d)
                return
        }

        var fi os.FileInfo
        for _, name := range names {
                dname := filepath.Join(d, name)
                //fmt.Printf("traverse: %s\n", dname)

                fi, err = os.Stat(dname)
                if err != nil {
                        //errorf("stat: %v\n", dname)
                        return
                }

                if !fun(dname, fi) {
                        continue
                }

                if fi.IsDir() {
                        if err = traverse(dname, fun); err != nil {
                                //errorf("traverse: %v\n", dname)
                                return
                        }
                        continue
                }
        }
        return
}

// findNFiles finds N files underneath the directory `d' recursively. It's going to
// find all files if N is less than 1.
func findNFiles(d string, sre string, num int) (files []string, err error) {
        re := regexp.MustCompile(sre)
        err = traverse(d, func(dname string, fi os.FileInfo) bool {
                if re.MatchString(dname) {
                        files = append(files, dname)
                        if num--; num == 0 { return false }
                }
                return true
        })
        return
}

// findFiles finds all files underneath the directory `d' recursively.
func findFiles(d string, sre string) (files []string, err error) {
        return findNFiles(d, sre, -1)
}

// findFile finds one file underneath the directory `d' recursively.
func findFile(d string, sre string) (file string) {
        if fs, err := findNFiles(d, sre, 1); err == nil {
                if 0 < len(fs) { file = fs[0] }
        }
        return
}

// copyFile copies a file from `s' to `d'.
func copyFile(s, d string) (err error) {
        var f1, f2 *os.File
        if f1, err = os.Open(s); err == nil {
                defer f1.Close()
                if f2, err = os.Create(d); err == nil {
                        defer f2.Close()
                        if _, err = io.Copy(f2, f1); err != nil {
                                os.Remove(d)
                        }
                }
        }
        return
}

// MatchFileInfo finds a matched FileMatcher with FileInfo.
func MatchFileInfo(fi os.FileInfo, ms []*FileMatcher) *FileMatcher {
        return matchFileInfo(fi, ms)
}

// MatchFileInfo finds a matched FileMatcher with filename.
func MatchFileName(fn string, ms []*FileMatcher) *FileMatcher {
        if fi, err := os.Stat(fn); err == nil {
                return matchFileInfo(fi, ms)
        }
        return nil
}

func matchFileInfo(fi os.FileInfo, ms []*FileMatcher) *FileMatcher {
        for _, g := range ms {
                if g.match(fi) { return g }
        }
        return nil
}

type checkRuleUpdater struct {
        checkRule *rule
}
func (c *checkRuleUpdater) check(ctx *Context, r *rule, m *match) bool {
        if c.checkRule != r {
                errorf("diverged check rule")
        }

        ec := &ruleExecuteContext{
                target: m.target, stem: m.stem,
        }

        for _, i := range c.checkRule.prerequisites {
                ec.prerequisites = append(ec.prerequisites, i)
        }

        //fmt.Printf("checkRuleUpdater.check: %v\n", m.target)
        if e := c.checkRule.execute(ctx, ec); e != nil {
                //fmt.Printf("checkRuleUpdater.check: %v\n", e)
                return true
        }

        return false
}
func (c *checkRuleUpdater) update(ctx *Context, r *rule, m *match) bool {
        //fmt.Printf("checkRuleUpdater.update: %v\n", m.target)
        if prev, ok := r.prev[m.target]; ok {
                return prev.update(ctx, m)
        }
        return false
}

type phonyTargetUpdater struct {
}
func (c *phonyTargetUpdater) check(ctx *Context, r *rule, m *match) bool {
        return r.ns.isPhonyTarget(ctx, m.target)
}
func (c *phonyTargetUpdater) update(ctx *Context, r *rule, m *match) bool {
        //fmt.Printf("phonyTargetUpdater.update: %v\n", m.target)

        err, matchedPrerequisites, _ := r.updatePrerequisites(ctx, m)
        if err != nil {
                fmt.Fprintf(os.Stderr, "%v\n", err)
                //os.Exit(-1)
                return false
        }

        needsExecute := true
        checkRules := r.ns.getRules(nodeRuleChecker, m.target)
        if 0 < len(checkRules) {
                for _, cr := range checkRules {
                        if needsExecute = cr.c.check(ctx, cr, m); needsExecute {
                                break
                        }
                }
        }

        if needsExecute {
                //fmt.Printf("phonyTargetUpdater.update: %v\n", m.target)
                ec := r.makeExecuteContext(ctx, nil, m, matchedPrerequisites)
                return r.execute(ctx, ec) == nil
        }
        return false
}

type defaultTargetUpdater struct {
}
func (c *defaultTargetUpdater) check(ctx *Context, r *rule, m *match) bool {
        if fi, err := os.Stat(m.target); err != nil || fi == nil {
                return true
        }
        return false
}
func (c *defaultTargetUpdater) update(ctx *Context, r *rule, m *match) bool {
        //fmt.Printf("defaultTargetUpdater.update: %v\n", m.target)
        
        err, matchedPrerequisites, updatedPrerequisites := r.updatePrerequisites(ctx, m)
        if err != nil {
                fmt.Fprintf(os.Stderr, "%v\n", err)
                //os.Exit(-1)
                return false
        }

        fi, err := os.Stat(m.target)
        ec := r.makeExecuteContext(ctx, fi, m, matchedPrerequisites)
updated_loop:
        for _, mr := range updatedPrerequisites {
                for _, s := range ec.newer {
                        if s == mr.match.target { continue updated_loop }
                }
                ec.newer = append(ec.newer, mr.match.target)
        }

        // Check if we need to update the target
        if err != nil || 0 < len(updatedPrerequisites) || 0 < len(ec.newer) {
                //fmt.Printf("defaultTargetUpdater.update: execute: %v\n", m.target)
                return r.execute(ctx, ec) == nil
        }

        return false
}

type match struct {
        target string
        stem string
}

func (m *match) unstem(s string) (ss string, ok bool) {
        if pos := strings.Index(s, "%"); 0 <= pos {
                prefix, suffix := s[0:pos], s[pos+1:]
                s, ok = fmt.Sprintf("%s%s%s", prefix, m.stem, suffix), true
        }
        return s, ok
}

type matchrule struct {
        match *match
        rule *rule
}

func (mr *matchrule) String() string {
        if mr.rule == nil {
                return fmt.Sprintf("{%v,<nil>}", *mr.match)
        }
        return fmt.Sprintf("{%v,%v}", *mr.match, *mr.rule)
}

func (r *rule) match(target string) (m *match, matched bool) {
        switch r.kind {
        case ruleFileTarget:
                for _, t := range r.targets {
                        if t == target {
                                matched, m = true, &match{
                                        target: t,
                                }
                        }
                }
        case rulePercentPattern:
                for _, pat := range r.targets {
                        if pos := strings.Index(pat, "%"); 0 <= pos {
                                prefix, suffix := pat[0:pos], pat[pos+1:]
                                if strings.HasPrefix(target, prefix) && strings.HasSuffix(target, suffix) {
                                        matched, m = true, &match{ target:target }
                                        m.stem = strings.TrimSuffix(strings.TrimPrefix(target, prefix), suffix)
                                        return
                                }
                        } else {
                                errorf(fmt.Sprintf("invalid pattern '%v'", pat))
                        }
                }
        case ruleRegexPattern:
        case ruleGlobPattern:
        }
        return
}

func (r *rule) findPrevRule(m *match) (prev *rule) {
        prev, _ = r.prev[m.target]
        return
}

func (r *rule) check(ctx *Context, m *match) (needsUpdate bool) {
        for !needsUpdate && r != nil {
                if needsUpdate = r.c.check(ctx, r, m); needsUpdate {
                        break
                }
                r = r.findPrevRule(m)
        }
        return needsUpdate
}

func (r *rule) update(ctx *Context, m *match) (updated bool) {
        updated = r.c.update(ctx, r, m)

        // TODO: update in the namespace instead, supporting multipart names (a.b.c)
        if m, ok := ctx.modules[m.target]; ok && m != nil {
                updated = m.update(ctx) || updated
        }
        return
}

func (r *rule) updateAll(ctx *Context) bool {
        var num = 0
        for _, t := range r.targets {
                m := &match{ target:t }
                if r.update(ctx, m) { num++ }
        }
        return 0 < num
}

// targetsFoundable returns true if targets are all foundable or generatable.
func (r *rule) targetsFoundable(ctx *Context, m *match) bool {
        switch r.kind {
        case rulePercentPattern:
                if fi, e := os.Stat(m.target); e == nil && fi != nil {
                        return true
                }

                for _, prerequisite := range r.prerequisites {
                        target, _ := m.unstem(prerequisite)
                        matchrules := r.ns.findMatchRules(ctx, target)
                        
                        if 0 < len(matchrules) {
                                fmt.Printf("targetsFoundable: %v: %v (%v)\n", m.target, target, m)
                        } else if fi, e := os.Stat(target); e == nil && fi != nil{
                                //fmt.Printf("targetsFoundable: %v: %v (%v)\n", m.target, target, m)
                                return true
                        }
                }
                
        default:
                for _, t := range r.targets {
                        if !ctx.targetFoundable(t) {
                                return false
                        }
                }
                return true
        }
        return false
}

func (r *rule) updatePrerequisites(ctx *Context, m *match) (err error, matchedPrerequisites, updatedPrerequisites []*matchrule) {
        var foundMatchRules [][]*matchrule
        //fmt.Printf("%v: %v\n", r.targets, r.prerequisites)
        for _, prerequisite := range r.prerequisites {
                target, ispat := m.unstem(prerequisite)
                matchrules := r.ns.findMatchRules(ctx, target)
        
                if 0 < len(matchrules) {
                        var (
                                validMatchrules []*matchrule
                                validPatternRules = 0
                        )
                        for _, mr := range matchrules {
                                //fmt.Printf("%v: %v: %v\n", validPatternRules, mr.rule.targets, mr.rule.prerequisites)
                                switch mr.rule.kind {
                                case rulePercentPattern:
                                        if mr.rule.targetsFoundable(ctx, mr.match) {
                                                if validPatternRules++; validPatternRules == 1 {
                                                        validMatchrules = append(validMatchrules, mr)
                                                        //fmt.Printf("%v: %v: %v (appended)\n", validPatternRules, mr.rule.targets, mr.rule.prerequisites)
                                                }
                                        }
                                default:
                                        validMatchrules = append(validMatchrules, mr)
                                }
                        }
                        if validPatternRules == 0 {
                                // If no foundable pattern targets, add the first
                                // one no matter if it's foundable
                                for _, mr := range matchrules {
                                        if rulePercentPattern == mr.rule.kind {
                                                if validPatternRules++; validPatternRules == 1 {
                                                        validMatchrules = append(validMatchrules, mr)
                                                        //fmt.Printf("%v: %v: %v (appended)\n", validPatternRules, mr.rule.targets, mr.rule.prerequisites)
                                                }
                                        }
                                }
                        }
                        foundMatchRules = append(foundMatchRules, validMatchrules)
                } else if ispat {
                        if r.kind != rulePercentPattern {
                                err = errors.New(fmt.Sprintf("invalid pattern prerequisite '%v'", prerequisite))
                                return
                        }
                        if ctx.targetFoundable(target) {
                                foundMatchRules = append(foundMatchRules, []*matchrule{
                                        &matchrule{ &match{ target, m.stem }, nil },
                                })
                        } else {
                                err = errors.New(fmt.Sprintf("no rule to update prerequisite '%v' (%v)", target, prerequisite))
                                //fmt.Printf("updatePrerequisites: missing `%v` (%v)\n", target, prerequisite)
                                return
                        }
                } else {
                        err = errors.New(fmt.Sprintf("no rule to update prerequisite '%v' (%v)", target, prerequisite))
                        return
                }
        }
        for _, matchrules := range foundMatchRules {
                matchedPrerequisites = append(matchedPrerequisites, matchrules...)
                for _, mr := range matchrules {
                        if mr.rule == nil {
                                // .........
                                continue
                        }
                        if ok := mr.rule.update(ctx, mr.match); ok {
                                updatedPrerequisites = append(updatedPrerequisites, mr)
                                break
                        }
                }
        }
        return
}

func (r *rule) makeExecuteContext(ctx *Context, ti os.FileInfo, m *match, matchedPrerequisites []*matchrule) *ruleExecuteContext {
        ec := &ruleExecuteContext{ target: m.target, stem: m.stem }
        for _, mr := range matchedPrerequisites {
                ec.prerequisites = append(ec.prerequisites, mr.match.target)
                if ti != nil {
                        if fi, err := os.Stat(mr.match.target); err == nil {
                                if fi.ModTime().After(ti.ModTime()) {
                                        ec.newer = append(ec.newer, mr.match.target)
                                }
                        }
                }
        }
        //fmt.Printf("makeExecuteContext: prerequisites: %v %v %v\n", r.targets, ec.prerequisites, matchedPrerequisites)
        return ec
}

func targetDirBaseItems(targets []string) (items, itemsDir, itemsBase Items) {
        for _, target := range targets {
                items = append(items, stringitem(target))
                itemsDir = append(itemsDir, stringitem(filepath.Dir(target)))
                itemsBase = append(itemsBase, stringitem(filepath.Base(target)))
        }
        return
}

type ruleExecuteContext struct {
        target, stem string
        prerequisites, newer []string
}

// https://www.gnu.org/software/make/manual/html_node/Automatic-Variables.html#Automatic-Variables
//   $@ The file name of the target of the rule. If the target is an archive member, then ‘$@’ is the name of the archive file. In a pattern rule that has multiple targets (see Introduction to Pattern Rules), ‘$@’ is the name of whichever target caused the rule’s recipe to be run.
//   $% The target member name, when the target is an archive member. See Archives. For example, if the target is foo.a(bar.o) then ‘$%’ is bar.o and ‘$@’ is foo.a. ‘$%’ is empty when the target is not an archive member.
//   $< The name of the first prerequisite. If the target got its recipe from an implicit rule, this will be the first prerequisite added by the implicit rule (see Implicit Rules).
//   $? The names of all the prerequisites that are newer than the target, with spaces between them. For prerequisites which are archive members, only the named member is used (see Archives).
//   $^ The names of all the prerequisites, with spaces between them. For prerequisites which are archive members, only the named member is used (see Archives). A target has only one prerequisite on each other file it depends on, no matter how many times each file is listed as a prerequisite. So if you list a prerequisite more than once for a target, the value of $^ contains just one copy of the name. This list does not contain any of the order-only prerequisites; for those see the ‘$|’ variable, below.
//   $+ This is like ‘$^’, but prerequisites listed more than once are duplicated in the order they were listed in the makefile. This is primarily useful for use in linking commands where it is meaningful to repeat library file names in a particular order.
//   $| The names of all the order-only prerequisites, with spaces between them.
//   $* The stem with which an implicit rule matches (see How Patterns Match). If the target is dir/a.foo.b and the target pattern is a.%.b then the stem is dir/foo. The stem is useful for constructing names of related files.
//      In a static pattern rule, the stem is part of the file name that matched the ‘%’ in the target pattern.
//      In an explicit rule, there is no stem; so ‘$*’ cannot be determined in that way. Instead, if the target name ends with a recognized suffix (see Old-Fashioned Suffix Rules), ‘$*’ is set to the target name minus the suffix. For example, if the target name is ‘foo.c’, then ‘$*’ is set to ‘foo’, since ‘.c’ is a suffix. GNU make does this bizarre thing only for compatibility with other implementations of make. You should generally avoid using ‘$*’ except in implicit rules or static pattern rules.
//      If the target name in an explicit rule does not end with a recognized suffix, ‘$*’ is set to the empty string for that rule.
//      
//   $(@D) $(@F) $(*D) $(*F) $(%D) $(%F) $(<D) $(<F) $(^D) $(^F) $(+D) $(+F) $(?D) $(?F)
//   
func (r *rule) execute(ctx *Context, ec *ruleExecuteContext) error {
        ns := ctx.g
        
        saveIndex, _ := ns.saveDefines(
                "@", "@D", "@F",
                "%", "%D", "%F",
                "<", "<D", "<F",
                "?", "?D", "?F",
                "^", "^D", "^F",
                "+", "+D", "+F",
                "|", "|D", "|F",
                "*", "*D", "*F")
        defer ns.restoreDefines(saveIndex)

        ns.Set(ctx, []string{ "@" },  stringitem(ec.target))
        ns.Set(ctx, []string{ "@D" }, stringitem(filepath.Dir(ec.target)))
        ns.Set(ctx, []string{ "@F" }, stringitem(filepath.Base(ec.target)))
        ns.Set(ctx, []string{ "*" },  stringitem(ec.stem))
        ns.Set(ctx, []string{ "*D" }, stringitem(filepath.Dir(ec.stem)))
        ns.Set(ctx, []string{ "*F" }, stringitem(filepath.Base(ec.stem)))
        if 0 < len(ec.prerequisites) {
                l, ld, lf := targetDirBaseItems(ec.prerequisites)
                ns.Set(ctx, []string{ "<" },  l[0])
                ns.Set(ctx, []string{ "<D" }, ld[0])
                ns.Set(ctx, []string{ "<F" }, lf[0])
                ns.Set(ctx, []string{ "^" }, l...)
                ns.Set(ctx, []string{ "^D" }, ld...)
                ns.Set(ctx, []string{ "^F" }, lf...)
        }
        if 0 < len(ec.newer) {
                l, ld, lf := targetDirBaseItems(ec.newer)
                ns.Set(ctx, []string{ "?" }, l...)
                ns.Set(ctx, []string{ "?D" }, ld...)
                ns.Set(ctx, []string{ "?F" }, lf...)
        }
        
        job := new(executeRecipes)
        for _, action := range r.recipes {
                var s string
                switch a := action.(type) {
                case string: s = a
                case *node: s = a.Expand(ctx)
                }
                job.recipes = append(job.recipes, s)
        }
        /*
        if *flagJ <= 1 {
                job.Action()
        } else {
                ctx.w.Do(job)
        } */
        job.Action()
        return job.error
}

type executeRecipes struct {
        recipes []string
        error error
}
func (job *executeRecipes) Action() worker.Result {
        for _, s := range job.recipes {
                echo := true
                if s[0] == '@' {
                        s, echo = s[1:], false
                }
                if cmd := exec.Command("sh", "-c", s); cmd != nil {
                        cmd.Stdout, cmd.Stderr = recipeStdout, recipeStderr
                        if echo { fmt.Fprintf(recipeEcho, "%v\n", s) }
                        if job.error = cmd.Run(); job.error != nil {
                                break
                        } else {
                                //fmt.Printf("rule: %v\n", s)
                        }
                } else {
                        errorf("nil command `sh`")
                }
        }
        return nil
}

// Update updates the specified targets given in `cmds`.
//
// Example (TODO):
//      
//      # Updates global target 'foo.txt'
//      smart -g foo.txt
//      
//      # Updates module foo's target 'bar.txt'
//      smart foo:bar.txt
//      
//      # Updates module 'foobar'
//      smart -m foobar
//      
//      # Updates both module and global 'foobar'
//      smart foobar
// 
func Update(ctx *Context, cmds ...string) {
        ctx.w.SpawnN(*flagJ); defer ctx.w.KillAll()

        if n := len(cmds); n == 0 {
                if goal := ctx.g.goal; goal == "" {
                        for _, m := range ctx.moduleOrderList { 
                                m.update(ctx)
                        }
                } else {
                        ctx.update(goal)
                }
        } else {
                for _, cmd := range cmds {
                        ctx.update(cmd)
                }
        }
}

// Build builds the project with specified variables and commands.
func Build(vars map[string]string, cmds ...string) (ctx *Context) {
        defer func() {
                if e := recover(); e != nil {
                        if se, ok := e.(*smarterror); ok {
                                fmt.Printf("smart: %v\n", se.message)
                                os.Exit(-1)
                        } else {
                                panic(e)
                        }
                }
        }()

        var (
                d string
                err error
        )
        if d = *flagC; d == "" { d = "." }

        init := new(bytes.Buffer)
        for _, s := range initScript {
                init.WriteString(s)
        }

        ctx, err = NewContext("init", init.Bytes(), vars)
        if err != nil {
                fmt.Printf("smart: %v\n", err)
                return
        }

        // Find and process modules.
        err = traverse(d, func(fn string, fi os.FileInfo) bool {
                fr := matchFileInfo(fi, generalMetaFiles)
                if *flagGG && fr != nil { return false }
                if fi.Name() == ".smart" {
                        if err := ctx.include(fn); err != nil {
                                errorf("include: `%v', %v\n", fn, err)
                        }
                }
                return true
        })
        if err != nil {
                fmt.Printf("error: %v\n", err)
        }

        Update(ctx, cmds...)
        return
}
