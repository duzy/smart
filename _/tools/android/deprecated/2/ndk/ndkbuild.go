//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "fmt"
        "bytes"
        "os"
        "os/exec"
        "io/ioutil"
        "path/filepath"
        //"runtime"
        "strings"
        . "github.com/duzy/smart/build"
)

func init() {
        ndk := &toolset{}
        RegisterToolset("ndk-build", ndk)

        if c, e := exec.LookPath("ndk-build"); e == nil {
                ndk.root = filepath.Dir(c)
        } else {
                if ndk.root = os.Getenv("ANDROIDNDK"); ndk.root == "" {
                        Message("cant locate Android NDK: %v", e)
                }
        }

        if ndk.root == "" {
                return
        }

        //Message("ndk-build: %v", ndk.root)
}

type toolset struct {
        BasicToolset
        root string
}

func (ndk *toolset) ConfigModule(ctx *Context, args Items, vars map[string]string) {
        var (
                abi = "armeabi" // all
                optim = "debug" // release|debug
                platform = "android-9" // minimal is "android-8"
                script = "out/boot.mk" // use BUILD_SCRIPT to override
                stl = "system" // system|stlport_static|gnustl_static
        )

        // APP_BUILD_SCRIPT, APP_PLATFORM, APP_STL, APP_ABI, APP_OPTIM
        if s, ok := vars["BUILD_SCRIPT"]; ok { script = s; ctx.Set("me.is_custom_script", StringItem("yes")) }
        if s, ok := vars["PLATFORM"];     ok { platform = s }
        if s, ok := vars["STL"];          ok { stl = s }
        if s, ok := vars["ABI"];          ok { abi = s }
        if s, ok := vars["OPTIM"];        ok { optim = s }

        ctx.Set("me.abi",      StringItem(strings.TrimSpace(abi)))
        ctx.Set("me.optim",    StringItem(strings.TrimSpace(optim)))
        ctx.Set("me.platform", StringItem(strings.TrimSpace(platform)))
        ctx.Set("me.script",   StringItem(strings.TrimSpace(script)))
        ctx.Set("me.stl",      StringItem(strings.TrimSpace(stl)))

        //m := ctx.CurrentModule()
        //Message("ndk-build: config: name=%v, dir=%v, args=%v, vars=%v", m.Name, m.GetDir(), args, vars)
        //Message("ndk-build: args=%v, vars=%v", args, vars)
}

func (ndk *toolset) CreateActions(ctx *Context) bool {
        m := ctx.CurrentModule()

        cmd := &delegate{
                abis:     strings.Fields(ctx.Call("me.abi").Expand(ctx)),
                abi:      ctx.Call("me.abi").Expand(ctx),
                script:   ctx.Call("me.script").Expand(ctx),
                platform: ctx.Call("me.platform").Expand(ctx),
                stl:      ctx.Call("me.stl").Expand(ctx),
                optim:    ctx.Call("me.optim").Expand(ctx),
        }
        if !filepath.IsAbs(cmd.script) {
                s, _, _ := m.GetDeclareLocation()
                cmd.script = filepath.Join(filepath.Dir(s), cmd.script)
        }

        targets, prerequisites := make(map[string]int, 4), make(map[string]int, 16)

        m.Action = new(Action)
        m.Action.Command = cmd
        m.Action.Prerequisites = append(m.Action.Prerequisites, NewAction(cmd.script, nil))
        prerequisites[cmd.script]++

        var scripts []string
        if ctx.Call("me.is_custom_script").Expand(ctx) == "yes" {
                scripts = append(scripts, cmd.script)
        } else {
                pa := m.Action.Prerequisites[0]
                pa.Command = &genBuildScript{}

                var e error
                if scripts, e = FindFiles(m.GetDir(ctx), `Android\.mk$`); e == nil {
                        for _, s := range scripts {
                                if filepath.Base(s) != "Android.mk" { continue }
                                pa.Prerequisites = append(pa.Prerequisites, NewAction(s, nil))
                        }
                }
        }

        for _, s := range cmd.abis {
                dump := cmd.dumpAll(s, scripts) //; fmt.Printf("dump: %v\n", dump)
                for _, p := range dump.m {
                        // skip standard modules, e.g. stdc++
                        if strings.HasPrefix(p.path, dump.ndkRoot) { continue }

                        /* fmt.Printf("target: %v: %v\n", s, p.built)
                        fmt.Printf("target: %v: %v\n", s, p.installed) */

                        targets[strings.TrimPrefix(p.built, "./")]++
                        prerequisites[strings.TrimPrefix(p.script, "./")]++
                        for _, s := range p.sources {
                                prerequisites[filepath.Join(p.path, s)]++
                        }
                }
        }

        //fmt.Printf("targets: %v\n", targets)

        for s, _ := range targets {
                m.Action.Targets = append(m.Action.Targets, s)
        }
        for s, _ := range prerequisites {
                m.Action.Prerequisites = append(m.Action.Prerequisites, NewAction(s, nil))
        }

        return true
}

type moduleInfo struct {
        name, filename, path, script, objsDir, built, installed, class string
        sources []string
}

type dump struct {
        ndkRoot, targetOut, targetObjs, targetGdbSetup, targetGdbServer string
        modules []string
        m map[string]moduleInfo
}

type genBuildScript struct {}

func (g *genBuildScript) Execute(targets []string, prerequisites []string) bool {
        //Message("ndkbuild: GenBuildScript: %v: %v", targets, prerequisites)

        buf := new(bytes.Buffer)
        fmt.Fprintf(buf, "# %v\n", targets)
        for _, p := range prerequisites {
                fmt.Fprintf(buf, "include %s\n", p)
        }

        for _, t := range targets {
                if e := os.MkdirAll(filepath.Dir(t), os.FileMode(0755)); e != nil {
                        Message("ndkbuild: GenBuildScript: %v", e)
                        return false
                }

                f, e := os.Create(t)
                if e != nil {
                        Message("ndkbuild: GenBuildScript: %v", e)
                        return false
                }
                defer f.Close()
                if _, e := f.Write(buf.Bytes()); e != nil {
                        Message("ndkbuild: GenBuildScript: %v", e)
                        return false
                }
        }
        return true
}

type genDumpScript struct {}

func (g *genDumpScript) Execute(targets []string, prerequisites []string) bool {
        Message("ndkbuild: GenDumpScript: %v: %v", targets, prerequisites)

        buf := new(bytes.Buffer)
        fmt.Fprintf(buf, "# %v\n", targets)
        for _, p := range prerequisites {
                fmt.Fprintf(buf, "include %s\n", p)
        }

        for _, t := range targets {
                f, e := os.Create(t)
                if e != nil {
                        Message("ndkbuild: GenBuildScript: %v", e)
                        return false
                }
                defer f.Close()
                if _, e := f.Write(buf.Bytes()); e != nil {
                        Message("ndkbuild: GenBuildScript: %v", e)
                        return false
                }
        }
        return true
}

type delegate struct {
        script, abi, platform, stl, optim string
        abis []string
}

func (n *delegate) Execute(targets []string, prerequisites []string) bool {
        //Message("ndkbuild: %v: %v", targets, prerequisites)

        vars := n.getBuildVars(n.abi, n.script)
        //fmt.Printf("%v\n", vars)

        c := NewExcmd("ndk-build")
        return c.Run(fmt.Sprintf("%v", targets), vars...)
}

func (n *delegate) getBuildVars(abi string, script string) []string {
        return []string{
                fmt.Sprintf("NDK_PROJECT_PATH=%s", "."),
                //fmt.Sprintf("NDK_MODULE_PATH=%s", "."),
                //fmt.Sprintf("NDK_LIBS_OUT=%s", "libs"),
                fmt.Sprintf("NDK_OUT=%s", "out"),
                fmt.Sprintf("APP_BUILD_SCRIPT=%s", script),
                fmt.Sprintf("APP_ABI=%s", abi),
                fmt.Sprintf("APP_PLATFORM=%s", n.platform),
                fmt.Sprintf("APP_STL=%s", n.stl),
                fmt.Sprintf("APP_OPTIM=%s", n.optim),
        }
}

func (n *delegate) dumpAll(abi string, scripts []string) (res *dump) {
        res = &dump{ m:make(map[string]moduleInfo) }

        tf := n.createSmartDumpFile(scripts); defer os.Remove(tf)

        vars := n.getBuildVars(abi, tf)
        vars = append(vars, filepath.Base(tf))

        c := NewExcmd("ndk-build")
        if c.Run("" /*fmt.Sprintf("%s (%s)", filepath.Base(tf), abi)*/, vars...) {
                //fmt.Printf("%v", c.GetStdout().String())
                ctx, e := NewContext("DummyDump", c.GetStdout().Bytes(), nil)
                if e != nil { Fatal("DummyDump: %v", e) }
                res.ndkRoot = ctx.Call("NDK_ROOT").Expand(ctx)
                res.targetOut = ctx.Call("TARGET_OUT").Expand(ctx)
                res.targetObjs = ctx.Call("TARGET_OBJS").Expand(ctx)
                res.targetGdbSetup = ctx.Call("TARGET_GDB_SETUP").Expand(ctx)
                res.targetGdbServer = ctx.Call("TARGET_GDB_SERVER").Expand(ctx)
                res.modules = strings.Fields(ctx.Call("MODULES").Expand(ctx))
                for _, s := range res.modules {
                        m := &moduleInfo{}
                        m.name = ctx.Call(s+"_NAME").Expand(ctx)
                        m.filename = ctx.Call(s+"_FILENAME").Expand(ctx)
                        m.path = ctx.Call(s+"_PATH").Expand(ctx)
                        m.sources = strings.Fields(ctx.Call(s+"_SOURCES").Expand(ctx))
                        m.script = ctx.Call(s+"_SCRIPT").Expand(ctx)
                        m.objsDir = ctx.Call(s+"_OBJS_DIR").Expand(ctx)
                        m.built = ctx.Call(s+"_BUILT").Expand(ctx)
                        m.installed = ctx.Call(s+"_INSTALLED").Expand(ctx)
                        m.class = ctx.Call(s+"_CLASS").Expand(ctx)
                        res.m[s] = *m
                }
        }
        return
}

func (n *delegate) dumpSingle(abi string) (res *dump) {
        res = &dump{}

        tf := n.createDummyDumpFile(); defer os.Remove(tf)

        c := NewExcmd("ndk-build")

        vars := []string{
                fmt.Sprintf("NDK_PROJECT_PATH=%s", "."),
                //fmt.Sprintf("NDK_MODULE_PATH=%s", "."),
                //fmt.Sprintf("NDK_APP_NAME=%s", "."),
                //fmt.Sprintf("NDK_LIBS_OUT=%s", "libs"),
                fmt.Sprintf("NDK_OUT=%s", "out"),
                fmt.Sprintf("APP_BUILD_SCRIPT=%s", tf),
                fmt.Sprintf("APP_ABI=%s", abi),
                fmt.Sprintf("APP_PLATFORM=%s", n.platform),
                fmt.Sprintf("APP_STL=%s", n.stl),
                fmt.Sprintf("APP_OPTIM=%s", n.optim),
                "dummy-dump",
        }

        if c.Run("", vars...) {
                //fmt.Printf( "%v", c.GetStdout().String() )
                ctx, e := NewContext("DummyDump", c.GetStdout().Bytes(), nil)
                if e != nil { Fatal("DummyDump: %v", e) }
                //res.appName = ctx.Call("NDK_APP_NAME")
                //res.module = ctx.Call("LOCAL_MODULE")
                //res.moduleClass = ctx.Call("LOCAL_MODULE_CLASS")
                //res.srcFiles = ctx.Call("LOCAL_SRC_FILES")
                //res.builtModule = ctx.Call("LOCAL_BUILT_MODULE")
                res.targetOut = ctx.Call("TARGET_OUT").Expand(ctx)
                res.targetObjs = ctx.Call("TARGET_OBJS").Expand(ctx)
                res.targetGdbSetup = ctx.Call("TARGET_GDB_SETUP").Expand(ctx)
                res.targetGdbServer = ctx.Call("TARGET_GDB_SERVER").Expand(ctx)
                //fmt.Printf( "%v\n", ctx.variables )
                //fmt.Printf( "%v\n", ctx.Call("LOCAL_OBJECTS") )
        }

        return
}

func (n *delegate) createDummyDumpFile() string {
        tf, e := ioutil.TempFile("/tmp", "smart-dummy-dump-")
        if e != nil {
                Fatal("TempFile: %v", e)
        }

        defer tf.Close()

        //$(call import-module,third_party/googletest)
        //$(call import-module,native_app_glue)

        fmt.Fprintf(tf, `# dummy
include %s
dummy-dump: DUMMY_LOCAL_MODULE := $(LOCAL_MODULE)
dummy-dump: DUMMY_LOCAL_MODULE_CLASS := $(LOCAL_MODULE_CLASS)
dummy-dump: DUMMY_LOCAL_SRC_FILES := $(LOCAL_SRC_FILES)
dummy-dump: DUMMY_LOCAL_OBJECTS := $(LOCAL_OBJECTS)
dummy-dump: DUMMY_LOCAL_RS_OBJECTS := $(LOCAL_RS_OBJECTS)
dummy-dump: DUMMY_LOCAL_BUILT_MODULE := $(LOCAL_BUILT_MODULE)
dummy-dump: DUMMY_LOCAL_INSTALLED := $(LOCAL_INSTALLED)
dummy-dump: DUMMY_TARGET_OUT := $(TARGET_OUT)
dummy-dump: DUMMY_TARGET_OBJS := $(TARGET_OBJS)
dummy-dump: DUMMY_TARGET_GDB_SETUP := $(TARGET_GDB_SETUP)
dummy-dump: DUMMY_TARGET_GDB_SERVER := $(TARGET_GDBSERVER)
dummy-dump: DUMMY_TARGET_SONAME_EXTENSION := $(TARGET_SONAME_EXTENSION)
dummy-dump: DUMMY_NDK_APP_NAME := $(NDK_APP_NAME)
dummy-dump:
	@echo "LOCAL_MODULE :=$(DUMMY_LOCAL_MODULE)"
	@echo "LOCAL_MODULE_CLASS :=$(DUMMY_LOCAL_MODULE_CLASS)"
	@echo "LOCAL_SRC_FILES :=$(DUMMY_LOCAL_SRC_FILES)"
	@echo "LOCAL_OBJECTS :=$(DUMMY_LOCAL_OBJECTS)"
	@echo "LOCAL_RS_OBJECTS :=$(DUMMY_LOCAL_RS_OBJECTS)"
	@echo "LOCAL_BUILT_MODULE :=$(DUMMY_LOCAL_BUILT_MODULE)"
	@echo "LOCAL_INSTALLED :=$(DUMMY_LOCAL_INSTALLED)"
	@echo "TARGET_OUT :=$(DUMMY_TARGET_OUT)"
	@echo "TARGET_LIBS :=$(DUMMY_TARGET_LIBS)"
	@echo "TARGET_OBJS :=$(DUMMY_TARGET_OBJS)"
	@echo "TARGET_GDB_SETUP :=$(DUMMY_TARGET_GDB_SETUP)"
	@echo "TARGET_GDB_SERVER :=$(DUMMY_TARGET_GDB_SERVER)"
	@echo "TARGET_SONAME_EXTENSION :=$(DUMMY_TARGET_SONAME_EXTENSION)"
	@echo "NDK_APP_NAME :=$(DUMMY_NDK_APP_NAME)"
	@echo
`, n.script)
        return tf.Name()
}

func (n *delegate) createSmartDumpFile(scripts []string) string {
        tf, e := ioutil.TempFile("/tmp", "smart-dummy-dump-")
        if e != nil {
                Fatal("TempFile: %v", e)
        }

        defer tf.Close()

        for _, s := range scripts {
                fmt.Fprintf(tf, "include %s\n", s)
        }
        fmt.Fprintf(tf, `# $(info $(modules-dump-database))
~ := %s
$~: DUMMY_TARGET_OUT := $(TARGET_OUT)
$~: DUMMY_TARGET_OBJS := $(TARGET_OBJS)
$~: DUMMY_TARGET_GDB_SETUP := $(TARGET_GDB_SETUP)
$~: DUMMY_TARGET_GDB_SERVER := $(TARGET_GDBSERVER)
$~: DUMMY_MODULES := $(modules-get-list)
$~:
	@echo "NDK_ROOT :=$(NDK_ROOT)"
	@echo "TARGET_OUT :=$(DUMMY_TARGET_OUT)"
	@echo "TARGET_OBJS :=$(DUMMY_TARGET_OBJS)"
	@echo "TARGET_GDB_SETUP :=$(DUMMY_TARGET_GDB_SETUP)"
	@echo "TARGET_GDB_SERVER :=$(DUMMY_TARGET_GDB_SERVER)"
	@echo "MODULES :=$(DUMMY_MODULES)"
	@$(foreach s,$(DUMMY_MODULES),\
echo "$(s)_NAME := $(__ndk_modules.$s.MODULE)" &&\
echo "$(s)_FILENAME := $(__ndk_modules.$s.MODULE_FILENAME)" &&\
echo "$(s)_PATH := $(__ndk_modules.$s.PATH)" &&\
echo "$(s)_SOURCES := $(__ndk_modules.$s.SRC_FILES)" &&\
echo "$(s)_SCRIPT := $(__ndk_modules.$s.MAKEFILE)" &&\
echo "$(s)_OBJS_DIR := $(__ndk_modules.$s.OBJS_DIR)" &&\
echo "$(s)_BUILT := $(__ndk_modules.$s.BUILT_MODULE)" &&\
echo "$(s)_INSTALLED := $(__ndk_modules.$s.INSTALLED)" &&\
echo "$(s)_CLASS := $(__ndk_modules.$s.MODULE_CLASS)" &&\
) true
	@echo
`, filepath.Base(tf.Name()))
        return tf.Name()
}
