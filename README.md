# 📰 NewsPocket

一个轻量级、自动化、零依赖的新闻聚合助手。每天早上自动抓取全球优质新闻源（RSS/JSON API），整理成精美的 HTML 简报发送到你的邮箱。

**本项目已从 Python 重写为 Go (v2.0)，带来极速的启动体验、原生的并发抓取和零依赖的单文件部署。**

![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)
![GitHub Actions](https://img.shields.io/badge/GitHub_Actions-Automated-green.svg)
![License](https://img.shields.io/badge/License-MIT-purple.svg)

## ✨ 特性

- **零成本**: 完全基于 GitHub Actions 运行，无需购买服务器。
- **极速性能**: 采用 Go 原生 Goroutine 并发抓取，编译为单一二进制文件，免去一切依赖安装烦恼。
- **多源支持**:
  - 📡 **标准 RSS**: 支持所有标准 RSS/Atom 源。
  - 🔌 **JSON API**: 支持微博热搜、B站热门等 JSON 接口，可自定义路径和时间字段解析。
- **智能过滤**: 自动清洗 HTML 噪音，智能在句子边界截断摘要，只推送最近 24 小时的内容。
- **精美排版**: "Morning Brew" 风格的响应式邮件模板，自动适配系统的亮色/暗色模式，支持可视化数据统计。

## 🚀 快速开始 (GitHub Actions 自动化)

本项目的核心理念是通过 Fork + GitHub Actions 来实现无人值守的自动化邮件推送。

1. **Fork 本仓库** 到你的 GitHub 账号下。
2. 转到你 Fork 后的仓库的 **Settings -> Secrets and variables -> Actions**, 点击 **New repository secret**，添加以下环境变量：
   - `EMAIL_HOST`: SMTP 服务器地址 (默认: `smtp.qq.com`)
   - `EMAIL_PORT`: SMTP 服务器端口 (默认: `465`)
   - `EMAIL_USER`: 你的发件邮箱账号 (例: `abc@qq.com`)
   - `EMAIL_PASS`: 你的发件邮箱授权码 (不是登录密码！)
   - `EMAIL_TO`: 收件人邮箱，支持英文逗号分隔的多个邮箱 (如果不填则默认发送给 `EMAIL_USER`)
3. 修改仓库按需配置 [config/sources.json](./config/sources.json)。
4. 切换到仓库的 **Actions** 面板，允许运行 workflows。点击左侧的 "Daily News Digest"，然后点击 **Run workflow** 手动触发第一次执行。
5. 之后每天北京时间早上 08:15（UTC 00:15）会自动将早报发送到你的邮箱！

## 💻 本地运行与开发

下载或克隆本仓库后，你可以按照以下步骤在本地运行和开发：

### 前置要求

- 安装 [Go 1.22+](https://go.dev/)

### 编译与运行

提供 `--test` 模式，该模式下抓取并渲染模板后会生成 `output.html` 而不会实际发送邮件：

```bash
# 下载依赖并编译
go mod tidy
go build -o newspocket ./cmd/newspocket

# 测试运行（生成 output.html 到当前目录）
./newspocket --test --config config/sources.json

# 生产运行（需配置环境变量）
export EMAIL_USER="..."
export EMAIL_PASS="..."
./newspocket --config config/sources.json
```

## 📂 核心配置说明

核心配置位于 `config/sources.json`。支持以下高级选项：

- **基础配置**: 支持源的重命名 (`name`)、URL (`url`) 和所属分类 (`category`)。
- **请求头**: 可以通过 `headers` 覆盖默认的 User-Agent 甚至提供 Cookie。
- **类型**: 支持 `rss` 和 `json_api`。
- **JSON API 解析**: 对于 `json_api`，可以通过 `json_config` 配置 `items_path` 提取文章列表，`title_field`, `link_field` 甚至 `link_template` 自定义组装链接格式。

## 🤝 贡献指南

1. 欢迎提交 Issue 或 Pull Request 加入新的信息源或功能。
2. 请确保提交前运行 `go fmt ./...` 并保证 `go build` 正常。

## 📄 License

[MIT License](LICENSE)
