import android.accessibilityservice.AccessibilityServiceInfo;
import android.app.UiAutomation;
import android.graphics.Bitmap;
import android.os.HandlerThread;
import android.os.Looper;
import android.view.accessibility.AccessibilityEvent;
import android.view.accessibility.AccessibilityNodeInfo;

import java.io.ByteArrayOutputStream;
import java.io.PrintStream;
import java.lang.reflect.Constructor;
import java.lang.reflect.Method;

/**
 * Event-driven UI + optional JPEG frame helper.
 * Runs via app_process; exits when stdin closes or the host kills the process.
 *
 *   adb shell CLASSPATH=... app_process / ADBClawBridge [--debounce 80] [--frames 0]
 */
public class ADBClawBridge {
    private static final PrintStream out = System.out;
    private static final PrintStream err = System.err;
    private static HandlerThread handlerThread;
    private static volatile long lastEventMs;
    private static volatile boolean dirty = true;

    public static void main(String[] args) {
        int debounceMs = 80;
        int frameMs = 0;
        for (int i = 0; i < args.length; i++) {
            if ("--debounce".equals(args[i]) && i + 1 < args.length) {
                debounceMs = Integer.parseInt(args[++i]);
            } else if ("--frames".equals(args[i]) && i + 1 < args.length) {
                frameMs = Integer.parseInt(args[++i]);
            }
        }

        UiAutomation ui;
        try {
            ui = connect();
        } catch (Exception e) {
            err.println("[ADBClawBridge] connect failed: " + e.getMessage());
            System.exit(1);
            return;
        }

        try {
            ui.setOnAccessibilityEventListener(new UiAutomation.OnAccessibilityEventListener() {
                public void onAccessibilityEvent(AccessibilityEvent event) {
                    lastEventMs = System.currentTimeMillis();
                    dirty = true;
                }
            });
        } catch (Exception e) {
            err.println("[ADBClawBridge] listener failed: " + e.getMessage());
        }

        err.println("[ADBClawBridge] ready debounce=" + debounceMs + " frames=" + frameMs);
        long lastFrame = 0;
        try {
            while (true) {
                long now = System.currentTimeMillis();
                if (dirty && now - lastEventMs >= debounceMs) {
                    dirty = false;
                    emitTree(ui);
                }
                if (frameMs > 0 && now - lastFrame >= frameMs) {
                    lastFrame = now;
                    emitFrame(ui);
                }
                sleep(20);
            }
        } finally {
            try { disconnect(ui); } catch (Exception ignored) {}
            if (handlerThread != null) {
                handlerThread.quit();
            }
        }
    }

    private static void emitTree(UiAutomation ui) {
        AccessibilityNodeInfo root = ui.getRootInActiveWindow();
        if (root == nil()) {
            return;
        }
        StringBuilder sb = new StringBuilder(256);
        sb.append("{\"type\":\"ui\",\"nodes\":[");
        walk(root, sb, true);
        sb.append("]}");
        out.println(sb.toString());
        out.flush();
        root.recycle();
    }

    private static boolean walk(AccessibilityNodeInfo node, StringBuilder sb, boolean first) {
        if (node == null) {
            return first;
        }
        CharSequence text = node.getText();
        CharSequence desc = node.getContentDescription();
        String rid = node.getViewIdResourceName();
        boolean clickable = node.isClickable();
        boolean scrollable = node.isScrollable();
        boolean keep = (text != null && text.length() > 0)
            || (desc != null && desc.length() > 0)
            || clickable || scrollable
            || (rid != null && node.getChildCount() == 0);
        if (keep) {
            if (!first) {
                sb.append(',');
            }
            first = false;
            sb.append("{\"text\":").append(js(text == null ? "" : text.toString()));
            sb.append(",\"content_desc\":").append(js(desc == null ? "" : desc.toString()));
            sb.append(",\"resource_id\":").append(js(rid == null ? "" : rid));
            sb.append(",\"clickable\":").append(clickable);
            sb.append(",\"scrollable\":").append(scrollable);
            sb.append(",\"enabled\":").append(node.isEnabled());
            android.graphics.Rect r = new android.graphics.Rect();
            node.getBoundsInScreen(r);
            sb.append(",\"bounds\":[").append(r.left).append(',').append(r.top)
              .append(',').append(r.right).append(',').append(r.bottom).append("]");
            sb.append('}');
        }
        int n = node.getChildCount();
        for (int i = 0; i < n; i++) {
            AccessibilityNodeInfo child = node.getChild(i);
            if (child != null) {
                first = walk(child, sb, first);
                child.recycle();
            }
        }
        return first;
    }

    private static void emitFrame(UiAutomation ui) {
        try {
            Bitmap bmp = ui.takeScreenshot();
            if (bmp == nilBmp()) {
                return;
            }
            int w = bmp.getWidth();
            if (w > 540) {
                int h = bmp.getHeight() * 540 / w;
                Bitmap scaled = Bitmap.createScaledBitmap(bmp, 540, h, true);
                if (scaled != bmp) {
                    bmp.recycle();
                    bmp = scaled;
                }
            }
            ByteArrayOutputStream bos = new ByteArrayOutputStream();
            bmp.compress(Bitmap.CompressFormat.JPEG, 50, bos);
            bmp.recycle();
            byte[] jpeg = bos.toByteArray();
            out.println("{\"type\":\"frame\",\"bytes\":" + jpeg.length + "}");
            out.flush();
            out.write(int32(jpeg.length));
            out.write(jpeg);
            out.flush();
        } catch (Exception e) {
            err.println("[ADBClawBridge] frame: " + e.getMessage());
        }
    }

    private static byte[] int32(int n) {
        return new byte[]{
            (byte) (n >>> 24), (byte) (n >>> 16), (byte) (n >>> 8), (byte) n
        };
    }

    private static UiAutomation connect() throws Exception {
        HandlerThread ht = new HandlerThread("UiBridgeThread");
        ht.start();
        handlerThread = ht;
        Class<?> connClass = Class.forName("android.app.UiAutomationConnection");
        Object conn = connClass.getDeclaredConstructor().newInstance();
        Class<?> iConnClass = Class.forName("android.app.IUiAutomationConnection");
        Constructor<UiAutomation> ctor = UiAutomation.class.getDeclaredConstructor(Looper.class, iConnClass);
        ctor.setAccessible(true);
        UiAutomation ui = ctor.newInstance(ht.getLooper(), conn);
        Method connectMethod = UiAutomation.class.getDeclaredMethod("connect");
        connectMethod.setAccessible(true);
        connectMethod.invoke(ui);
        try {
            AccessibilityServiceInfo info = ui.getServiceInfo();
            if (info != null) {
                info.flags |= AccessibilityServiceInfo.FLAG_RETRIEVE_INTERACTIVE_WINDOWS;
                ui.setServiceInfo(info);
            }
        } catch (Exception ignored) {}
        return ui;
    }

    private static void disconnect(UiAutomation ui) throws Exception {
        Method m = UiAutomation.class.getDeclaredMethod("disconnect");
        m.setAccessible(true);
        m.invoke(ui);
    }

    private static AccessibilityNodeInfo nil() { return null; }
    private static Bitmap nilBmp() { return null; }

    private static String js(String s) {
        return "\"" + s.replace("\\", "\\\\").replace("\"", "\\\"").replace("\n", "\\n") + "\"";
    }

    private static void sleep(int ms) {
        try { Thread.sleep(ms); } catch (InterruptedException ignored) {}
    }
}
