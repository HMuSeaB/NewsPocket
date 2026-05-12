# NewsPocket

## 项目概述

NewsPocket 是一个轻量级、自动化、零成本的新闻聚合助手。每天早上自动抓取全球优质新闻源（RSS/JSON API），整理成精美的 HTML 简报发送到指定邮箱。
本项目已使用 **Golang** 进行全面重构，不仅极大地提高了单机并发抓取的性能，还通过 Wails 实现了现代化的跨平台管理后台。

## 项目结构

```
NewsPocket/
├── .github/
│   └── workflows/
│       ├── daily_news.yml          # GitHub Actions 每日定点推送
│       └── release.yml             # GitHub Actions 自动交叉编译发版
├── cmd/
│   ├── newspocket/
│   │   └── main.go                 # NewsPocket 核心引擎主入口
│   └── newspocket-gui/
│       ├── main.go                 # Wails 桌面管理软件入口
│       └── frontend/               # Wails 前端 (Vanilla JS + HTML/CSS)
├── config/
│   └── sources.json                # 新闻源配置文件
├── internal/
│   ├── config/                     # 配置反序列化解析
│   ├── fetcher/                    # 并发请求、RSS/JSON API 下载器
│   ├── mailer/                     # SMTP 邮件发送渲染引擎
│   └── parser/                     # 降噪、时间过滤、归类解析器
├── templates/
│   └── email.gohtml                # 邮件早报 HTML Go 模板
├── go.mod                          # Go 模块文件
└── README.md                       # 项目说明
```

## 技术栈

- **核心引擎**: Golang 1.23+
- **并发抓取**: 原生 Goroutine 并发编排
- **GUI 界面**: Wails v2 (Go + Webview2 / Webkit)
- **前端栈**: Vanilla HTML/JavaScript/CSS (零 React/Vue 负担，究极轻量)
- **自动化**: GitHub Actions (支持跨平台打 Tag 自动跨译)

## 核心工作流

1. **桌面配置管理** (`newspocket-gui`) 
   通过友好的 UI 管理订阅源（支持 RSS / 复杂 JSON API 的 JQ 路径解析）。
   点击“测试抓取”即可立刻调用 Core Engine 返回抓取验证数据。
2. **零成本每日下发** (`daily_news.yml`)
   GitHub Actions 每日 00:15 UTC 拉取最新的 GitHub Release 二进制文件，传入内置的 `sources.json` 和 SMTP Secrets（存储于 Settings），执行一键推送！

## 配置文件说明

### sources.json 示例

```json
{
  "sources": [
    {
      "name": "Weibo 热搜",
      "url": "https://weibo.com/ajax/side/hotSearch",
      "category": "Social",
      "type": "json_api",
      "enabled": true,
      "headers": {"Referer": "https://weibo.com"},
      "json_config": {
        "items_path": "data.realtime",
        "title_field": "word",
        "link_template": "https://s.weibo.com/weibo?q=%23{word}%23"
      }
    }
  ],
  "settings": {
    "max_items_per_source": 10,
    "hours_lookback": 24,
    "summary_max_length": 300
  }
}
```

## 快速开发与编译

### 构建 CLI 核心引擎
```bash
go build -o newspocket.exe ./cmd/newspocket
.\newspocket.exe --test
```

### 构建/调试桌面配置端
```bash
cd cmd/newspocket-gui
wails dev     # 启动热重载开发模式
wails build   # 打包独立二进制桌面程序
```
