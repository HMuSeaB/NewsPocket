# 📰 NewsPocket

一个轻量级、自动化、零成本的新闻聚合助手。每天早上自动抓取全球优质新闻源（RSS/JSON API/自定义脚本），整理成精美的 HTML 简报发送到你的邮箱。

![Python](https://img.shields.io/badge/Python-3.9+-blue.svg)
![GitHub Actions](https://img.shields.io/badge/GitHub_Actions-Automated-green.svg)
![License](https://img.shields.io/badge/License-MIT-purple.svg)

## ✨ 特性

- **零成本**: 完全基于 GitHub Actions 运行，无需购买服务器。
- **多源支持**:
  - 📡 **标准 RSS**: 支持所有标准 RSS/Atom 源。
  - 🔌 **JSON API**: 支持微博热搜、B站热门等 JSON 接口，可自定义字段映射。
  - 🐍 **自定义脚本**: 支持 Python/Node.js 脚本，处理复杂抓取逻辑。
- **可视化管理**: 基于 Streamlit 的强大管理面板，支持所见即所得的配置修改和真实环境测试。
- **智能聚合**: 自动清洗 HTML 标签，统一格式，按板块分类。
- **精美排版**: "Morning Brew" 风格的响应式邮件模板，手机阅读体验极佳，浏览器预览支持可视化图表。

## 🚀 快速开始

### 1. Fork 本仓库
点击右上角的 **Fork** 按钮，将项目复制到你自己的 GitHub 账号。

### 2. 配置 GitHub Secrets
进入你的 Fork 仓库，点击 `Settings` -> `Secrets and variables` -> `Actions` -> `New repository secret`，添加以下变量：

| 变量名 | 必填 | 说明 | 示例 |
|--------|------|------|------|
| `EMAIL_USER` | ✅ | 发件人邮箱地址 | `example@qq.com` |
| `EMAIL_PASS` | ✅ | SMTP 授权码 (非登录密码) | `abcdefghijklmn` |
| `EMAIL_TO` | ❌ | 收件人邮箱 (默认发给自己) | `user1@test.com,user2@test.com` |
| `EMAIL_HOST` | ❌ | SMTP 服务器 (默认 QQ 邮箱) | `smtp.qq.com` |
| `EMAIL_PORT` | ❌ | SMTP 端口 (默认 465 SSL) | `465` |

### 3. 自定义新闻源
你可以直接编辑 `config/sources.json`，或者使用下方的**管理面板**进行可视化配置。

#### 支持的源类型：
1. **RSS 源**: 标准的 RSS/Atom 订阅链接。
2. **JSON API**: 如微博热搜，需配置 `json_config` 映射字段。
3. **Script**: 自定义 Python 脚本 (位于 `scripts/` 目录)。

### 4. 手动测试
进入 `Actions` 页面，选择 `Daily News Digest`，点击 `Run workflow` 手动触发一次，检查是否收到邮件。

## 🎛️ API 管理面板 (推荐)

NewsPocket 内置了一个基于 **Streamlit** 的可视化管理面板，彻底解决了跨域问题，支持本地配置的实时保存与测试。

1. **安装依赖**：
   ```bash
   pip install -r requirements.txt
   ```

2. **启动面板**：
   ```bash
   streamlit run admin_dashboard.py
   ```
   浏览器会自动打开 `http://localhost:8501`。

3. **功能亮点**：
   - **📊 模块化看板**：内置数据可视化图表，直观展示来源分布。
   - **📝 所见即所得**：支持 Table（批量编辑）和 Card（折叠查看）两种视图，轻松管理配置。
   - **🪄 智能模版**：内置 "RSS", "JSON API", "微博热搜" 等模版，一键添加新源，告别手写 JSON。
   - **🧪 真机测试**：后端 Python 发起真实请求（无视 CORS），直接预览抓取到的清洗数据和原始 JSON。
   - **📧 邮件预览**：一键生成包含图表和导航的 HTML 日报预览。

## 🛠️ 本地开发

1. 克隆仓库并安装依赖
```bash
git clone https://github.com/HMuSeaB/NewsPocket.git
cd NewsPocket
pip install -r requirements.txt
```

2. 运行测试 (生成 HTML 文件但不发送邮件)
```bash
python main.py --test
```

3. 发送真实邮件
设置环境变量 (`EMAIL_USER`, `EMAIL_PASS`) 后直接运行 `python main.py`。

## 📄 License
MIT License
