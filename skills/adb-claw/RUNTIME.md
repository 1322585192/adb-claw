# ADB Claw Runtime Rules

Use this page during a Gemini 3.8 Flash control loop. Perception is a JPEG file only.

## Fast loop

1. `frame.latest` (or `adb-claw observe --width 720 --quality 60`)
2. Read `path`. Do not paste JSON. Do not look for UI text nodes.
3. `act` with that `frame_seq` and 0–999 coordinates (`adb-claw tap --normalized X Y`)
4. `frame.wait_after` until the image hash changes (or timeout returns the latest frame)
5. If `STALE_FRAME`, get a new frame before acting again

## Defaults for Gemini 3.8 Flash

- Model: `gemini-3.8-flash`
- `thinking_level=low`
- Inline JPEG from the file at `path`
- `media_resolution=medium` (upgrade to `high` only for a dense small-text page)
- Coordinates: Gemini Computer Use 0–999 grid, never preview-image pixels

## Do not do this

- Ask for a UI tree, element index, resource-id, or on-screen text node
- Sleep a fixed number of milliseconds after a tap
- Tap using JPEG pixel coordinates
- Put image bytes or base64 into the tool JSON

## Persistent session

```text
adb-claw serve --stdio --width 720
```

Methods: `frame.latest` → `act {frame_seq, action, x, y}` → `frame.wait_after`.
Frames start at 720p/q60 and may drop to 540p/q50 for the rest of the session.
