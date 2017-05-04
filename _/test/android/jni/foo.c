#include <jni.h>
#include <errno.h>
#include <android/log.h>

#define LOGI(...) ((void)__android_log_print(ANDROID_LOG_INFO, "foo", __VA_ARGS__))
#define LOGW(...) ((void)__android_log_print(ANDROID_LOG_WARN, "foo", __VA_ARGS__))

JNIEXPORT void Java_org_smart_test_ASDK_Foo_test(JNIEnv *Env, jobject Obj)
{
  LOGI("Java_org_smart_test_ASDK_test");
}
