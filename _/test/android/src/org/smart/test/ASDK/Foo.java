package org.smart.test.ASDK;

import android.app.Activity;
import android.os.Bundle;

public class Foo extends Activity
{
    /** Called when the activity is first created. */
    @Override
    public void onCreate(Bundle savedInstanceState)
    {
        super.onCreate(savedInstanceState);

        setContentView(R.layout.main);

        test();
    }

    static { System.loadLibrary("foo"); }
    private native void test();
}
