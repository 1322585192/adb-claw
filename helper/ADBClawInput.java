import android.content.ClipData;
import android.content.ClipboardManager;
import android.content.Context;
import android.os.Looper;
import android.util.Base64;

import java.lang.reflect.Method;
import java.nio.charset.StandardCharsets;

/**
 * One-shot Unicode input helper run by app_process as the shell user.
 *
 * It creates a com.android.shell Context, writes UTF-8 text to the clipboard,
 * and exits. The host follows this with KEYCODE_PASTE. No APK or IME is
 * installed on the device.
 *
 * Usage:
 *   CLASSPATH=/data/local/tmp/adbclaw-input.dex app_process / \
 *     ADBClawInput --base64 <base64-utf8>
 */
public final class ADBClawInput {
    private ADBClawInput() {}

    public static void main(String[] args) {
        try {
            String encoded = parseBase64(args);
            String text = new String(
                Base64.decode(encoded, Base64.NO_WRAP),
                StandardCharsets.UTF_8
            );

            if (Looper.myLooper() == null) {
                Looper.prepareMainLooper();
            }

            Context systemContext = systemContext();
            Context shellContext = systemContext.createPackageContext(
                "com.android.shell",
                Context.CONTEXT_IGNORE_SECURITY
            );
            ClipboardManager clipboard =
                (ClipboardManager) shellContext.getSystemService(Context.CLIPBOARD_SERVICE);
            if (clipboard == null) {
                throw new IllegalStateException("clipboard service unavailable");
            }
            clipboard.setPrimaryClip(ClipData.newPlainText("", text));
            System.out.println("OK");
        } catch (Throwable t) {
            System.err.println("ADBClawInput: " + rootMessage(t));
            System.exit(1);
        }
    }

    private static String parseBase64(String[] args) {
        if (args.length == 2 && "--base64".equals(args[0]) && !args[1].isEmpty()) {
            return args[1];
        }
        throw new IllegalArgumentException("usage: ADBClawInput --base64 <base64-utf8>");
    }

    private static Context systemContext() throws Exception {
        Class<?> activityThread = Class.forName("android.app.ActivityThread");
        Method systemMain = activityThread.getDeclaredMethod("systemMain");
        systemMain.setAccessible(true);
        Object thread = systemMain.invoke(null);
        Method getSystemContext = activityThread.getDeclaredMethod("getSystemContext");
        getSystemContext.setAccessible(true);
        return (Context) getSystemContext.invoke(thread);
    }

    private static String rootMessage(Throwable t) {
        Throwable current = t;
        while (current.getCause() != null) {
            current = current.getCause();
        }
        String message = current.getMessage();
        return current.getClass().getSimpleName() +
            (message == null || message.isEmpty() ? "" : ": " + message);
    }
}
