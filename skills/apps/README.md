# App Profiles

App Profile 是针对具体 Android 应用的操作知识库，供 AI agent 在操作该 App 时参考，避免从头摸索。

## 使用方式

Agent 操作某个 App 前，按包名匹配对应的 Profile 文件：

1. `adb-claw app current` 获取当前前台 App 包名
2. 在本目录查找对应 Profile
3. 有 → 读取后按 Profile 中的方法操作（优先使用深度链接）
4. 无 → 常规 `observe` + 探索

## 设备形态

同一个 App 在不同设备形态（Phone / Pad / Fold）上的布局可能不同。Profile 中通过 `## 设备差异` 章节标注差异，Agent 应结合 `adb-claw device info` 返回的屏幕尺寸判断当前设备形态。

简易判断规则：
- 短边 < 1200px → Phone
- 短边 >= 1200px → Pad / Fold

## 文件规范

- 文件名：App 常用英文名小写，如 `douyin.md`、`wechat.md`
- 每个 Profile 必须包含以下章节：

```
# {App 名称}
- 包名
- Scheme

## 深度链接
## 已知布局
## 设备差异
## 常见工作流
## 已知问题
```

## 贡献

欢迎通过 PR 提交新的 App Profile。要求：
1. 基于真机实际操作验证，注明测试的 App 版本和设备
2. 深度链接需验证可用
3. 标注 Phone / Pad 差异（如果有条件测试）
4. 禁止写 `sleep` / `time.sleep` / 固定秒数间隔。动作后立刻 `observe`，或 `wait --changed` / `wait --activity`（有画面变化才返回）
5. 禁止把 `wait --changed` 和 `observe` 写成无条件固定组合；先 `observe`，只有画面仍在切换且无法决策时才等待
6. 所有 Agent 点击必须使用 `--normalized`，不得记录或复用设备像素
7. 禁止使用 Python、curl、jq、节点 JSON、resource-id、content-desc 或已删除的文本定位器
8. 中文输入直接使用 `adb-claw type`，不得安装第三方输入法或修改 IME
