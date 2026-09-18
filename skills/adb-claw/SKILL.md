---
name: adb-claw
version: 2.0.0
description: "Controls a connected Android device via adb-claw: observe a JPEG frame, then tap, swipe, type, open deep links, and manage apps on a 0-999 grid. Use when the user asks to control, test, screenshot, tap, swipe, scroll, type, unlock, record audio, or automate an Android phone or tablet. Image-only — no UI tree or text-node locators. Built-in Unicode input needs no APK or IME change."
homepage: https://github.com/llm-net/adb-claw
metadata:
  {
    "openclaw":
      {
        "emoji": "📱",
        "version": "2.0.0",
        "os": ["darwin", "linux"],
        "tags": ["android", "adb", "mobile", "automation", "device-control", "deep-link", "screenshot", "computer-use"],
        "requires": { "bins": ["adb-claw", "adb"] },
        "install":
          [
            {
              "id": "adb-claw-darwin-arm64",
              "kind": "download",
              "url": "https://github.com/llm-net/adb-claw/releases/latest/download/adb-claw-darwin-arm64",
              "bins": ["adb-claw"],
              "os": "darwin",
              "label": "Download adb-claw (macOS Apple Silicon)",
            },
            {
              "id": "adb-claw-darwin-amd64",
              "kind": "download",
              "url": "https://github.com/llm-net/adb-claw/releases/latest/download/adb-claw-darwin-amd64",
              "bins": ["adb-claw"],
              "os": "darwin",
              "label": "Download adb-claw (macOS Intel)",
            },
            {
              "id": "adb-claw-linux-amd64",
              "kind": "download",
              "url": "https://github.com/llm-net/adb-claw/releases/latest/download/adb-claw-linux-amd64",
              "bins": ["adb-claw"],
              "os": "linux",
              "label": "Download adb-claw (Linux x86_64)",
            },
            {
              "id": "adb-claw-linux-arm64",
              "kind": "download",
              "url": "https://github.com/llm-net/adb-claw/releases/latest/download/adb-claw-linux-arm64",
              "bins": ["adb-claw"],
              "os": "linux",
              "label": "Download adb-claw (Linux ARM64)",
            },
            {
              "id": "adb-brew",
              "kind": "brew",
              "formula": "android-platform-tools",
              "bins": ["adb"],
              "label": "Install ADB (brew)",
            },
          ],
      },
  }
---

# ADB Claw — Android Device Control

Eyes and hands on a connected Android device. Perception is one JPEG file. Actions use a 0–999 grid bound to that frame's `frame_token`. There is no UI tree, element index, resource-id, or text-node locator.

Claude Code and OpenClaw use the one-shot CLI below. Gemini Flash adapters use [RUNTIME.md](RUNTIME.md). Full flags and diagnostics are in [COMMANDS.md](COMMANDS.md).

## Triggers

- User asks to control, interact with, or automate an Android device
- User asks to test a mobile app or UI on Android
- User mentions tapping, swiping, scrolling, screenshots, or app management on Android
- User wants to open a URL, deep link, or specific app screen on a connected device
- User wants to manage screen state (on/off/unlock/rotation) on Android
- User wants to push/pull files to/from an Android device
- User wants to run shell commands on an Android device
- User wants to record or hear system audio from an Android device

## Binary

The adb-claw binary is located at `${CLAUDE_PLUGIN_ROOT}/bin/adb-claw`.

The binary is installed automatically via the SessionStart hook. If `adb-claw` is not available, tell the user the plugin needs to be reinstalled — do not download or install it yourself.

Packaged OpenClaw binaries are macOS/Linux. On Windows, use an `adb-claw` already on `PATH` (local build). Do not invent an install URL.

## Hard rules

1. **Read the JPEG before every coordinate action.** Parse JSON only to get `data.screenshot.path` and the token. Open that image. Decide from pixels, not from memory.
2. **Never sleep.** No `sleep`, `time.sleep`, `adb-claw shell sleep`, `adb shell sleep`, “wait 1 second”, or a `wait --timeout` with neither `--changed` nor `--activity`.
3. **Never tap JPEG pixels or device pixels.** Always `--normalized --frame TOKEN`. Never `--raw`.
4. **Never reuse an old path or token.** After `--wait-changed`, the next action uses `data.screenshot.path` and `data.screenshot.frame_token`. Top-level `data.frame_token` is the frame you just acted on — do not reuse it.
5. **Device work uses adb-claw only.** No host `curl` / `find` / Python control loops. No `uiautomator`, layout/accessibility `dumpsys`, clipboard binder calls, IME changes, or helper APKs (including ADBKeyboard).
6. **Stay on the one-shot CLI.** Do not start `serve`, `bench`, `screenshot`, `--capture pull`, or `pump start/stop` unless the user asked for that diagnostic, or [COMMANDS.md](COMMANDS.md) says to.

```bash
# WRONG
adb-claw tap --normalized 500 500 --frame FRAME_TOKEN
sleep 2
adb-claw observe

# RIGHT — act and read the returned JPEG
adb-claw tap --normalized 500 500 --frame FRAME_TOKEN --wait-changed 1500
```

`long-press --duration` is the press itself, not a pause between commands.

## First session

1. `adb-claw doctor` on the first request (or whenever a command says no device).
2. USB debugging must be on. Walk the user through it if doctor fails: Settings → About phone → tap **Build number** 7 times → Developer options → **USB debugging** → connect USB → tap **Allow** → `adb-claw doctor` again. A device already listed by `adb-claw device list` is usable even over wireless ADB; this skill has no `device connect` helper.
3. If more than one device is listed, pass `-s SERIAL` on every command.
4. `adb-claw observe` then open `data.screenshot.path`.
5. `adb-claw app current` → if a Profile exists, read it before tapping.

Requires `adb` on `PATH` (Android SDK platform-tools). Quality defaults to 60; do not add `--quality 60` every time.

## Default loop

```text
observe → open JPEG → (app current / Profile / open) →
tap|swipe|scroll --normalized --frame TOKEN --wait-changed →
open the returned JPEG → repeat
```

| What just happened | Next step |
|--------------------|-----------|
| `observe` returned | Open `data.screenshot.path`. Keep `data.screenshot.frame_token`. |
| `tap` / `swipe` / `scroll` / `long-press` with `--wait-changed` | Open `data.screenshot.path`. Next token is `data.screenshot.frame_token`. |
| `type` / `key` / `clear-field` | `observe` immediately. Use the new token. |
| `open` / `app launch` | `observe` immediately. Wait only if that frame is still a loading splash you cannot act on. |
| Action ran without `--wait-changed` | `wait --changed --after-frame TOKEN` and open its `data.screenshot.path`. |
| Goal screen is visible | Stop. Report what you see. |
| Login, captcha, payment, or a permission dialog the user must answer | Stop and ask the user. Do not guess passwords or tap through consent. |

`--timeout` is a deadline, not a sleep. If the next step is already `observe`, skip `wait`.

## Coordinates

The grid is 1000×1000 on **this JPEG** (`action_width` × `action_height`), not preview pixels.

- `(0,0)` is top-left. `(999,999)` is bottom-right.
- Tap the **center of the target control**, not `500 500` unless that control is actually in the middle.
- Stay off the status bar, gesture bar, and system nav keys unless that is the target.
- If a sheet, dialog, or keyboard covers the old target, tap what is visible now.
- Lists: tap a card, or `scroll` to reveal more. Do not swipe unless you need a custom path.
- Video/live feeds change hash every frame. `changed: true` is not proof the tap worked — judge the new JPEG.

## `--wait-changed`

Use `--wait-changed 1500` when the action should change the pixels you care about (navigation, opening a sheet). Skip it for taps that only focus a field before `type`.

On return, always open `data.screenshot.path` (present even when `changed` is false):

- `changed: true` → decide from the new picture. On a video feed this may be unrelated motion.
- `changed: false` → the latest frame is the current state. Act, stop, or `wait --changed --after-frame NEW_TOKEN` only if it is still mid-transition and you cannot decide. Do not raise the timeout to fake a sleep.

## App Profiles

Profiles are deep links and visual landmarks — not a UI tree. They live at `../apps/` relative to this file (`skills/apps/` from the repo root):

| File | App | Package |
|------|-----|---------|
| `douyin.md` | 抖音 | `com.ss.android.ugc.aweme` |
| `xiaohongshu.md` | 小红书 | `com.xingin.xhs` |
| `meituan.md` | 美团 | `com.sankuai.meituan` |

1. `adb-claw app current`
2. Read the matching file if it exists
3. Prefer `adb-claw open 'scheme://...'` over typing into a search box
4. Form factor: `adb-claw device info` → short edge < 1200px = Phone, >= 1200px = Pad

No Profile → observe and tap.

## Command index

```bash
adb-claw observe
adb-claw tap --normalized X Y --frame TOKEN --wait-changed 1500
adb-claw long-press --normalized X Y --frame TOKEN --duration 2000
adb-claw swipe --normalized X1 Y1 X2 Y2 --frame TOKEN --wait-changed 1500
adb-claw scroll down --pages 2 --frame TOKEN --wait-changed 1500
adb-claw type "中文或 ASCII"
adb-claw clear-field
adb-claw key HOME|BACK|ENTER|DEL|APP_SWITCH|PASTE
adb-claw open 'snssdk1128://search/result?keyword=猫咪'
adb-claw wait --changed --after-frame TOKEN --timeout 3000
adb-claw wait --activity .MainActivity
adb-claw app current|list|launch <pkg>|stop <pkg>
adb-claw screen status|on|off|unlock
adb-claw device list|info
adb-claw doctor
```

`type` needs a focused editable field. Unicode uses the built-in helper — no APK, no IME change. Prefer a deep link when it avoids UI steps.

`shell` / `file` / `app install` are allowed for the user's stated task. `shell` still rejects UI dumps, layout/accessibility `dumpsys`, clipboard binder calls, and IME changes.

**Not in the visual loop**

- `screenshot` — save a file for a human. Do not use it to decide taps.
- `audio capture` — WAV only (Android 11+). Use when the user asks to record or hear device audio; never as perception for taps.
- `pump` — first `observe` starts it. Do not start/stop it unless doctor says the stream is down.
- `serve --stdio` — Gemini adapters only. See [RUNTIME.md](RUNTIME.md).
- `bench` / `observe --profile` / `--capture pull` — diagnostics, not the loop.

## Retry and stop

- Empty results, error pages, placeholders, and unchanged screens are observed states, not proof the tap missed.
- If the target is still visible, adjust once from the latest JPEG and its new token. Do not tap the same `hash + normalized point` more than twice.
- Two frames in a row with no progress toward the goal → switch strategy (`open`, another app entry, or ask the user). Do not keep tapping the same region.
- `STALE_FRAME` → `observe` again. The token is invalid when that frame's action width/height no longer match the live screen (true rotate). A landscape JPEG is still valid when its action size matches the live screen.
- Missed tap → re-read the latest JPEG and estimate a new 0–999 point. Do not switch to `--raw` or image-pixel coordinates.
- `type` fails → keep focus, retry once, then observe. Do not install an IME.
- No device / screen off → `doctor`, then `screen on` or `screen unlock`.
- Image-only path cannot proceed → report the visible state or ask the user.

## Returned JSON

Every command wraps `{ok, command, data, error, duration_ms, timestamp}`. Errors may include `suggestion`. After a wait-bound action, ignore top-level `data.frame_token` and use the nested screenshot:

```json
{
  "ok": true,
  "command": "tap",
  "data": {
    "x": 540,
    "y": 1170,
    "normalized": true,
    "frame_token": "TOKEN_YOU_JUST_USED",
    "changed": true,
    "next": "read_path_directly",
    "screenshot": {
      "path": "/tmp/adb-claw-<unique>.jpg",
      "frame_token": "TOKEN_FOR_NEXT_ACTION",
      "hash": "…",
      "action_width": 1080,
      "action_height": 2400
    }
  }
}
```

`observe` uses the same nested shape: `data.screenshot.path` and `data.screenshot.frame_token`. JSON never includes image bytes or base64.

## Agent scope

adb-claw commands are the only device commands this skill should run. If `adb-claw` or `adb` is missing, tell the user. The CLI does not embed a model, collect telemetry, or install a persistent device service — the optional Frame DEX exits with the session.
