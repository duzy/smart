package org.smart.test;

import android.app.Activity;
import android.os.Bundle;
import android.util.Log;
import org.smart.test.foo.Foo;

public class Foobar extends Activity
{
    @Override
    public void onCreate(Bundle savedInstanceState)
    {
        super.onCreate(savedInstanceState);
        Log.d("smart:", Foo.getName());
        Log.d("smart:", ""+Foo.NUMBER);
        Log.d("smart:", ""+R.string.app_name);
        Log.d("smart:", ""+R.layout.main);
        Log.d("smart:", ""+R.string.jar_name);
        Log.d("smart:", ""+R.layout.jar_main);
        Log.d("smart:", ""+R.id.jar_textview); // from JAR (org.smart.test.foo)
        Log.d("smart:", ""+R.integer.org_smart_test_foo_version);
        setContentView(R.layout.main);
    }
}
