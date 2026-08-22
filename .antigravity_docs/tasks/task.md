# NewsPocket 核心功能升级任务看板 (AI 速览 · 邮件模板 · GUI 交互)

本看板用于追踪 AI 每日要闻速览模块、Cinematic Dusk 邮件模板跨客户端优化以及 Wails GUI 桌面端体验增强的开发进度。

---

## 📋 任务状态清单

- `[x]` 任务一：实现 AI 每日要闻速览模块 (`internal/ai`)
  - [x] 编写 `internal/ai/ai.go`，实现轻量级 OpenAI/DeepSeek 协议客户端与 Prompt 构造
  - [x] 编写 `internal/ai/ai_test.go`，覆盖正常生成、网络超时与无 Key 降级用例
  - [x] 在 `cmd/newspocket/main.go` 中挂载 AI 要闻提炼流程
- `[x]` 任务二：升级 Cinematic Dusk 邮件模板与排版兼容性
  - [x] 在 `internal/renderer/renderer.go` 中支持 `AISummary` 字段与 Markdown-to-HTML 渲染
  - [x] 在 `internal/renderer/templates/email.gohtml` 中新增 AI 速览霓虹卡片与客户端表格样式兼容优化
  - [x] 编写 `internal/renderer/renderer_test.go` 单元测试
- `[x]` 任务三：增强 Wails 桌面管理端 (OPML 导入导出与邮件预览)
  - [x] 在 `cmd/newspocket-gui/app.go` 中实现 `ImportOPML`、`ExportOPML`、`PreviewEmail`
  - [x] 在 `cmd/newspocket-gui/app_test.go` 中测试 OPML 导入、去重与导出
  - [x] 在 `frontend/index.html` 中新增搜索框、OPML 导入导出按钮和邮件预览 iframe 弹窗
  - [x] 在 `frontend/src/main.js` 中实现侧边栏一键启停开关、搜索过滤与预览联动
  - [x] 在 `frontend/src/style.css` 中注入 Cinematic Dusk 暮光组件样式
- `[x]` 任务四：全量测试与验证
  - [x] 运行 `go test -v ./...` 确保 100% 通过
  - [x] 运行 CLI 测试模式 `go run ./cmd/newspocket --test` 验证 `output.html` 渲染
