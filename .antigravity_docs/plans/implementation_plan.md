# NewsPocket 借鉴 GitHub 类似项目升级实施计划

基于对 GitHub 上同类热门开源项目（如 YeeKal/ai-daily、Thysrael/Horizon、JackyST0/hotpush、climpea/daily-news-digest 等）的深入调研与对比，本计划旨在将 NewsPocket 从「单一邮件抓取工具」全面演进为「轻量、全渠道、防疲劳、抗噪音的现代化自动化早报系统」。

---

## 一、 对标调研与核心借鉴点

| 模块 | GitHub 类似项目优势 | NewsPocket 现有现状 | 本次升级目标 |
| :--- | :--- | :--- | :--- |
| **推送渠道** | 支持 Webhook、飞书、企业微信、钉钉、Telegram | 仅支持 SMTP 邮件 | 打造零依赖的 `internal/pusher` 多通道分发系统 |
| **链接清洁** | 剥离各种追踪参数 (`utm_*`, `spm`, 各种来源统计) | 保留原始脏 URL，易影响去重 | 引入 `CleanURL` 净化器，杜绝因参数导致的去重失效 |
| **跨日去重** | 具备历史记录缓存（History/Anti-Fatigue） | 仅在单次运行内存去重，长热搜会连推几天 | 引入轻量级 `internal/history` 跨日持久化去重 |
| **输出格式** | 支持 Markdown、HTML、JSON 多模态输出 | 仅渲染 HTML 模板 | 增加极佳排版的 Markdown 渲染器，支持 `--markdown` 导出 |
| **运维感知** | 抓取健康度看板（响应时间、条目、成功率） | 仅有失败 warning 日志 | 统计每个源的耗时与条目，提供终端友好看板与状态汇总 |
| **源预设集** | 提供开箱即用的分类源集（AI、科技、热搜、金融） | 用户需自行编写 JSON | 增设 `config/presets/` 精选订阅预设模板库 |

---

## 二、 详细设计与步骤

### 阶段 1：URL 深度清洗器（净化与强化去重）
- 文件：`internal/parser/cleaner.go` 与 `internal/parser/cleaner_test.go`
- 功能：剥离 `utm_source`, `utm_medium`, `utm_campaign`, `utm_term`, `utm_content`, `spm_id_from`, `from`, `source`, `feature` 等常见跟踪参数。
- 保证 URL 规范化，消除同一条新闻在不同源或不同分享入口下的哈希差异。

### 阶段 2：Markdown 优雅渲染器
- 文件：`internal/renderer/markdown.go` 与 `internal/renderer/markdown_test.go`
- 功能：输出标准 GitHub Flavored Markdown 晨报，包含：
  - 今日日期与总览卡片
  - 🤖 AI 今日核心要闻 1 分钟速读
  - 分类板块与带时间、来源、评分的高亮条目
  - 数据源概览

### 阶段 3：多渠道 Webhook 推送系统 (`internal/pusher`)
- 文件：`internal/pusher/pusher.go` 与 `internal/pusher/pusher_test.go`
- 支持推送目标：
  1. **飞书 (Feishu / Lark)**：富文本卡片/Markdown 消息
  2. **企业微信 (WeCom)**：Markdown 消息
  3. **钉钉 (DingTalk)**：Markdown 消息
  4. **Telegram Bot**：支持 HTML 格式发送长图文
  5. **通用 Webhook**：标准 JSON Payload
- 零第三方依赖：原生基于 Go `net/http`。
- 支持环境变量一键配置：`FEISHU_WEBHOOK`, `WECOM_WEBHOOK`, `DINGTALK_WEBHOOK`, `TELEGRAM_BOT_TOKEN`, `TELEGRAM_CHAT_ID`, `WEBHOOK_URL`。

### 阶段 4：轻量级跨日历史去重 (`internal/history`)
- 文件：`internal/history/history.go` 与 `internal/history/history_test.go`
- 功能：
  - 维护一个体积小巧的 JSON 文件（默认 `cache/history.json`）
  - 记录已推送新闻的 URL Hash 与推送时间
  - 自动清理超过 N 天（默认 7 天）的过期记录
  - CLI 增加 `--history` 参数支持开启

### 阶段 5：源抓取健康度看板与 CLI 体验升级
- 文件：`internal/digest/digest.go`、`cmd/newspocket/main.go`
- 功能：记录每个源的耗时与成功条数；CLI 打印精美 ASCII 报告；增加 `--format` 与 `--output-md` 参数。

### 阶段 6：精选源模板库 (Presets) 与 文档更新
- 目录：`config/presets/`（`tech.json`, `ai.json`, `hotsearch.json`）
- 更新 `README.md` 与配置说明。

---

## 三、 验证计划

1. **单元测试**：针对 `cleaner_test.go`、`markdown_test.go`、`pusher_test.go`、`history_test.go` 编写 100% 覆盖的测试。
2. **集成测试**：
   - 运行 `./newspocket --test --output-md output.md`，检查 `output.html` 与 `output.md` 生成效果。
   - 验证 URL 清洗效果。
   - 验证跨日历史去重：连续运行两次，第二次应识别并跳过已推内容。
