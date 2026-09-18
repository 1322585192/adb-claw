# ADB Claw Runtime Rules

Use this page during a Gemini 3.8 Flash control loop. Perception is a JPEG file only.

## Fast loop

1. `adb-claw observe --quality 60` — reads the newest livestream frame (a background pump stays up; do not recapture or sleep)
2. Read its unique `screenshot.path` and keep `frame_token`. Do not paste JSON or reuse an older path.
3. Act with that token and 0–999 coordinates: `adb-claw tap --normalized X Y --frame FRAME_TOKEN --wait-changed 1500`.
4. Read the returned `screenshot.path` directly. It has a new token for the next action; never add `sleep`.
5. If `STALE_FRAME`, observe again. Rotation invalidates the previous token.

## Defaults for Gemini 3.8 Flash

- Model: `gemini-3.8-flash`
- `thinking_level=low`
- Inline JPEG from the file at `path`
- `media_resolution=medium` (upgrade to `high` only for a dense small-text page)
- Coordinates: Gemini Computer Use 0–999 grid, never preview-image pixels
- Capture on a USB device: `auto` selects `stream`; the pump keeps the latest JPEG. Do not switch to pull to troubleshoot a stale fixed path

## Do not do this

- Insert any delay between commands: `sleep`, `time.sleep`, `adb-claw shell sleep`, `adb shell sleep`, “wait 1 second”, “pause briefly”
- Ask for a UI tree, element index, resource-id, or on-screen text node
- Tap using JPEG pixel coordinates
- Reuse a fixed path such as `adb-claw-observe.jpg`; every frame path is unique
- Put image bytes or base64 into the tool JSON
- Call `wait` with only `--timeout` to fake a sleep
- Run `uiautomator`, layout/accessibility `dumpsys`, raw clipboard service calls, or IME-changing shell commands
- Download/install ADBKeyboard or another helper APK; `adb-claw type` already handles Unicode
- Use host `curl`, `find`, Python, or another tool to build a device-control workaround

After a frame-bound action, read its returned changed frame. If the action ran without `--wait-changed`, use `wait --changed --after-frame FRAME_TOKEN`; that wait returns the JPEG path directly.

## Wait decision

- One-shot CLI: `observe` → action with `--frame`; use `--wait-changed` when the action should visibly change the screen.
- Serve: `act` → `frame.latest`, or `frame.wait_after` when change is required. Read the returned path directly.
- Empty results and error/placeholder pages are real outcomes. Do not wait or tap repeatedly to make them disappear.
- On the same visual hash, do not tap the same point more than twice. Never switch to raw pixels.

## Text input

Focus the field with a frame-bound tap, then call `adb-claw type TEXT`. ASCII uses `adb shell input text`; Unicode uses the embedded app_process `ACTION_SET_TEXT` helper. It installs no APK and does not change the active IME. On failure, keep focus and retry once.

## Persistent session

```text
adb-claw serve --stdio
```

Methods: `frame.latest` → `act {frame_seq, action, x, y}` → `frame.wait_after`.
Frames keep the device's native aspect (quality 60) and may drop to a 540px-wide uniform scale if they are stale.
