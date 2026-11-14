# 📰 NewsPocket

一个轻量级、自动化、零依赖的新闻聚合助手。每天早上自动抓取全球优质新闻源（RSS/JSON API），整理成精美的 HTML 简报与 Markdown 早报，推送至邮箱及各即时通讯群组。

**本项目已从 Python 重写为 Go (v2.2)，带来极速的启动体验、原生的并发抓取和零依赖的单文件部署。**

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)
![GitHub Actions](https://img.shields.io/badge/GitHub_Actions-Automated-green.svg)
![License](https://img.shields.io/badge/License-MIT-purple.svg)

## ✨ 特性

- **零成本**: 完全基于 GitHub Actions 运行，无需购买服务器。
- **极速性能**: 采用 Go 原生 Goroutine 并发抓取，毫秒级响应，编译为单一二进制文件。
- **多源与相对链接补全**:
  - 📡 **标准 RSS**: 支持所有标准 RSS/Atom 源。
  - 🔌 **JSON API**: 支持微博热搜、B站热门、IT之家等复杂接口，自动补全相对路径与时区解析。
- **🤖 多渠道消息推送 (全网首创)**:
  - 📧 **SMTP 邮件**: 经典的 Cinematic Dusk 霓虹暮光 HTML 邮件排版，全客户端兼容。
  - 💬 **即时通信 Webhook**: 原生支持 **企业微信群机器人**、**飞书/Lark**、**钉钉**、**Telegram Bot** 及通用 Webhook，一键直达手机群聊！
- **⚡ Jev 智能精选与降噪**: 基于 TypeSafe System One (Jev) 认知模型，运用 **Speculative Fan-out（推测式展开）** 并发对抓取的全量新闻进行毫秒级价值评分（Score）与领域分类（Choice），自动剔除低质营销水文，精准筛选 Top 精华。
- **🧠 AI 要闻速读**: 兼容任意 OpenAI Chat Completions 协议 (DeepSeek / Moonshot / OpenAI / Ollama 等)，晨报顶部自动附加大模型提炼的「今日核心要点 1 分钟速读」。
- **🛡️ 跨日持久化去重 (Anti-Fatigue)**: 内置轻量去重记忆库，自动过滤最近 N 天内已推送过的长效热点，告别同个热搜连续多日重复轰炸。
- **🧹 URL 深度清洗 (Anti-Tracking)**: 自动剥离 `utm_*`, `spm_*`, `refer_flag` 等推广埋点参数，净化链接，避免伪重复。
- **📑 双模渲染导出**: 同时生成极致视觉的 HTML 邮件与 GitHub Flavored Markdown 晨报文档，完美适配知识库与博客归档。
- **📊 信源健康度看板**: 抓取完成后终端打印直观的 ASCII 健康度面板，毫秒级耗时与收录状态一览无余。
- **📦 精选预设集**: 内置 `config/presets/`（AI 开发者、科技数码、国内热搜）多套开箱即用源配置。
- **桌面管理端**: 内置 Wails 跨平台 GUI，可视化增删改订阅源、单源测试、OPML 导入导出与实时渲染预览。

## 🚀 快速开始 (GitHub Actions 自动化)

本项目的核心理念是通过 Fork + GitHub Actions 来实现无人值守的自动化早报推送。

1. **Fork 本仓库** 到你的 GitHub 账号下。
2. 转到你 Fork 后的仓库的 **Settings -> Secrets and variables -> Actions**, 点击 **New repository secret**，按需添加以下环境变量：

   **A. 邮件推送（可选）：**
   - `EMAIL_HOST`: SMTP 服务器地址 (默认: `smtp.qq.com`)
   - `EMAIL_PORT`: SMTP 服务器端口 (默认: `465`)
   - `EMAIL_USER`: 你的发件邮箱账号 (例: `abc@qq.com`)
   - `EMAIL_PASS`: 你的发件邮箱授权码 (不是登录密码！)
   - `EMAIL_TO`: 收件人邮箱，支持英文逗号分隔的多个邮箱 (如果不填则默认发送给 `EMAIL_USER`)

   **B. 即时通讯群机器人推送（可选，按需配置任意一个）：**
   - `WECOM_WEBHOOK`: 企业微信机器人 Webhook 地址
   - `FEISHU_WEBHOOK`: 飞书自定义机器人 Webhook 地址
   - `DINGTALK_WEBHOOK`: 钉钉机器人 Webhook 地址
   - `TELEGRAM_BOT_TOKEN` & `TELEGRAM_CHAT_ID`: Telegram Bot 密钥与目标会话 ID
   - `WEBHOOK_URL`: 通用标准 JSON Webhook 接收地址

   **C. ⚡ Jev 智能精选与降噪（可选，推荐配置）：**
   - `TYPESAFE_API_KEY`: TypeSafe API 密钥（配置后自动开启 Jev 毫秒级打分重排与水文过滤，零幻觉）
   - `TYPESAFE_BASE_URL`: API 地址（可选，默认: `https://api.typesafe.ai/v1`）

   **D. 🤖 AI 要闻速读（可选，不配则跳过此模块）：**
   - `AI_API_KEY`: 大模型服务的 API Key (如 [DeepSeek](https://platform.deepseek.com/) 平台密钥)
   - `AI_BASE_URL`: API 地址 (默认: `https://api.deepseek.com/v1`)
   - `AI_MODEL`: 模型名 (默认: `deepseek-chat`)
   - `AI_PROMPT`: 自定义系统提示词（可选）
   - `AI_TIMEOUT`: 请求超时秒数 (默认: `45`)
   - `AI_MAX_ITEMS`: 送入大模型的最大条目数 (默认: `25`)

3. 修改仓库按需配置 [config/sources.json](./config/sources.json) 或选用 [config/presets/](./config/presets/) 预设。
4. 切换到仓库的 **Actions** 面板，允许运行 workflows。点击左侧的 "Daily News Digest"，然后点击 **Run workflow** 手动触发第一次执行。
5. 之后每天北京时间早上 08:15（UTC 00:15）会自动将早报发送到你的目标渠道！

## 💻 本地运行与开发

下载或克隆本仓库后，你可以按照以下步骤在本地运行和开发：

### 前置要求

- 安装 [Go 1.22+](https://go.dev/)
- （仅桌面管理端需要）安装 [Wails v2](https://wails.io/) 与 WebView2 运行时

### 编译与运行 CLI 核心引擎

提供 `--test` 模式，该模式下抓取并渲染后会输出终端看板并生成 `output.html` 与 `output.md`，不会实际外发：

```bash
# 下载依赖并编译
go mod tidy
go build -o newspocket ./cmd/newspocket

# 测试运行（生成 output.html 与 output.md）
./newspocket --test --config config/sources.json

# 启用跨日历史去重运行（记忆 7 天，自动去疲劳）
./newspocket --test --history cache/history.json --retention 7

# 生产运行（需配置环境变量）
export EMAIL_USER="..."
export EMAIL_PASS="..."
export FEISHU_WEBHOOK="https://open.feishu.cn/open-apis/bot/v2/hook/..." # 或 WECOM_WEBHOOK 等
./newspocket --config config/sources.json
```

### 桌面管理端 (GUI)

```bash
cd cmd/newspocket-gui
wails dev     # 启动热重载开发模式
wails build   # 打包独立二进制桌面程序
```

GUI 支持可视化编辑订阅源、单源测试抓取、OPML 导入导出与晨报实时预览。

## 📂 核心配置与开箱即用预设

核心配置位于 `config/sources.json`。仓库内置了开箱即用的预设源模板：

- [config/presets/ai.json](./config/presets/ai.json)：AI 开发者与极客专用精选源（Hacker News、OpenAI Blog、Hugging Face、TechCrunch AI 等）
- [config/presets/tech.json](./config/presets/tech.json)：科技数码与综合行业前沿（少数派、爱范儿、The Verge、IT之家等）
- [config/presets/hotsearch.json](./config/presets/hotsearch.json)：国内热榜舆情综合（微博热搜、B站热门等）

## 🤝 贡献指南

1. 欢迎提交 Issue 或 Pull Request 加入新的信息源或功能。
2. 请确保提交前运行 `go fmt ./...` 并保证 `go test ./...` 全量通过。

## 📄 License

[MIT License](LICENSE)
