# ADB Claw Runtime — Gemini adapter

Use this page only for a long-lived **Gemini 3.8 Flash** adapter that speaks `adb-claw serve --stdio`. Claude Code and OpenClaw stay on the one-shot CLI in [SKILL.md](SKILL.md).

Perception is a JPEG file. Do not put image bytes or base64 in tool JSON.

## Adapter defaults

- Model: `gemini-3.8-flash`
- `thinking_level=low`
- Inline JPEG from the file at `path`
- `media_resolution=medium` (one turn at `high` only for dense small text)
- Coordinates: Gemini Computer Use 0–999 grid, never preview-image pixels
- USB capture: `auto` selects `stream`. The pump keeps the latest JPEG. Do not switch to `pull` because a reused path looks stale — every frame path is unique.

## Serve loop

```text
adb-claw serve --stdio
```

Methods: `ping`, `frame.latest`, `frame.wait_after`, `act`, `device.info`, `close`.

1. `frame.latest` — writes a unique JPEG; JSON has `path`, `frame_seq`, `frame_age_ms`, rotation, sizes, scale.
2. Read that JPEG. Act with `act {frame_seq, action, x, y}` on the 0–999 grid.
3. When the next picture is required, `frame.wait_after` and read its `path` directly. Do not call `frame.latest` just to duplicate it.
4. `STALE_FRAME` → `frame.latest` again. The seq is invalid when that frame's action width/height no longer match the live screen (true rotate). A landscape JPEG is still valid when its action size matches the live screen.

Frames start at the device's native aspect (quality 60) and may drop to a 540px-wide uniform scale if they are stale. The session does not auto-upgrade.

One-shot equivalent (no serve): `observe` → `tap` or `chain --normalized --frame TOKEN --wait-changed 1500` → read `data.screenshot.path`. Serve still sends one `act` per request this release. Same stop/retry rules as [SKILL.md](SKILL.md).

## Do not do this

- Insert any delay: `sleep`, `time.sleep`, `adb-claw shell sleep`, `adb shell sleep`, “wait 1 second”, “pause briefly”
- Ask for a UI tree, element index, resource-id, or on-screen text node
- Tap JPEG pixel coordinates or reuse a fixed path such as `adb-claw-observe.jpg`
- Call `wait` with only `--timeout` to fake a sleep
- Run `uiautomator`, layout/accessibility `dumpsys`, raw clipboard service calls, or IME-changing shell
- Download/install ADBKeyboard or another helper APK; `adb-claw type` handles Unicode
- Use host `curl`, `find`, Python, or another tool as a device-control workaround
- Treat `changed: true` on a video/live feed as proof a tap worked — judge the new JPEG

## Wait and retry

- Serve: `act` then `frame.wait_after` when a new picture is required.
- One-shot: `--wait-changed` on the action, or `wait --changed --after-frame TOKEN`.
- Empty results and error/placeholder pages are real outcomes.
- Same visual hash: do not tap the same point more than twice. Never switch to raw pixels.
- Login, captcha, payment, or a permission dialog the user must answer → stop and ask.

## Text input

Focus the field with a frame-bound tap, then `adb-claw type TEXT`. ASCII uses `adb shell input text`; Unicode uses the embedded `ACTION_SET_TEXT` helper. No APK, no IME change. On failure, keep focus and retry once.
