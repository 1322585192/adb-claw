# ADB Claw command reference

Agent decision rules live in [SKILL.md](SKILL.md). This page is flags, extras, and diagnostics. Do not start a visual loop from this page.

## Global flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--serial` | `-s` | Target device serial | auto-detect |
| `--output` | `-o` | `json`, `text`, `quiet` | `json` |
| `--timeout` | | Command timeout in milliseconds | `30000` |
| `--verbose` | | Debug output to stderr | `false` |

If `device list` shows more than one device, pass `-s SERIAL` on every command.

## observe

Copies the newest livestream JPEG to a unique path. The first call starts a background pump; later calls copy `latest.jpg`. Do not sleep to “wait for a new screenshot”.

```bash
adb-claw observe
adb-claw observe --width 540
adb-claw observe --max-pixels 650000
```

| Flag | Default | Agent note |
|------|---------|------------|
| `--width` | `0` (native) | Optional uniform downscale cap |
| `--max-pixels` | `0` (native) | Rotation-invariant pixel budget |
| `--format` | `jpeg` | Keep jpeg |
| `--quality` | `60` | Already the default; do not repeat it |
| `--file` | unique temp path | Do not hardcode |
| `--capture` | `auto` | Do not switch to `pull` to debug a reused path |
| `--profile` | off | Diagnostics only |

Returns `data.screenshot` with `path`, `frame_token`, `hash`, `captured_at`, `rotation`, `action_width` / `action_height`, image size, and scale. No UI elements.

## Frame-bound input

Agents always pass `--normalized` and `--frame TOKEN`. Bare coordinates are rejected. `--raw` is a human debugging escape — never use it in this skill.

```bash
adb-claw tap --normalized X Y --frame TOKEN [--wait-changed 1500]
adb-claw long-press --normalized X Y --frame TOKEN [--duration 2000] [--wait-changed 1500]
adb-claw swipe --normalized X1 Y1 X2 Y2 --frame TOKEN [--duration 300] [--wait-changed 1500]
adb-claw scroll {up|down|left|right} --frame TOKEN [--pages N] [--wait-changed 1500]
```

`--wait-changed` requires a bound frame. After it returns, the next token is `data.screenshot.frame_token`, not top-level `data.frame_token`.

`scroll --distance` is an optional pixel override. Prefer `--pages` with `--frame`.

## type / key / clear-field

```bash
adb-claw type "Hello world"
adb-claw type "王者荣耀"
adb-claw clear-field
adb-claw key KEY
```

`type` needs a focused editable field. Unicode uses embedded `ACTION_SET_TEXT`. No APK, no IME change.

Key aliases: `HOME`, `BACK`, `ENTER`, `TAB`, `DEL` / `DELETE`, `POWER`, `VOLUME_UP`, `VOLUME_DOWN`, `MENU`, `SEARCH`, `DPAD_*`, `APP_SWITCH` / `RECENTS`, `CAMERA`, `SPACE`, `ESCAPE`, `PASTE`, `COPY`, `CUT`, `FORWARD_DEL`, `MOVE_HOME`, `MOVE_END`, `PAGE_UP`, `PAGE_DOWN`, `WAKEUP`, `SLEEP`. Unknown names are sent as `KEYCODE_<NAME>`.

## open / wait

```bash
adb-claw open https://www.google.com
adb-claw open 'snssdk1128://search/result?keyword=猫咪'
adb-claw wait --changed --after-frame TOKEN [--timeout 3000]
adb-claw wait --activity .MainActivity
adb-claw wait --activity .Splash --gone
```

`--timeout` is a deadline. Never run `wait` without `--changed` or `--activity`. `--gone` applies only to `--activity`. `--changed` requires `--after-frame`.

## screen / app / device

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
adb-claw device list
adb-claw device info
adb-claw doctor
```

## shell / file

Allowed when the user asked for that task. `shell` rejects UI hierarchy dumps, layout/accessibility `dumpsys`, raw clipboard binder calls, and IME changes.

```bash
adb-claw shell "getprop ro.build.version.release"
adb-claw file push ./local.apk /sdcard/
adb-claw file pull /sdcard/photo.jpg ./
```

## Out of the visual loop

### screenshot

Saves a picture for a human. Do not use it to choose taps — it is not the observe token loop.

```bash
adb-claw screenshot
adb-claw screenshot -f output.jpg
adb-claw screenshot --width 540
```

### audio

Independent WAV capture (Android 11+, REMOTE_SUBMIX). Speakers mute while capturing. Not perception for taps.

```bash
adb-claw audio capture --file recording.wav
adb-claw audio capture --duration 30000 --stream
```

### pump

Latest-frame livestream. First `observe` starts it. Use these only when doctor says the stream is down.

```bash
adb-claw pump --daemon
adb-claw pump status
adb-claw pump stop
```

### serve / bench

`serve --stdio` is for Gemini adapters ([RUNTIME.md](RUNTIME.md)): `ping`, `frame.latest`, `frame.wait_after`, `act`, `device.info`, `close`. `act` needs `frame_seq` and 0–999. Stale seq → `STALE_FRAME`.

```bash
adb-claw serve --stdio
adb-claw bench --rounds 5
```

`adb-claw skill` prints the machine-readable `skill.json` catalog. Do not treat it as a second workflow.
