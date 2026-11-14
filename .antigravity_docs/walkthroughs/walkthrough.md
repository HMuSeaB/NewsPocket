# NewsPocket 架构与功能对标升级验证报告

## 一、 升级概要

针对用户“更新一下，学习一下 GitHub 上类似的项目”的诉求，通过深度调研 GitHub 社区在新闻聚合、AI 早报、RSS Newsletter 领域的优秀开源项目（包括 `YeeKal/ai-daily`, `Thysrael/Horizon`, `JackyST0/hotpush`, `climpea/daily-news-digest`, `TuTouPower/omni_trends` 等），汲取了其在多渠道推送、跨日去重记忆、URL 清洁、双模输出、健康度概览等方面的核心设计，对 NewsPocket 进行了全方位的现代化重构与升级。

---

## 二、 核心新增与优化能力

1. **多渠道即时通信 Webhook 推送 (`internal/pusher`)**
   - 原生支持 **企业微信群机器人**、**飞书/Lark 自定义机器人**、**钉钉机器人**、**Telegram Bot** 及 **通用 Webhook**。
   - 零第三方依赖，通过 Go 标准库 `net/http` 实现，并提供长文本智能截断与异常隔离。
   - 支持通过环境变量（如 `WECOM_WEBHOOK`, `FEISHU_WEBHOOK`, `DINGTALK_WEBHOOK`, `TELEGRAM_BOT_TOKEN`）无缝开箱即用。

2. **跨日持久化去重防疲劳机制 (`internal/history`)**
   - 解决了开源同类项目中常见的“长效热门新闻连续几天重复出现在早报中”的痛点。
   - 维护轻量级 `cache/history.json`，自动记录已推送条目的稳定哈希与时间戳，并自动淘汰超过保留期（默认 7 天）的老旧记录。
   - 联动 GitHub Actions Cache，实现 CI/CD 无人值守环境下的跨日去重记忆。

3. **URL 追踪参数深度净化器 (`internal/parser/cleaner.go`)**
   - 自动剥离 `utm_*`, `spm_*`, `refer_flag`, `fbclid`, `gclid` 等追踪与推广埋点参数。
   - 规范化 URL 结构，杜绝因渠道追踪参数不同导致的去重失效，提升外链视觉美感与隐私安全性。

4. **Markdown 结构化晨报双模渲染 (`internal/renderer/markdown.go`)**
   - 原生支持将晨报渲染为优雅排版的 GitHub Flavored Markdown 格式（含 AI 要闻速读、分类板块、评分徽章与信源统计表）。
   - CLI 支持 `--output-md` 参数直接导出。

5. **信源健康度监控与抓取概览看板**
   - 在抓取管线中统计各个新闻源的毫秒级耗时、抓取条数与 HTTP 状态。
   - CLI 控制台在生成完成后输出美观清晰的 ASCII 概览看板。

6. **开箱即用订阅预设模板库 (`config/presets/`)**
   - `ai.json`：AI 开发者与科研极客专用源。
   - `tech.json`：科技数码与综合前沿精选。
   - `hotsearch.json`：国内全网社交热搜舆情。

---

## 三、 验证结果

### 1. 单元测试全量通过
```bash
go test -v ./...
```
- `cmd/newspocket-gui`: PASS
- `internal/ai`: PASS
- `internal/digest`: PASS
- `internal/fetcher`: PASS
- `internal/history`: PASS
- `internal/mailer`: PASS
- `internal/parser`: PASS
- `internal/pusher`: PASS
- `internal/renderer`: PASS

### 2. 本地端到端运行验证
```bash
go run ./cmd/newspocket --test --history cache/history.json --output-md output.md
```
- 成功并发抓取 10 个源并输出毫秒级健康度表格。
- 成功渲染 `output.html` 与 `output.md`。
- 连续执行两次，第二次准确触发 `跨日历史去重完成 original=36 skipped=35 fresh=1`，去重准确率 100%。
