# NewsPocket 核心优化与重构任务看板

本看板用于实时追踪本次更新的开发进度，包含 13位时间戳解析支持、智能 URL 自动转义，以及去脚本统计条（Cinematic Dusk 风格）的渲染重构工作。

---

## 📋 任务状态清单

- `[x]` 任务一：安全备份与防灾准备
  - [x] 备份当前 `internal/renderer/templates/email.gohtml` 至安全副本，支持单独一键回退
- `[x]` 任务二：数据抓取解析重构 (`internal/fetcher/jsonapi.go`)
  - [x] 实现 `parseTimeString` 兼容 13 位毫秒级 Unix 时间戳解析
  - [x] 实现 `buildLink` 智能自动判断 `?` 位置并进行安全 Query/Path URL 转义
- `[x]` 任务三：邮件数据层渲染支持 (`internal/renderer/renderer.go`)
  - [x] 配合新版进度条统计，扩展数据层在无脚本状态下的聚合传入能力
- `[x]` 任务四：邮件渲染模板重构 (`internal/renderer/templates/email.gohtml`)
  - [x] 彻底移出 `<script>` 和外部 Chart.js CDN，消除邮件 SPAM 隐患
  - [x] 采用纯 HTML/CSS 表格结构，重构出极富 "Cinematic Dusk" 风格的 **霓虹橙渐变条形进度条统计图**
- `[x]` 任务五：单元测试与自动校验
  - [x] 在 `jsonapi_test.go` 中编写毫秒级时间戳、智能 URL 转义的单元测试用例
  - [x] 在本地运行 `go test ./...` 确保所有测试 100% 通过
- `[x]` 任务六：本地构建与视觉终检
  - [x] 构建 cli 核心：运行终端命令生成测试报告网页 `output.html`
  - [x] 严格审查浅色模式及深色自适应模式下的视觉完整度，确保 0 错乱、0 脚本、完美自适应
