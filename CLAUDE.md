# adb-claw

Android 设备控制 CLI，供 AI agent 自动化调用。纯工具层，不含 LLM/Agent 逻辑。

v2 起感知面只有 JPEG 帧。模型动作使用 Gemini Computer Use 的 0–999 归一化坐标。仓库中不再包含 UI dump / accessibility 文本树。

## 发布渠道

adb-claw 同时作为两个平台的 Skill 发布，**共用一份 `skills/adb-claw/SKILL.md`**：

- **Claude Code**：通过插件市场安装（`.claude-plugin/`），按 `## Triggers` 触发，`## Binary` 指示二进制位置
- **OpenClaw**：通过 ClawHub 安装，读取 YAML frontmatter 中的 `metadata.openclaw`（OS 要求、依赖、安装脚本）

两个平台读同一个文件，Claude Code 忽略 frontmatter，OpenClaw 忽略 `## Binary` 段落。

```
.claude-plugin/              # Claude Code 插件配置
├── plugin.json              # 插件元数据
└── marketplace.json         # 市场发布配置
helper/                      # 设备端 Java 辅助程序源码
├── ADBClawBridge.java       # 持久 JPEG Frame DEX
├── ADBClawAudio.java        # 系统音频采集 DEX
└── ADBClawInput.java        # Unicode ACTION_SET_TEXT DEX（不安装 APK/IME）
skills/
├── adb-claw/SKILL.md        # Skill 定义（两个平台共用）：决策循环
├── adb-claw/RUNTIME.md      # Gemini Flash / serve 适配器短规则
├── adb-claw/COMMANDS.md     # 完整 flag 与非循环命令（按需读）
└── apps/                    # App Profile 知识库（运行时按需加载）
    ├── README.md
    ├── douyin.md
    ├── xiaohongshu.md
    └── meituan.md
```

### App Profile

1. `adb-claw app current` → 获取当前 App 包名
2. 检查 `skills/apps/` 下有无对应 Profile
3. 有 → 按深度链接和视觉地标操作；无 → `observe` 看图

## 项目结构

```
src/
├── main.go
├── Makefile
├── cmd/                    # Cobra CLI
└── pkg/
    ├── adb/                # Commander 接口
    ├── frame/              # 持久 JPEG 帧源 + 容量 1 缓冲
    ├── coord/              # 0-999 ↔ 设备像素
    ├── observe/            # observe 挂 livestream；screencap 回退
    ├── stream/             # latest.jpg + sidecar
    ├── pump/               # 后台持续出帧（只留最新一帧）
    ├── server/             # serve JSONL：frame.latest / act
    ├── input/              # tap/swipe/key/type
    ├── chain/              # 同帧多步动作链
    ├── device/             # 屏幕状态
    ├── audio/              # 独立 audio CLI
    ├── output/             # JSON envelope
    └── perf/               # 分段计时
```

## 构建

```bash
cd src
make build     # 产物 → bin/adb-claw
make test
make lint
make dex       # 重新编译 Frame DEX（需要 Android SDK）
make audio-dex
make input-dex # 重新编译 Unicode 输入 DEX
```

Go 1.24，依赖 cobra v1.10.2 + golang.org/x/image v0.36.0。

## 架构要点

- **Commander 接口** — 所有 pkg 通过 `Commander` 调用 ADB
- **JSON Envelope** — `{ok, command, data, error, duration_ms, timestamp}`
- **图片-only** — observe 每帧返回唯一 JPEG 路径、hash 和 token，不返回 UI 节点
- **帧绑定坐标** — Skill 使用 `--normalized --frame TOKEN`；裸设备像素必须显式 `--raw`
- **Frame DEX** — `UiAutomation.takeScreenshot()` + 设备端 JPEG；写阻塞时丢中间帧
- **Livestream** — 后台 `pump` 只保留最新一帧；一次性 `observe` / `--wait-changed` 拷贝该帧，不重开 PNG screencap
- **自适应** — 默认原分辨率/q60，帧龄或传输 P95 超门槛按原比例降到宽 540/q50，会话内不自动升档
- **回退** — DEX 不可用时回退 `screencap`，绝不回退文本树
- **文本输入安全** — ASCII 走 `adb shell input text`；Unicode 由内嵌 DEX 通过 `ACTION_SET_TEXT` 写入焦点控件，不安装 APK、不切换 IME

## 命令树

```
adb-claw
├── device list | info
├── observe [--width px] [--max-pixels N] [--capture auto|stream|pull] [--profile]
├── screenshot [--file path] [--width px] [--max-pixels N]
├── pump [--daemon] | status | stop
├── tap <x> <y> --normalized --frame TOKEN [--wait-changed ms]
├── chain --normalized --frame TOKEN tap X Y tap X Y [--wait-changed ms]
├── long-press <x> <y> --normalized --frame TOKEN [--duration ms]
├── swipe <x1> <y1> <x2> <y2> --normalized --frame TOKEN
├── key <HOME|BACK|ENTER|...>
├── type <text>
├── clear-field
├── open <uri>
├── scroll <up|down|left|right> --frame TOKEN [--pages N]
├── wait --activity NAME | --changed --after-frame TOKEN
├── serve --stdio [--width px]
├── bench [--rounds N] [--width px]
├── screen status|on|off|unlock|rotation
├── app list|current|launch|stop|install|uninstall|clear
├── audio capture [--file path] [--duration ms] [--stream]
├── shell <command>
├── file push|pull
├── skill
└── doctor
```

## 全局 Flags

```
-s, --serial <id>
-o, --output <format>   # json | text | quiet
--timeout <ms>
--verbose
```

## 代码约定

- 新命令放 `src/cmd/`，新包放 `src/pkg/`
- 所有 ADB 调用必须通过 `Commander` 接口，不直接 exec
- 命令输出必须使用 `output.Writer` 写 JSON envelope
- 测试文件与源码同目录，用 `_test.go` 后缀
- 错误码用大写下划线格式，如 `STALE_FRAME`、`DEVICE_NOT_FOUND`
- `skills/adb-claw/SKILL.md` 同时服务 Claude Code 和 OpenClaw

## 技术方案

标准 `adb` 完成输入、截屏回退、App/屏幕管理。实时路径由 Frame DEX（`helper/ADBClawBridge.java`）通过 `app_process` 持续输出长度前缀 JPEG；后台 `pump` 只保留最新一帧，一次性 `observe` 拷贝该帧。Unicode 输入 DEX（`helper/ADBClawInput.java`）通过 accessibility `ACTION_SET_TEXT` 写入焦点控件，不安装设备应用。产品目标见 `docs/product-and-research.md`。

## 音频采集与 ASR 协作

`audio capture` 仍是独立 CLI（设备 → WAV），不做 ASR，也不进入 Skill 视觉决策。

```bash
adb-claw audio capture --stream | asrclaw transcribe --stream --lang zh
```

## 从 v1 UI API 迁移

| 旧用法 | v2 |
|--------|----|
| `observe` 返回 `ui.elements` / `state_id` | 只返回 JPEG `path` + 尺寸 |
| `tap --index/--id/--text` | `tap --normalized X Y` 或设备像素 |
| `wait --text/--id` | `wait --changed` 或 `--activity` |
| `ui tree` / `monitor` / `live cart` | 已删除；看图决策 |
| `serve observe` + `act(state_id)` | `frame.latest` + `act(frame_seq, x, y)` |

## 开发工作流

- 编码完成后运行 `cd src && make test && make build`
- 更新 SKILL.md / RUNTIME.md / COMMANDS.md / CLAUDE.md 命令树
- DEX 无法构建时必须走截图回退，不能引入文本树

## 发布流程

使用 `/adb-claw-release <版本号>`，详见 `.claude/commands/adb-claw-release.md`。

```
Git remote: origin → llm-net/adb-claw
ClawHub:    https://clawhub.ai/dionren/adb-claw
官网:       https://adb-claw.llm.net
```
