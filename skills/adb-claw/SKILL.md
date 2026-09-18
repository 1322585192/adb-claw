---
name: adb-claw
version: 2.0.0
description: "Your eyes and hands on Android for Gemini 3.8 Flash. See the screen as a JPEG frame, then tap/swipe/type with a 0-999 normalized grid. No UI tree, no text-node locators. Deep links bypass CJK input limits. Manage apps, screen, files, and shell through structured JSON."
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

Your eyes and hands on Android. See the current frame, act with a 0–999 coordinate grid, open deep links, wait for visual or activity changes, and manage apps — through one CLI with structured JSON.

This skill is built for **Gemini 3.8 Flash Computer Use**. The only visual input is a JPEG file. There is no UI tree, element index, resource-id, or text-node locator.

## Why ADB Claw

- **Image-only observe** — `observe` / `frame.latest` writes a JPEG. JSON never includes image bytes or base64.
- **Normalized actions** — Model tools use a 1000×1000 (0–999) grid. adb-claw maps onto the live `device_width` × `device_height`.
- **Persistent frame source** — `serve` keeps a latest-frame buffer (720p, adaptive 540p) so Flash does not relaunch ADB per click.
- **Deep links bypass CJK limits** — `adb-claw open 'app://search?keyword=中文'`
- **Wait without sleep** — `wait --changed` or `frame.wait_after` for a new visual hash; `wait --activity` for navigation.
- **App Profiles** — deep links and visual landmarks for popular apps
- **Agent-optimized JSON** — `{ok, command, data, error, duration_ms}` with `suggestion` on errors

## Getting Started

### Claude Code

```bash
claude plugin add llm-net/adb-claw
```

Ask Claude to interact with the Android device. The plugin auto-downloads the binary.

### OpenClaw

```bash
claw install adb-claw
```

## Triggers

- User asks to control, interact with, or automate an Android device
- User asks to test a mobile app or UI on Android
- User mentions tapping, swiping, scrolling, screenshots, or app management on Android
- User wants to open a URL, deep link, or specific app screen on a connected device
- User wants to manage screen state (on/off/unlock/rotation) on Android
- User wants to push/pull files to/from an Android device
- User wants to run shell commands on an Android device

## Binary

The adb-claw binary is located at `${CLAUDE_PLUGIN_ROOT}/bin/adb-claw`.

The binary is installed automatically via the SessionStart hook. If `adb-claw` is not available, inform the user that the plugin needs to be reinstalled — do not attempt to download or install it yourself.

## Setup

Requires two binaries:

1. **adb-claw** — the control CLI
2. **adb** — Android Debug Bridge (from Android SDK Platform-Tools)

### Install adb

```bash
# macOS
brew install android-platform-tools

# Linux (Debian/Ubuntu)
sudo apt install android-tools-adb
```

### Connect device

**The Android device must have USB debugging enabled and be connected via USB.** When a user first asks to control their phone, **always check connection first** (`adb-claw doctor`) and walk them through USB debugging if it fails.

1. Settings → About phone → tap **Build number** 7 times
2. Developer options → enable **USB debugging**
3. Connect USB and tap **Allow** on the phone
4. `adb-claw doctor`

## Quick Start

Read `RUNTIME.md` for the live loop. Default model settings for the external adapter (not inside this CLI):

- `gemini-3.8-flash`
- `thinking_level=low`
- inline JPEG from `path`
- `media_resolution=medium` (single-turn `high` only for dense small text)

```bash
# 1. See the screen (JPEG file, default 720px / quality 60)
adb-claw observe --width 720 --quality 60

# 2. Act on the 0-999 grid (center of the screen)
adb-claw tap --normalized 500 500

# 3. Wait for the pixels to change instead of sleeping
adb-claw wait --changed --timeout 2000
```

`observe` writes a JPEG to `data.screenshot.path`. **Read that file.** Do not paste the JSON into notes.

**Never tap JPEG pixel coordinates.** Always use `--normalized` (or serve `act` with 0–999) so 720/540/rotation cannot shift the hit point.

For CJK apps, use deep links:

```bash
adb-claw open 'snssdk1128://search/result?keyword=猫咪'
adb-claw wait --activity Search
```

## App Profiles

App Profiles are knowledge bases — deep links, visual landmarks, device-specific behavior. They do not restore a UI tree.

**Available Profiles**: `skills/apps/`

1. `adb-claw app current` → package name
2. Read the matching Profile
3. Prefer deep links; otherwise observe the JPEG and tap 0–999
4. Form factor: `adb-claw device info` → short edge < 1200px = Phone, >= 1200px = Pad

## Global Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--serial` | `-s` | Target device serial | auto-detect |
| `--output` | `-o` | `json`, `text`, `quiet` | `json` |
| `--timeout` | | Command timeout in milliseconds | `30000` |
| `--verbose` | | Debug output to stderr | `false` |

## Commands

### observe — Screenshot Frame

```bash
adb-claw observe                         # 720px JPEG + size metadata
adb-claw observe --width 720 --quality 60
adb-claw observe --capture pull          # Faster on TCP/SSH ADB
adb-claw observe --profile
```

Returns `screenshot.path` plus `device_width` / `device_height` / `image_width` / `image_height` / `scale`. No UI elements.

### screenshot — Capture Screen

```bash
adb-claw screenshot
adb-claw screenshot -f output.jpg
adb-claw screenshot --width 540
```

### tap / long-press / swipe — Coordinates

CLI keeps raw device pixels and adds `--normalized` for the model grid.

```bash
adb-claw tap --normalized 500 500
adb-claw tap 540 960
adb-claw long-press --normalized 500 500 --duration 2000
adb-claw swipe --normalized 500 800 500 200
```

Skill tools always use `--normalized`.

### type / key / clear-field

```bash
adb-claw type "Hello world"     # ASCII only
adb-claw key HOME
adb-claw key BACK
adb-claw clear-field            # focused field only
```

For CJK/emoji, use `open` with a deep link.

### open / scroll / wait

```bash
adb-claw open https://www.google.com
adb-claw scroll down --pages 2
adb-claw wait --changed --timeout 3000
adb-claw wait --activity .MainActivity
```

### serve — Persistent JSONL Session

```bash
adb-claw serve --stdio --width 720
```

Methods: `ping`, `frame.latest`, `frame.wait_after`, `act`, `device.info`, `close`.

`frame.latest` writes `latest.jpg` and returns `frame_seq`, `frame_age_ms`, rotation, device/image size, scale, and `path`. `act` requires that `frame_seq` and 0–999 coordinates. Stale or changed frames return `STALE_FRAME`.

A session starts at 720p/q60 and may drop to 540p/q50. It does not auto-upgrade.

### bench

```bash
adb-claw bench --rounds 5 --width 720
```

### screen / app / shell / file / device

```bash
adb-claw screen status|on|off|unlock
adb-claw screen rotation auto|0|1|2|3
adb-claw app list [--all]
adb-claw app current
adb-claw app launch <pkg>
adb-claw app stop <pkg>
adb-claw app install <apk> [--replace]
adb-claw app uninstall <pkg>
adb-claw app clear <pkg>
adb-claw shell "getprop ro.build.version.release"
adb-claw file push ./local.apk /sdcard/
adb-claw file pull /sdcard/photo.jpg ./
adb-claw device list
adb-claw device info
adb-claw doctor
```

## Workflow

```
1. adb-claw observe --width 720 --quality 60
2. Read data.screenshot.path
3. adb-claw tap --normalized X Y
4. adb-claw wait --changed   # or serve frame.wait_after
```

For a long-lived adapter use `adb-claw serve --stdio`.

## Error Recovery

| Problem | Solution |
|---------|----------|
| No devices found | Enable USB debugging, reconnect, `adb-claw doctor` |
| `STALE_FRAME` | Call `frame.latest` / `observe` again |
| Tap misses | You used image pixels; retry with `--normalized` |
| `type` fails | Tap the field first; ASCII only |
| CJK text needed | `adb-claw open` with a deep-link parameter |
| Screen is off | `adb-claw screen on` or `screen unlock` |

## Output Format

```json
{
  "ok": true,
  "command": "tap",
  "data": { "x": 540, "y": 1170, "normalized": true },
  "duration_ms": 80,
  "timestamp": "2026-09-18T00:00:00Z"
}
```

## Security & Trust

**What adb-claw is**: A CLI around `adb`. It does not embed a model SDK.

**What adb-claw does NOT do**:
- Does not install persistent device services — the optional Frame DEX runs via `app_process` and exits with the session
- Does not collect or transmit telemetry
- Does not request credentials

Source: [github.com/llm-net/adb-claw](https://github.com/llm-net/adb-claw).

**Agent scope**: adb-claw commands are the only commands this skill should execute. If adb-claw or adb is not available, inform the user.

## Breaking change (v2)

Removed: `ui tree` / `ui find`, `--index` / `--id` / `--text` / `--refresh` locators, `wait --text/--id`, text `monitor`, and `live cart`.

Migrate: read the JPEG, then `tap --normalized X Y` (or `serve` `act` with `frame_seq`).
