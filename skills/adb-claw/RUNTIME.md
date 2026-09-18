# ADB Claw Runtime Rules

Use this page during a live control loop. Load the main Skill only for setup, troubleshooting, or rare commands.

## Fast loop

1. `adb-claw observe --width 540 --quality 50 --ui-mode realtime`
2. Read `data.screenshot.path`. Use `data.ui.elements[].center` or `--index` from this snapshot.
3. `adb-claw tap <center.x> <center.y>` or `adb-claw tap --index N`
4. Re-observe only after navigation or a `STALE_STATE` error.

Do not paste observe JSON into notes. Do not tap screenshot-image pixels. `bounds` / `center` are device pixels.

## Do not do this

- `tap --index` without a recent observe — it will fail with `STALE_STATE` instead of dumping again
- `tap --refresh` on every click
- observe after every tap when the page did not change
- `wait --text` plus extra observe polling for the same condition

## Persistent session

For Flash/Live adapters:

```text
adb-claw serve --stdio --ui-mode realtime --capture auto --width 540
```

Methods: `observe` → `act {state_id, action, index|handle|x,y}`. A mismatched `state_id` returns `STALE_STATE`.

## Capture modes

- `auto`: `pull` on TCP/SSH ADB (`host:port`), `stream` on USB
- `pull`: device PNG + `adb pull` (faster on remote tunnels)
- `stream`: `adb exec-out screencap -p`
