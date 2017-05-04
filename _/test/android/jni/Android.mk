LOCAL_PATH:= $(call my-dir)

include $(CLEAR_VARS)

LOCAL_MODULE := foo
LOCAL_SRC_FILES := foo.c
LOCAL_LDLIBS    := -llog -landroid -lEGL -lGLESv1_CM

include $(BUILD_SHARED_LIBRARY)
