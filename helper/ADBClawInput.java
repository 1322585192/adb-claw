import android.app.UiAutomation;
import android.os.Bundle;
import android.os.HandlerThread;
import android.os.Looper;
import android.util.Base64;
import android.view.accessibility.AccessibilityNodeInfo;

import java.lang.reflect.Constructor;
import java.lang.reflect.Method;
import java.nio.charset.StandardCharsets;

/**
 * One-shot Unicode input helper run by app_process as the shell user.
 *
 * It writes text into the focused editable field through UiAutomation
 * ACTION_SET_TEXT. That keeps focus in the current app and avoids installing
 * an APK or changing the IME.
 *
 * Usage:
 *   CLASSPATH=/data/local/tmp/adbclaw-input.dex app_process / \
 *     ADBClawInput --base64 <base64-utf8>
 */
public final class ADBClawInput {
    private static HandlerThread handlerThread;

    private ADBClawInput() {}

    public static void main(String[] args) {
        UiAutomation ui = null;
        try {
            String text = new String(
                Base64.decode(parseBase64(args), Base64.NO_WRAP),
                StandardCharsets.UTF_8
            );
            ui = connect();
            if (!setTextOnFocused(ui, text)) {
                throw new IllegalStateException("no focused editable field");
            }
            System.out.println("OK SET_TEXT");
        } catch (Throwable t) {
            System.err.println("ADBClawInput: " + rootMessage(t));
            System.exit(1);
        } finally {
            if (ui != null) {
                try { disconnect(ui); } catch (Exception ignored) {}
            }
            if (handlerThread != null) {
                handlerThread.quit();
            }
        }
    }

    private static String parseBase64(String[] args) {
        if (args.length == 2 && "--base64".equals(args[0]) && !args[1].isEmpty()) {
            return args[1];
        }
        throw new IllegalArgumentException("usage: ADBClawInput --base64 <base64-utf8>");
    }

    private static boolean setTextOnFocused(UiAutomation ui, String text) {
        AccessibilityNodeInfo root = ui.getRootInActiveWindow();
        if (root == null) {
            return false;
        }
        AccessibilityNodeInfo focused = root.findFocus(AccessibilityNodeInfo.FOCUS_INPUT);
        if (focused == null) {
            focused = findEditable(root);
        }
        if (focused == null || !focused.isEditable()) {
            return false;
        }
        Bundle extras = new Bundle();
        extras.putCharSequence(
            AccessibilityNodeInfo.ACTION_ARGUMENT_SET_TEXT_CHARSEQUENCE,
            text
        );
        return focused.performAction(AccessibilityNodeInfo.ACTION_SET_TEXT, extras);
    }

    private static AccessibilityNodeInfo findEditable(AccessibilityNodeInfo node) {
        if (node == null) {
            return null;
        }
        if (node.isFocused() && node.isEditable()) {
            return node;
        }
        for (int i = 0; i < node.getChildCount(); i++) {
            AccessibilityNodeInfo child = node.getChild(i);
            AccessibilityNodeInfo found = findEditable(child);
            if (found != null) {
                return found;
            }
        }
        return null;
    }

    private static UiAutomation connect() throws Exception {
        HandlerThread ht = new HandlerThread("UiInputThread");
        ht.start();
        handlerThread = ht;
        Class<?> connClass = Class.forName("android.app.UiAutomationConnection");
        Object conn = connClass.getDeclaredConstructor().newInstance();
        Class<?> iConnClass = Class.forName("android.app.IUiAutomationConnection");
        Constructor<UiAutomation> ctor =
            UiAutomation.class.getDeclaredConstructor(Looper.class, iConnClass);
        ctor.setAccessible(true);
        UiAutomation ui = ctor.newInstance(ht.getLooper(), conn);
        Method connectMethod = UiAutomation.class.getDeclaredMethod("connect");
        connectMethod.setAccessible(true);
        connectMethod.invoke(ui);
        return ui;
    }

    private static void disconnect(UiAutomation ui) throws Exception {
        Method m = UiAutomation.class.getDeclaredMethod("disconnect");
        m.setAccessible(true);
        m.invoke(ui);
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
