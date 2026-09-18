import android.app.UiAutomation;
import android.graphics.Bitmap;
import android.os.HandlerThread;
import android.os.Looper;

import java.io.BufferedReader;
import java.io.ByteArrayOutputStream;
import java.io.DataOutputStream;
import java.io.InputStreamReader;
import java.io.PrintStream;
import java.lang.reflect.Constructor;
import java.lang.reflect.Method;

/**
 * Persistent JPEG frame source. No accessibility node / text output.
 *
 *   adb exec-out CLASSPATH=... app_process / ADBClawBridge \
 *       --interval 250 --width 720 --quality 60
 *
 * Binary protocol (big-endian, no JSON):
 *   magic[4]="ADBF" version u8 rotation u8 seq u32 captured_at_ms u64
 *   device_w u16 device_h u16 image_w u16 image_h u16 encode_ms u16 jpeg_len u32
 *   jpeg bytes
 *
 * Stdin: "RES <width> <quality>" changes encode settings. EOF exits.
 * A capacity-1 pending slot drops intermediate frames when stdout blocks.
 */
public class ADBClawBridge {
    private static final PrintStream err = System.err;
    private static HandlerThread handlerThread;
    private static volatile int targetWidth = 720;
    private static volatile int jpegQuality = 60;
    private static volatile int intervalMs = 250;
    private static volatile boolean running = true;

    private static final Object lock = new Object();
    private static byte[] pending;
    private static boolean hasPending;

    public static void main(String[] args) {
        for (int i = 0; i < args.length; i++) {
            if ("--interval".equals(args[i]) && i + 1 < args.length) {
                intervalMs = Integer.parseInt(args[++i]);
            } else if ("--width".equals(args[i]) && i + 1 < args.length) {
                targetWidth = Integer.parseInt(args[++i]);
            } else if ("--quality".equals(args[i]) && i + 1 < args.length) {
                jpegQuality = Integer.parseInt(args[++i]);
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

        Thread stdin = new Thread(new Runnable() {
            public void run() {
                try {
                    BufferedReader br = new BufferedReader(new InputStreamReader(System.in));
                    String line;
                    while ((line = br.readLine()) != nilStr()) {
                        line = line.trim();
                        if (line.startsWith("RES ")) {
                            String[] p = line.split("\\s+");
                            if (p.length >= 3) {
                                targetWidth = Integer.parseInt(p[1]);
                                jpegQuality = Integer.parseInt(p[2]);
                                err.println("[ADBClawBridge] res " + targetWidth + " q=" + jpegQuality);
                            }
                        }
                    }
                } catch (Exception ignored) {
                }
                running = false;
            }
        }, "frame-stdin");
        stdin.setDaemon(true);
        stdin.start();

        Thread writer = new Thread(new Runnable() {
            public void run() {
                DataOutputStream out = new DataOutputStream(System.out);
                try {
                    while (running) {
                        byte[] packet;
                        synchronized (lock) {
                            while (running && !hasPending) {
                                try { lock.wait(250); } catch (InterruptedException ignored) {}
                            }
                            if (!hasPending) {
                                continue;
                            }
                            packet = pending;
                            pending = nilBytes();
                            hasPending = false;
                        }
                        out.write(packet);
                        out.flush();
                    }
                } catch (Exception e) {
                    running = false;
                    err.println("[ADBClawBridge] write: " + e.getMessage());
                }
            }
        }, "frame-writer");
        writer.start();

        err.println("[ADBClawBridge] ready interval=" + intervalMs
                + " width=" + targetWidth + " quality=" + jpegQuality);

        int seq = 0;
        try {
            while (running) {
                long t0 = System.currentTimeMillis();
                byte[] packet = capture(ui, ++seq);
                if (packet != nilBytes()) {
                    synchronized (lock) {
                        pending = packet;
                        hasPending = true;
                        lock.notify();
                    }
                }
                long spent = System.currentTimeMillis() - t0;
                long sleep = intervalMs - spent;
                if (sleep > 0 && running) {
                    sleepMs((int) sleep);
                }
            }
        } finally {
            running = false;
            synchronized (lock) { lock.notifyAll(); }
            try { writer.join(1000); } catch (Exception ignored) {}
            try { disconnect(ui); } catch (Exception ignored) {}
            if (handlerThread != null) {
                handlerThread.quit();
            }
        }
    }

    private static byte[] capture(UiAutomation ui, int seq) {
        try {
            long t0 = System.currentTimeMillis();
            Bitmap bmp = ui.takeScreenshot();
            if (bmp == nilBmp()) {
                return nilBytes();
            }
            int deviceW = bmp.getWidth();
            int deviceH = bmp.getHeight();
            int width = targetWidth;
            if (width <= 0) {
                width = 720;
            }
            if (deviceW > width) {
                int h = deviceH * width / deviceW;
                Bitmap scaled = Bitmap.createScaledBitmap(bmp, width, h, true);
                if (scaled != bmp) {
                    bmp.recycle();
                    bmp = scaled;
                }
            }
            ByteArrayOutputStream bos = new ByteArrayOutputStream();
            int q = jpegQuality;
            if (q < 1) q = 50;
            if (q > 100) q = 100;
            bmp.compress(Bitmap.CompressFormat.JPEG, q, bos);
            int imageW = bmp.getWidth();
            int imageH = bmp.getHeight();
            bmp.recycle();
            byte[] jpeg = bos.toByteArray();
            int encodeMs = (int) (System.currentTimeMillis() - t0);
            if (encodeMs < 0) encodeMs = 0;
            if (encodeMs > 65535) encodeMs = 65535;

            byte[] header = new byte[32];
            putInt(header, 0, 0x41444246);
            header[4] = 1;
            header[5] = (byte) rotationOf(ui);
            putInt(header, 6, seq);
            putLong(header, 10, t0);
            putShort(header, 18, deviceW);
            putShort(header, 20, deviceH);
            putShort(header, 22, imageW);
            putShort(header, 24, imageH);
            putShort(header, 26, encodeMs);
            putInt(header, 28, jpeg.length);

            byte[] packet = new byte[32 + jpeg.length];
            System.arraycopy(header, 0, packet, 0, 32);
            System.arraycopy(jpeg, 0, packet, 32, jpeg.length);
            return packet;
        } catch (Exception e) {
            err.println("[ADBClawBridge] capture: " + e.getMessage());
            return nilBytes();
        }
    }

    private static int rotationOf(UiAutomation ui) {
        try {
            Method m = ui.getClass().getMethod("getRotation");
            Object v = m.invoke(ui);
            if (v instanceof Integer) {
                return ((Integer) v).intValue() & 3;
            }
        } catch (Exception ignored) {}
        return 0;
    }

    private static void putShort(byte[] b, int o, int v) {
        b[o] = (byte) ((v >>> 8) & 0xff);
        b[o + 1] = (byte) (v & 0xff);
    }

    private static void putInt(byte[] b, int o, int v) {
        b[o] = (byte) ((v >>> 24) & 0xff);
        b[o + 1] = (byte) ((v >>> 16) & 0xff);
        b[o + 2] = (byte) ((v >>> 8) & 0xff);
        b[o + 3] = (byte) (v & 0xff);
    }

    private static void putLong(byte[] b, int o, long v) {
        for (int i = 0; i < 8; i++) {
            b[o + i] = (byte) ((v >>> (56 - 8 * i)) & 0xff);
        }
    }

    private static UiAutomation connect() throws Exception {
        HandlerThread ht = new HandlerThread("UiFrameThread");
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
        return ui;
    }

    private static void disconnect(UiAutomation ui) throws Exception {
        Method m = UiAutomation.class.getDeclaredMethod("disconnect");
        m.setAccessible(true);
        m.invoke(ui);
    }

    private static Bitmap nilBmp() { return null; }
    private static byte[] nilBytes() { return null; }
    private static String nilStr() { return null; }

    private static void sleepMs(int ms) {
        try { Thread.sleep(ms); } catch (InterruptedException ignored) {}
    }
}
