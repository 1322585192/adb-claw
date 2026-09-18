# 小红书 (Xiaohongshu / RedNote)

- 包名: `com.xingin.xhs`
- Scheme: `xhsdiscover://`

## 深度链接

优先使用深度链接，避免手动经过多层搜索页面。URI 可直接包含中文：

```bash
adb-claw open 'xhsdiscover://search/result?keyword=保健品推荐&type=51'
adb-claw open 'xhsdiscover://search'
adb-claw app launch com.xingin.xhs
```

| 动作 | 链接 | 参数说明 |
|------|------|----------|
| 搜索内容 | `xhsdiscover://search/result?keyword={keyword}&type=51` | `type=51` 经测试可进入综合结果页 |
| 打开搜索页 | `xhsdiscover://search` | 只打开搜索页，不提交搜索 |

若深度链接失效，可聚焦搜索框后使用内置 Unicode 输入：

```bash
adb-claw tap --normalized X Y
adb-claw clear-field
adb-claw type "保健品推荐"
adb-claw key ENTER
```

不要安装输入法、修改 IME 或用外部脚本编码关键词。

## 已知布局

### 搜索结果页

```text
┌─────────────────────────────────────────────┐
│ [返回]  搜索关键词                 [X] [搜索] │
├─────────────────────────────────────────────┤
│ 全部 │ 用户 │ 商品 │ 图片 │ 地点 │ 问一问     │
├─────────────────────────────────────────────┤
│ 综合 │ 可购买 │ 最新 │ 场景筛选              │
├─────────────────────────────────────────────┤
│ [封面] 标题 / 作者 / 日期 / 点赞             │
│ [封面] 标题 / 作者 / 日期 / 点赞             │
└─────────────────────────────────────────────┘
```

看 JPEG 识别顶部搜索栏、分类 Tab 和双列内容卡片。不要请求节点树、文本定位器或 bounds。

### 用户搜索结果页

顶部「用户」Tab 下是头像、账号名、粉丝数和关注按钮组成的纵向列表。先 `observe`，再用 `--normalized` 点击目标账号所在卡片。

### 用户主页

```text
┌─────────────────────────────────────────────┐
│ [返回] 头图                         [更多]    │
├─────────────────────────────────────────────┤
│ [头像] 账号名称 / 认证 / 小红书号 / 简介       │
│ 关注数 │ 粉丝数 │ 获赞与收藏数       [关注]    │
├─────────────────────────────────────────────┤
│ 动态栏 / 橱窗预览                            │
├─────────────────────────────────────────────┤
│ 笔记 │ 选品 │ 收藏                           │
│ 双列帖子瀑布流                               │
└─────────────────────────────────────────────┘
```

账号数据、认证标签、帖子标题和点赞数都从 JPEG 读取。页面布局会随账号内容变化，禁止复用旧设备像素坐标。

### 帖子详情

- 图文笔记：图片、标题、正文、标签、评论区纵向排列。
- 视频笔记：通常为全屏播放器；整页纵向滚动会切换到下一视频。
- 需要评论时，先看图定位评论区域，再在区域内用归一化端点小幅 `swipe`。

### 橱窗页

顶部是橱窗搜索和分类，下面是排序栏与商品卡片。商品名可能插入零宽字符，模型应直接按画面理解，不做字符串精确匹配或节点解析。

## 设备差异

### Phone（短边 < 1200px）

- 竖屏为主。
- 搜索结果和主页帖子通常是双列瀑布流。

### Pad / Fold（短边 >= 1200px）

- 可能显示三列或更宽布局。
- 必须根据最新 JPEG 重新选取 0–999 坐标。

## 常见工作流

### 搜索内容

```bash
adb-claw open 'xhsdiscover://search/result?keyword=保健品推荐&type=51'
adb-claw observe
```

如果首帧仍是旧页面且明显处于切换中：

```bash
adb-claw wait --changed --timeout 5000
# 画面已经变化，再读取一次新 JPEG
adb-claw observe
```

不要无条件执行 `wait + observe`；先看首帧再决定是否需要等变化。

### 搜索用户并进入主页

```bash
adb-claw open 'xhsdiscover://search/result?keyword=注册营养师Yuanyuan&type=51'
adb-claw observe
# 看图点击「用户」Tab
adb-claw tap --normalized X Y
adb-claw observe
# 看图点击目标账号卡片
adb-claw tap --normalized X Y
adb-claw observe
```

### 读取主页与帖子

```bash
adb-claw observe
adb-claw scroll down
adb-claw observe
```

每次只根据当前 JPEG 读取可见内容。若页面未变化，同一坐标最多再试一次，不循环点击。

### 查看橱窗商品

```bash
adb-claw observe
# 看图点击橱窗入口
adb-claw tap --normalized X Y
adb-claw observe
adb-claw scroll down
adb-claw observe
```

### 查看帖子正文或评论

```bash
adb-claw observe
# 看图点击目标帖子卡片
adb-claw tap --normalized X Y
adb-claw observe
```

图文帖可继续 `scroll down`；视频帖不要整页滚动，先确认评论入口或评论区域。

## 已知问题

### 深度链接后首帧还是旧页面

页面切换和渲染可能晚于命令返回。先立即 `observe`；只有看到旧页或加载态且无法继续决策时，再 `wait --changed` 后 `observe`。禁止固定 sleep。

### 「全部」Tab 与清空按钮容易误触

搜索框聚焦时清空按钮靠近分类区域。根据 JPEG 选择 Tab 中心，不使用文本定位器；点击后立即看新图确认。

### 用户 Tab 搜索范围较窄

「用户」通常按账号名匹配。找 KOL 时可先在综合结果中识别作者，再搜索具体账号名。

### 视频帖滚动会切换视频

视频播放器中的全屏纵向手势会切换内容。需要评论时使用画面中的评论入口，或只在评论区域小幅滑动。

### 搜索或内容为空

空结果、错误提示和占位页都是有效状态。不要重复点同一搜索按钮或等待到 timeout；更换关键词、返回上一页或向用户报告。

## 推荐能力

| 能力 | 方法 |
|------|------|
| 搜索内容/用户 | 深度链接；失效时聚焦后 `type` Unicode |
| 读取主页数据 | `observe` 看 JPEG |
| 浏览帖子 | `scroll` → `observe` |
| 打开目标 | `tap --normalized` → `observe` |
| 读取橱窗 | 看图点击、滚动、继续看图 |

---

> 测试设备: Xiaomi M2007J1SC，Android 13，1080×2120
> 测试 App 版本: 2026 年 3 月版
