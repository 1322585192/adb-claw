# ADB Claw Runtime Rules

Use this page during a Gemini 3.8 Flash control loop. Perception is a JPEG file only.

## Fast loop

1. `frame.latest` (or `adb-claw observe --width 720 --quality 60`)
2. Read `path`. Do not paste JSON. Do not look for UI text nodes.
3. `act` with that `frame_seq` and 0–999 coordinates (`adb-claw tap --normalized X Y`)
4. Next command immediately — no `sleep`. Use `frame.wait_after` only if you must wait for the pixels to change. If the next step is already `frame.latest` / `observe`, skip the wait.
5. If `STALE_FRAME`, get a new frame before acting again

## Defaults for Gemini 3.8 Flash

- Model: `gemini-3.8-flash`
- `thinking_level=low`
- Inline JPEG from the file at `path`
- `media_resolution=medium` (upgrade to `high` only for a dense small-text page)
- Coordinates: Gemini Computer Use 0–999 grid, never preview-image pixels

## Do not do this

- Insert any delay between commands: `sleep`, `time.sleep`, `adb-claw shell sleep`, `adb shell sleep`, “wait 1 second”, “pause briefly”
- Ask for a UI tree, element index, resource-id, or on-screen text node
- Tap using JPEG pixel coordinates
- Put image bytes or base64 into the tool JSON
- Call `wait` with only `--timeout` to fake a sleep

After an action, the next command is `observe` / `frame.latest`, or `wait --changed` / `frame.wait_after` / `wait --activity`. Those waits return early. Do not add a sleep before or after them.

## Persistent session

```text
adb-claw serve --stdio --width 720
```

Methods: `frame.latest` → `act {frame_seq, action, x, y}` → `frame.wait_after`.
Frames start at 720p/q60 and may drop to 540p/q50 for the rest of the session.
