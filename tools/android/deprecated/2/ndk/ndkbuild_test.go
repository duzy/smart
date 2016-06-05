//
//  Copyright (C) 2012-2016, Duzy Chan <code@duzy.info>, all rights reserverd.
//
package smart

import (
        "testing"
        "os"
        . "github.com/duzy/smart/build"
        . "github.com/duzy/smart/test"
)

func testCleanFiles(t *testing.T) {
        //modules, moduleOrderList, moduleBuildList := GetModules(), GetModuleOrderList(), GetModuleBuildList()
        if e := os.RemoveAll("libs"); e != nil { t.Errorf("failed remove `libs' directory") }
        if e := os.RemoveAll("out"); e != nil { t.Errorf("failed remove `out' directory") }
}

func testToolsetNdkBuild(t *testing.T) {
        testCleanFiles(t)

        Build(ComputeTestRunParams())

        if fi, e := os.Stat("out"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi/objs-debug"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi/objs-debug/android_native_app_glue"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi/objs-debug/native-activity"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a/objs-debug"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a/objs-debug/android_native_app_glue"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a/objs-debug/native-activity"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a-hard"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a-hard/objs-debug"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a-hard/objs-debug/android_native_app_glue"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a-hard/objs-debug/native-activity"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }

        if fi, e := os.Stat("out/boot.mk"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi/libandroid_native_app_glue.a"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi/libnative-activity.so"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a/libandroid_native_app_glue.a"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a/libnative-activity.so"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a-hard/libandroid_native_app_glue.a"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a-hard/libnative-activity.so"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi/objs-debug/android_native_app_glue/android_native_app_glue.o"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi/objs-debug/android_native_app_glue/android_native_app_glue.o.d"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi/objs-debug/native-activity/main.o"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi/objs-debug/native-activity/main.o.d"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a/objs-debug/android_native_app_glue/android_native_app_glue.o"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a/objs-debug/android_native_app_glue/android_native_app_glue.o.d"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a/objs-debug/native-activity/main.o"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a/objs-debug/native-activity/main.o.d"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a-hard/objs-debug/android_native_app_glue/android_native_app_glue.o"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a-hard/objs-debug/android_native_app_glue/android_native_app_glue.o.d"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a-hard/objs-debug/native-activity/main.o"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("out/local/armeabi-v7a-hard/objs-debug/native-activity/main.o.d"); fi == nil || e != nil { t.Errorf("failed: %v", e); return }

        if fi, e := os.Stat("libs"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("libs/armeabi"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("libs/armeabi-v7a"); fi == nil || e != nil || !fi.IsDir() { t.Errorf("failed: %v", e); return }

        if fi, e := os.Stat("libs/armeabi/libnative-activity.so"); fi == nil || e != nil || fi.IsDir() { t.Errorf("failed: %v", e); return }
        if fi, e := os.Stat("libs/armeabi-v7a/libnative-activity.so"); fi == nil || e != nil || fi.IsDir() { t.Errorf("failed: %v", e); return }

        testCleanFiles(t)
}

func TestToolsetNdkBuild(t *testing.T) {
        RunToolsetTestCase(t, "../../..", "ndkbuild", testToolsetNdkBuild)
}
