//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        //"path/filepath"
        "strings"
        "testing"
        //"fmt"
        "os"
        . "github.com/duzy/smart/build"
        . "github.com/duzy/smart/test"
)

func testCleanFiles(t *testing.T) {
        if e := os.RemoveAll("out"); e != nil { t.Errorf("failed remove `out' directory") }
        if e := os.RemoveAll("obj"); e != nil { t.Errorf("failed remove `obj' directory") }
}

func testToolsetAndroidSDK(t *testing.T) {
        testCleanFiles(t)

        Build(ComputeTestRunParams())

        if fi, e := os.Stat("out"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/res"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/res/org/smart/test/foo"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/res/org/smart/test/foo/R.java"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/classes"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/classes/org/smart/test/foo"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/classes/org/smart/test/foo/Foo.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/classes/org/smart/test/foo/R.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/classes/org/smart/test/foo/R$attr.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/classes/org/smart/test/foo/R$id.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/classes/org/smart/test/foo/R$layout.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/classes/org/smart/test/foo/R$string.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_jar/foo_androidsdk_jar.jar"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/res"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/res/org/smart/test"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/res/org/smart/test/R.java"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/classes"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/classes/org/smart/test"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/classes/org/smart/test/Foobar.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/classes/org/smart/test/R.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/classes/org/smart/test/R$attr.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/classes/org/smart/test/R$layout.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/classes/org/smart/test/R$string.class"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/classes.dex"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/unsigned.apk"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/signed.apk"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_use_jar/foo_androidsdk_use_jar.apk"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_apk"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_apk/res"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_apk/classes"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_apk/classes.dex"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_apk/unsigned.apk"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_apk/signed.apk"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_androidsdk_apk/foo_androidsdk_apk.apk"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }

        l1 := Runcmd("jar", "tf", "out/foo_androidsdk_jar/foo_androidsdk_jar.jar")
        l2 := Runcmd("jar", "tf", "out/foo_androidsdk_use_jar/foo_androidsdk_use_jar.apk")
        l3 := Runcmd("jar", "tf", "out/foo_androidsdk_apk/foo_androidsdk_apk.apk")

        if s := "org/smart/test/foo/Foo.class\n"; !strings.Contains(l1, s)      { t.Errorf("missing %v----------\n%v", s, l1); return }
        if s := "org/smart/test/foo/R.class\n"; !strings.Contains(l1, s)        { t.Errorf("missing %v----------\n%v", s, l1); return }
        if s := "org/smart/test/foo/R$attr.class\n"; !strings.Contains(l1, s)   { t.Errorf("missing %v----------\n%v", s, l1); return }
        if s := "org/smart/test/foo/R$id.class\n"; !strings.Contains(l1, s)     { t.Errorf("missing %v----------\n%v", s, l1); return }
        if s := "org/smart/test/foo/R$layout.class\n"; !strings.Contains(l1, s) { t.Errorf("missing %v----------\n%v", s, l1); return }
        if s := "org/smart/test/foo/R$string.class\n"; !strings.Contains(l1, s) { t.Errorf("missing %v----------\n%v", s, l1); return }
        if s := "META-INF/MANIFEST.MF\n"; !strings.Contains(l2, s)              { t.Errorf("missing %v----------\n%v", s, l2); return }
        if s := "META-INF/CERT.SF\n"; !strings.Contains(l2, s)                  { t.Errorf("missing %v----------\n%v", s, l2); return }
        if s := "META-INF/CERT.RSA\n"; !strings.Contains(l2, s)                 { t.Errorf("missing %v----------\n%v", s, l2); return }
        if s := "res/layout/main.xml\n"; !strings.Contains(l2, s)               { t.Errorf("missing %v----------\n%v", s, l2); return }
        if s := "resources.arsc\n"; !strings.Contains(l2, s)                    { t.Errorf("missing %v----------\n%v", s, l2); return }
        if s := "classes.dex\n"; !strings.Contains(l2, s)                       { t.Errorf("missing %v----------\n%v", s, l2); return }
        if s := "AndroidManifest.xml\n"; !strings.Contains(l2, s)               { t.Errorf("missing %v----------\n%v", s, l2); return }
        if s := "META-INF/MANIFEST.MF\n"; !strings.Contains(l3, s)              { t.Errorf("missing %v----------\n%v", s, l3); return }
        if s := "META-INF/CERT.SF\n"; !strings.Contains(l3, s)                  { t.Errorf("missing %v----------\n%v", s, l3); return }
        if s := "META-INF/CERT.RSA\n"; !strings.Contains(l3, s)                 { t.Errorf("missing %v----------\n%v", s, l3); return }
        if s := "res/layout/main.xml\n"; !strings.Contains(l3, s)               { t.Errorf("missing %v----------\n%v", s, l3); return }
        if s := "resources.arsc\n"; !strings.Contains(l3, s)                    { t.Errorf("missing %v----------\n%v", s, l3); return }
        if s := "classes.dex\n"; !strings.Contains(l3, s)                       { t.Errorf("missing %v----------\n%v", s, l3); return }
        if s := "AndroidManifest.xml\n"; !strings.Contains(l3, s)               { t.Errorf("missing %v----------\n%v", s, l3); return }

        testCleanFiles(t)
}

func testToolsetAndroidSDKJNI(t *testing.T) {
        testCleanFiles(t)

        Build(ComputeTestRunParams())

        if fi, e := os.Stat("out"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi/objs-debug"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi/objs-debug/foo"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a/objs-debug"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a/objs-debug/foo"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a-hard"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a-hard/objs-debug"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a-hard/objs-debug/foo"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }

        if fi, e := os.Stat("out/boot.mk"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi/libfoo.so"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a/libfoo.so"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a-hard/libfoo.so"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi/objs-debug/foo/foo.o"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi/objs-debug/foo/foo.o.d"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a/objs-debug/foo/foo.o"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a/objs-debug/foo/foo.o.d"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a-hard/objs-debug/foo/foo.o"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a-hard/objs-debug/foo/foo.o.d"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }

        if fi, e := os.Stat("libs"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("libs/armeabi"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("libs/armeabi-v7a"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }

        if fi, e := os.Stat("libs/armeabi/libfoo.so"); fi == nil || e != nil || fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("libs/armeabi-v7a/libfoo.so"); fi == nil || e != nil || fi.IsDir() { t.Errorf("failed: %v", e); return }

        if fi, e := os.Stat("out"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_apk"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_apk/res"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_apk/classes"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_apk/classes.dex"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_apk/unsigned.apk"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_apk/signed.apk"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/foo_apk/foo_apk.apk"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }

        l := Runcmd("jar", "tf", "out/foo_apk/foo_apk.apk")

        if s := "lib/armeabi/libfoo.so\n"; !strings.Contains(l, s) { t.Errorf("missing %v----------\n%v", s, l); return }
        if s := "lib/armeabi-v7a/libfoo.so\n"; !strings.Contains(l, s) { t.Errorf("missing %v----------\n%v", s, l); return }

        testCleanFiles(t)
}

func TestToolsetAndroidSDK(t *testing.T) {
        RunToolsetTestCase(t, "../../..", "android-sdk", testToolsetAndroidSDK)
        RunToolsetTestCase(t, "../../..", "android", testToolsetAndroidSDKJNI)
}
