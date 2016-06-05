//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        //"bytes"
        "archive/zip"
        "fmt"
        "os"
        "path/filepath"
        "runtime"
        "strings"
        "io/ioutil"
        "io"
        "os/exec"
        . "github.com/duzy/smart/build"
)

var (
        sdk = "/android-sdk-linux_x86"
        sdkBuildToolVersion = "23.0.2"
        sdkDefaultPlatform = "android-23" // "android-10"
        sdkSourceCompatibility = "1.7" // gradle: sourceCompatibility = 1.7
        sdkTargetCompatibility = "1.7" // gradle: targetCompatibility = 1.7
)

func init() {
        RegisterToolset("android-sdk", &toolset{})

        if c, e := exec.LookPath("android"); e == nil {
                sdk = filepath.Dir(filepath.Dir(c))
        } else if c, e := exec.LookPath("aapt"); e == nil {
                sdk = filepath.Dir(filepath.Dir(c))
        } else {
                if sdk = os.Getenv("ANDROIDSDK"); sdk == "" {
                        fmt.Printf("can't locate Android SDK: %v\n", e)
                }
        }
}

func getBuildTool(name string) string {
        version := sdkBuildToolVersion
        return filepath.Join(sdk, "build-tools", version, name)
}

func getPlatformTool(name string) string {
        return filepath.Join(sdk, "platform-tools", name)
}

func getTool(name string) string {
        return filepath.Join(sdk, "tools", name)
}

type toolset struct { BasicToolset }

func (sdk *toolset) getResourceFiles(ds ...string) (as []*Action) {
        for _, d := range ds {
                Traverse(d, func(fn string, fi os.FileInfo) bool {
                        if !strings.HasSuffix(fn, "~") && !fi.IsDir() {
                                as = append(as, NewAction(fn, nil))
                        }
                        return true
                })
        }
        return
}

func (sdk *toolset) ConfigModule(ctx *Context, args Items, vars map[string]string) {
        var (
                kind, platform string
                sources []string
                err error
        )
        if 0 < len(args) { kind = strings.TrimSpace(args[0].Expand(ctx)) }
        if s, ok := vars["PLATFORM"]; ok { platform = s } else { platform = sdkDefaultPlatform }
        if kind == "external" {
                // ...
        } else {
                d := filepath.Dir(ctx.CurrentScope())
                sources, err = FindFiles(filepath.Join(d, "src"), `\.java$`)
                if len(sources) == 0 {
                        sources, err = FindFiles(filepath.Join(d, "java"), `\.java$`)
                }
                for i := range sources {
                        sources[i] = strings.TrimPrefix(sources[i], d)
                }

                //fmt.Printf("sources: (%v) %v\n", d, sources)

                if err != nil {
                        Fatal(fmt.Sprintf("no Java sources in `%v'", d))
                }

                ctx.Set("me.sources", StringItem(strings.Join(sources, " ")))
        }
        ctx.Set("me.platform", StringItem(platform))
        ctx.Set("me.kind", StringItem(kind))
}

func (sdk *toolset) CreateActions(ctx *Context) bool {
        m, platform := ctx.CurrentModule(), strings.TrimSpace(ctx.Call("me.platform").Expand(ctx))
        if platform == "" { Fatal("no platform selected (%v)", m.GetName(ctx)) }

        //fmt.Printf("platform: %v\n", platform)

        gen := &basicGen{
                d:filepath.Dir(ctx.CurrentScope()),
                out:filepath.Join("out", m.GetName(ctx)),
                platform:platform,
        }

        var a *Action
        var prerequisites []*Action
        var staticLibs []string
        var hasRes, hasAssets, hasSrc bool
        if fi, err := os.Stat(filepath.Join(gen.d, "src")); err == nil && fi.IsDir() { hasSrc = true }
        if fi, err := os.Stat(filepath.Join(gen.d, "res")); err == nil && fi.IsDir() { hasRes = true }
        if fi, err := os.Stat(filepath.Join(gen.d, "assets")); err == nil && fi.IsDir() { hasAssets = true }
        if hasRes { gen.res = filepath.Join(gen.d, "res") }
        if hasAssets { gen.assets = filepath.Join(gen.d, "assets") }
        if hasRes || hasAssets {
                a = NewInterAction("R.java", &genResJavaFiles{ basicGen:gen },
                        sdk.getResourceFiles(gen.res, gen.assets)...)
        }
        if hasSrc {
                var ps []*Action
                if a != nil { ps = append(ps, a) }
                if sources := m.GetSources(ctx); 0 < len(sources) {
                        var classpath []string
                        for _, u := range m.Using {
                                if strings.ToLower(u.Get(ctx, "kind")) != "jar" { Fatal("using `%v' module", u.Get(ctx, "kind")) }
                                ctx.With(u, func() {
                                        classpath = append(classpath, strings.Fields(ctx.Call("me.export.jar").Expand(ctx))...)
                                        classpath = append(classpath, strings.Fields(ctx.Call("me.export.libs.static").Expand(ctx))...)
                                })
                        }

                        staticLibs = append(staticLibs, strings.Fields(ctx.Call("me.static_libs").Expand(ctx))...)
                        classpath = append(classpath, strings.Fields(ctx.Call("me.classpath").Expand(ctx))...)
                        classpath = append(classpath, staticLibs...)

                        for _, src := range sources { ps = append(ps, NewAction(src, nil)) }
                        c := &genClasses{
                                basicGen:gen,
                                sourcepath:filepath.Join(gen.d, "src"),
                                classpath:classpath,
                        }
                        a = NewInterAction("*.class", c, ps...)
                }
        }

        if a != nil {
                prerequisites = append(prerequisites, a)
        }

        switch strings.ToLower(m.Get(ctx, "kind")) {
        case "apk":
                c := &genAPK{genTar{ basicGen:gen, target:filepath.Join(gen.out, m.GetName(ctx)+".apk"), staticlibs:staticLibs, }}
                m.Action = NewInterAction(m.GetName(ctx)+".apk", c, prerequisites...)
        case "jar":
                c := &genJAR{genTar{ basicGen:gen, target:filepath.Join(gen.out, m.GetName(ctx)+".jar"), staticlibs:staticLibs, }}
                ctx.Set("me.export.jar", StringItem(c.target))
                m.Action = NewInterAction(m.GetName(ctx)+".jar", c, prerequisites...)
        case "external":
                Fatal("TODO: `%v' of `%v'", m.GetName(ctx), m.Get(ctx, "kind"))
        default:
                s, l, c := m.GetDeclareLocation()
                fmt.Printf("%v:%v:%v: `%v' of `%v'\n", s, l, c, m.GetName(ctx), m.Get(ctx, "kind"))

                s, l, c = m.GetCommitLocation()
                fmt.Printf("%v:%v:%v: `%v'\n", s, l, c, m.GetName(ctx))

                Fatal("unknown type `%v'", m.Get(ctx, "kind"))
        }
        return true
}

type basicGen struct{
        platform, out, d, res, assets string
}

func (gen *basicGen) aapt_S() (args []string) {
        s := filepath.Join(gen.d, "res")
        if fi, err := os.Stat(s); err == nil && fi != nil && fi.IsDir() {
                args = append(args, "-S", s)
        }
        return
}

func (gen *basicGen) aapt_A() (args []string) {
        s := filepath.Join(gen.d, "assets")
        if fi, err := os.Stat(s); err == nil && fi != nil && fi.IsDir() {
                args = append(args, "-A", s)
        }
        return
}

type genResJavaFiles struct{
        *basicGen
        r string // "r" holds the R.java file path
        outdates int
}
func (ic *genResJavaFiles) Targets(prerequisites []*Action) (targets []string, needsUpdate bool) {
        if ic.r != "" {
                targets = []string{ ic.r }
                needsUpdate = 0 < ic.outdates
                return
        }

        targets, outdates, _ := ComputeInterTargets(filepath.Join(ic.out, "res"), `R\.java$`, prerequisites)
        if 0 < len(targets) { ic.r = targets[0] }

        needsUpdate = ic.r == "" || 0 < outdates
        return
}
func (ic *genResJavaFiles) Execute(targets []string, prerequisites []string) bool {
        ic.r = ""

        outRes := filepath.Join(ic.out, "res")
        os.RemoveAll(outRes)

        args := []string{
                "package", "-m",
                "-J", filepath.Join(ic.out, "res"),
                "-M", filepath.Join(ic.d, "AndroidManifest.xml"),
                "-I", filepath.Join(sdk, "platforms", ic.platform, "android.jar"),
        }

        if ic.res != ""    { args = append(args, "-S", ic.res) }
        if ic.assets != "" { args = append(args, "-A", ic.assets) }
        // TODO: -P -G

        c := NewExcmd(getBuildTool("aapt"))
        c.SetIA32(IsIA32Command(c.GetPath()))
        c.SetMkdir(outRes)
        /*
        if GetFlagV() {
                if ic.res != ""    { Verbose("resources `%v'", ic.res) }
                if ic.assets != "" { Verbose("assets `%v'", ic.assets) }
        } */
        //args = append(args, "--min-sdk-version", "7")
        //args = append(args, "--target-sdk-version", "7")
        if !c.Run("resources", args...) {
                Fatal("resources: %v", outRes)
        }

        if ic.r = FindFile(outRes, `R\.java$`); ic.r == "" {
                Fatal("resources: R.java not found")
        }
        return ic.r != ""
}

type genClasses struct{
        *basicGen
        sourcepath string
        classpath []string // holds the *.class file
        outdates int
}
func (ic *genClasses) Targets(prerequisites []*Action) (targets []string, needsUpdate bool) {
        targets, outdates, _ := ComputeInterTargets(filepath.Join(ic.out, "classes"), `\.class$`, prerequisites)
        needsUpdate = len(targets) == 0 || 0 < outdates
        //fmt.Printf("classes: %v\n", targets);
        return
}
func (ic *genClasses) Execute(targets []string, prerequisites []string) bool {
        classpath := filepath.Join(sdk, "platforms", ic.platform, "android.jar")
        if 0 < len(ic.classpath) {
                classpath += ":" + strings.Join(ic.classpath, ":")
        }

        outClasses := filepath.Join(ic.out, "classes")
        os.RemoveAll(outClasses)

        args := []string {
                "-d", filepath.Join(ic.out, "classes"),
                "-sourcepath", ic.sourcepath,
                "-cp", classpath,
        }

        args = append(args, "-source", sdkSourceCompatibility)
        args = append(args, "-target", sdkTargetCompatibility)
        args = append(args, prerequisites...)
        c := NewExcmd("javac")
        c.SetMkdir(outClasses)
        if !c.Run("classes", args...) {
                Fatal("classes: %v", outClasses)
                return false
        }

        return true
}

// createDummyPackage is deprecated
func createDummyPackage(name string) bool {
        if f, e := os.Create(filepath.Join(filepath.Dir(name), "dummy")); e == nil {
                f.Close()
        } else {
                return false
        }

        c := NewExcmd("jar")
        c.SetDir(filepath.Dir(name))
        if !c.Run("DummyPackage", "cf", filepath.Base(name), "dummy") {
                return false
        }

        if e := os.Remove(filepath.Join(filepath.Dir(name), "dummy")); e != nil {
                Fatal("remove: %v (%v)\n", "dummy", e)
        }

        c = NewExcmd("zip")
        c.SetDir(filepath.Dir(name))
        if !c.Run("DummyPackage", "-qd", filepath.Base(name), "dummy") {
                return false
        }

        return true
}

type genTar struct {
        *basicGen
        target string
        staticlibs []string
}
type genAPK struct { genTar }
type genJAR struct { genTar }

func (ic *genTar) Targets(prerequisites []*Action) (targets []string, needsUpdate bool) {
        if ic.target == "" {
                Fatal("unknown APK name")
        }
        afi, _ := os.Stat(ic.target)
        newerCount := 0
        Traverse(filepath.Join(ic.out, "classes"), func(fn string, fi os.FileInfo) bool {
                if afi == nil { return false }
                if strings.HasSuffix(fn, ".class") && !fi.IsDir() {
                        if afi.ModTime().Before(fi.ModTime()) {
                                newerCount ++
                        }
                }
                return true
        })
        targets = []string{ ic.target }
        needsUpdate = afi == nil || 0 < newerCount
        return
}

func (ic *genTar) createPackage(packFilename string, files ...string) {
        var (
                f *os.File
                err error
        )

        f, err = os.Create(packFilename)
        if err != nil {
                Fatal("can't find keystore for sigining APK")
        }

        zw := zip.NewWriter(f)

        defer func() {
                if err := zw.Close(); err != nil {
                        Fatal("%v (%s)", err, f.Name())
                }

                f.Close()
        }()

        libsDir := filepath.Join(ic.d, "libs")
        if fi, err := os.Stat(libsDir); err != nil || !fi.IsDir() {
                return
        }

        err = filepath.Walk(libsDir, func(path string, fi os.FileInfo, err error) error {
                if err != nil { return err }
                if fi.IsDir() { return nil } // path == libsDir

                h, err := zip.FileInfoHeader(fi)
                if err != nil { return err }

                h.Method = zip.Deflate
                h.Name = filepath.Join("lib", strings.TrimPrefix(
                        strings.TrimPrefix(path, libsDir), "/"))

                z, err := zw.CreateHeader(h)
                if err != nil { return err }
                if fi.IsDir() { return nil }

                Message("package: %v(%v)", f.Name(), h.Name)

                l, err := os.Open(path)
                if err != nil { return err }
                defer l.Close()

                _, err = io.Copy(z, l)
                return err
        })

        if err != nil {
                Fatal("libs: %v", err)
        }
}

func (ic *genTar) packageAddFiles(packFilename string, files ...string) {
        var (
                f *os.File
                err error
        )

        f, err = os.OpenFile(packFilename, os.O_RDWR, 0)
        if err != nil {
                Fatal("can't find keystore for sigining APK")
        }

        zw := zip.NewWriter(f)

        defer func() {
                if err := zw.Close(); err != nil {
                        Fatal("%v (%s)", err, f.Name())
                }

                f.Close()
        }()
        
        for _, s := range files {
                fi, err := os.Stat(s)
                if err != nil { Fatal("%v", err) }

                h, err := zip.FileInfoHeader(fi)
                if err != nil { Fatal("package: add: %v", err) }

                // h.Name ???
                h.Method = zip.Deflate

                Message("package: %v(%v)", f.Name(), h.Name)

                z, err := zw.CreateHeader(h)
                if err != nil { Fatal("package: add: %v", err) }

                l, err := os.Open(s)
                if err != nil { Fatal("package: add: %v", err) }
                defer l.Close()

                _, err = io.Copy(z, l)
                if err != nil { Fatal("package: add: %v", err) }
        }
}

func getKeystore() (keystore, keypass, storepass string) {
        var canditates []string

        defaultPass := "smart.android"

        readpass := func(s, sn string) string {
                if f, e := os.Open(filepath.Join(filepath.Dir(s), sn)); e == nil {
                        defer f.Close()
                        if b, e := ioutil.ReadAll(f); e == nil {
                                return strings.TrimSpace(string(b))
                        }
                }
                return defaultPass
        }

        find := func() (s1, s2, s3 string) {
                for _, s := range canditates {
                        s1, s2, s3 = "", "", ""
                        if fi, e := os.Stat(s); e == nil && !fi.IsDir() {
                                s1, s2, s3 = s, readpass(s, "keypass"), readpass(s, "storepass")
                                return
                        }
                }
                return
        }

        if wd, e := os.Getwd(); e == nil {
                for {
                        //fmt.Printf("wd: %v\n", wd)
                        canditates = []string{
                                filepath.Join(wd, ".androidsdk", "keystore"),
                        }
                        if s1, s2, s3 := find(); s1 != "" { keystore, keypass, storepass = s1, s2, s3; return }
                        if s := filepath.Dir(wd); wd == s || s == "" { break } else { wd = s }
                }
        }

        if s, e := exec.LookPath("smart"); e == nil {
                canditates = []string{
                        filepath.Join(filepath.Dir(s), "data", "androidsdk", "keystore"),
                }
                if s1, s2, s3 := find(); s1 != "" { keystore, keypass, storepass = s1, s2, s3; return }
        }

        return "", "", ""
}

func extractClasses(outclasses, lib string, cs []string) (classes []string) {
        f, err := os.Open(lib)
        if err != nil { Fatal("open: %v (%v)", lib, err) }
        defer f.Close()

        wd, err := os.Getwd()
        if err != nil {
                Fatal("getwd: %v", err)
                return
        }

        if e := os.Chdir(outclasses); e != nil {
                Fatal("chdir: %v", e)
                return
        }

        defer func() {
                if e := os.Chdir(wd); e != nil { Fatal("chdir: %v", e) }
        }()

        c := NewExcmd("jar")
        c.SetStdin(f)
        args := append([]string{ "-x" }, cs...)
        if !c.Run("classes", args...) { Fatal("static %v\n", lib) }

        for _, s := range cs {
                if fi, er := os.Stat(s); er != nil || fi == nil {
                        Fatal("class `%v' not extracted (%v)", s, lib); return
                }
                classes = append(classes, s)
        }

        return
}

func extractStaticLibsClasses(outclasses string, libs []string) (classes []string) {
        for _, lib := range libs {
                if lib = strings.TrimSpace(lib); lib == "" { continue }

                c := NewExcmd("jar")
                args := []string{ "-tf", lib }
                if !c.Run("classes", args...) { Fatal("static %v\n", lib) }

                var cs []string
                for _, s := range strings.Split(c.GetStdout().String(), "\n") {
                        if strings.HasSuffix(s, ".class") { cs = append(cs, s) }
                }

                //fmt.Printf("jar: %v: %v\n", lib, cs)

                classes = append(classes, extractClasses(outclasses, lib, cs)...)
        }
        //fmt.Printf("embeded-classes: %v\n", classes)
        return
}

func (ic *genAPK) Execute(targets []string, prerequisites []string) bool {
        outclasses := filepath.Join(ic.out, "classes")

        // extract classes from static libraries (me.static_libs)

        embclasses := extractStaticLibsClasses(outclasses, ic.staticlibs)
        //fmt.Printf("staticlibs: %v\n", embclasses)

        // make classes.dex

        args := []string {}
        if runtime.GOOS != "windows" { args = append(args, "-JXms16M", "-JXmx1536M") }
        args = append(args, "--dex", "--output=../classes.dex")

        countClasses := 0
        for _, s := range prerequisites {
                if s == "" { continue }
                if strings.HasPrefix(s, outclasses) {
                        args = append(args, s[len(outclasses)+1:])
                } else {
                        args = append(args, s)
                }
                countClasses ++
        }
        if countClasses == 0 { Fatal("no classes for `%v'", targets) }

        args = append(args, embclasses...) // add classes from static libraries

        c := NewExcmd(getBuildTool("dx"))
        c.SetDir(outclasses)
        if !c.Run("classes", args...) { Fatal("dex: %v\n", "classes.dex") }

        // Create new package unsigned.apk
        unsignedApk := filepath.Join(ic.out, "unsigned.apk")
        ic.createPackage(unsignedApk, filepath.Join(ic.out, "classes.dex")) // createDummyPackage

        // package resources into unsigned.apk

        c = NewExcmd(getBuildTool("aapt"))
        c.SetIA32(IsIA32Command(c.GetPath()))

        args = []string{ "package", "-u", "-F", unsignedApk,
                "-M", filepath.Join(ic.d, "AndroidManifest.xml"),
                "-I", filepath.Join(sdk, "platforms", ic.platform, "android.jar"),
        }
        if ic.res != ""    { args = append(args, "-S", ic.res) }
        if ic.assets != "" { args = append(args, "-A", ic.assets) }
        //args = append(args, "--min-sdk-version", "7")
        //args = append(args, "--target-sdk-version", "7")
        if GetFlagV() { Verbose("resources -> %v", targets) }
        if !c.Run("resources", args...) {
                Fatal("package: %v(%v)", unsignedApk, "assets,res,...")
        }

        //ic.packageAddFiles(unsignedApk, filepath.Join(ic.out, "classes.dex"))
        args = []string{ "add", "-k", unsignedApk, filepath.Join(ic.out, "classes.dex") }
        if !c.Run("classes", args...) {
                Fatal("package: %v(%v)", unsignedApk, "classes")
        }

        keystore, keypass, storepass := getKeystore()
        if keystore == ""       { Fatal("keystore is empty") }
        if keypass == ""        { Fatal("keypass is empty") }
        if storepass == ""      { Fatal("storepass is empty") }

        if GetFlagV() { Verbose("signing -> %v (%v)", targets, keystore) }
        if e := CopyFile(filepath.Join(ic.out, "unsigned.apk"), filepath.Join(ic.out, "signed.apk")); e != nil {
                return false
        }

        // signing unsigned.apk into signed.apk

        args = []string{
                "-keystore", keystore,
                "-keypass", keypass,
                "-storepass", storepass,
                filepath.Join(ic.out, "signed.apk"), "cert",
        }

        c = NewExcmd("jarsigner")
        if !c.Run("signing", args...) { os.Remove(filepath.Join(ic.out, "signed.apk")); return false }

        // zipalign signed.apk into aligned.apk then rename aligned.apk into final target

        if GetFlagV() { Verbose("aligning -> %v", targets) }

        c = NewExcmd(getBuildTool("zipalign"))
        c.SetIA32(IsIA32Command(c.GetPath()))

        args = []string{ "4", filepath.Join(ic.out, "signed.apk"), filepath.Join(ic.out, "aligned.apk"), }
        if !c.Run("aligning", args...) { os.Remove(filepath.Join(ic.out, "aligned.apk")); return false }

        if e := os.Rename(filepath.Join(ic.out, "aligned.apk"), ic.target); e != nil { Fatal("rename: %v", ic.target) }
        return true
}

func (ic *genJAR) Execute(targets []string, prerequisites []string) bool {
        // package native libs
        libname := filepath.Join(ic.out, "library.jar")
        ic.createPackage(libname) // createDummyPackage

        c := NewExcmd(getBuildTool("aapt"))
        c.SetIA32(IsIA32Command(c.GetPath()))

        args := []string{ "package", "-u",
                "-M", filepath.Join(ic.d, "AndroidManifest.xml"),
                "-I", filepath.Join(sdk, "platforms", ic.platform, "android.jar"),
        }
        if ic.res != ""    { args = append(args, "-S", ic.res) }
        if ic.assets != "" { args = append(args, "-A", ic.assets) }
        //args = append(args, "--min-sdk-version", "7")
        //args = append(args, "--target-sdk-version", "7")
        if !c.Run("resources", args...) {
                Fatal("package: %v(%v)", libname, "assets,res,...")
        }

        if GetFlagV() { Verbose("classes -> %v", targets) }

        manifest := ""
        args = []string{}
        if manifest != "" {
                args = append(args, "-ufm")
        } else {
                args = append(args, "-uf")
        }
        args = append(args, libname, "-C", filepath.Join(ic.out, "classes"), ".")
        c = NewExcmd("jar")
        if !c.Run("classes", args...) {
                Fatal("package: %v(%v)", libname, "classes")
        }

        if e := os.Rename(libname, ic.target); e != nil {
                Fatal("rename: %v", ic.target)
        }

        return true
}
