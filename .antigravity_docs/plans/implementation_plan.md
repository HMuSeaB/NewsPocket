# NewsPocket 核心引擎与邮件模板重构实施计划

本计划旨在通过对 NewsPocket 数据抓取解析引擎及邮件模板进行重构，全面增强系统的稳定性、安全兼容性以及视觉展现质量。

---

## 1. 变更详情汇总 (Proposed Changes)

为了实现已对齐的决策，我们将对以下模块和文件进行修改：

### 🛠️ 数据抓取与解析模块 (fetcher)

#### [MODIFY] [jsonapi.go](file:///d:/4rchive/Code/NewsPocket/internal/fetcher/jsonapi.go)
* **时间戳解析增强 (`parseTimeString`)**：
  * 新增对 13 位毫秒级 Unix 时间戳字符串/数值的安全识别与解析。
  * 保持对 10 位秒级 Unix 时间戳的完美向后兼容。
* **智能自动 URL 编码 (`buildLink`)**：
  * 解析模板占位符 `{key}` 在 URL 中所处的位置：
    * 若在问号 `?` 之后（Query 参数区），采用 `url.QueryEscape` 自动转义替换的值。
    * 若在问号 `?` 之前（Path 路径区），采用 `url.PathEscape` 对特殊及中文字符进行安全转义，保留链接基础骨架。
  * 用户无需修改现有的 `sources.json` 配置文件，实现零配置无感升级。

---

### 🎨 邮件渲染与表现模块 (renderer)

#### [MODIFY] [renderer.go](file:///d:/4rchive/Code/NewsPocket/internal/renderer/renderer.go)
* 配合邮件模板，支持统计数据的传入。
* 修改 `TemplateData` 的字段，或通过模板内置方法对传入的新闻源及分类数据进行列表聚合，支持在无脚本（No-JS）环境下直接循环渲染来源占比进度条。

#### [MODIFY] [email.gohtml](file:///d:/4rchive/Code/NewsPocket/internal/renderer/templates/email.gohtml)
* **彻底去脚本化 (0 SPAM / 100% 兼容)**：
  * 彻底移除底部的 Chart.js 外部 CDN 链接（`<script src="https://..."></script>`）以及所有内部 `<script>` 逻辑，消除邮件网关反垃圾过滤的拦截隐患。
* **Cinematic Dusk 风格来源统计条 (Progress Bar List)**：
  * 在邮件顶部（原 Canvas 区域）设计一套基于纯 HTML/CSS（`<table width="100%">`）配合 HSL / 霓虹渐变色的“来源抓取量进度条列表”。
  * 采用 HSL 精修的深色背景（`#0d0d12`）与柔和的霓虹橙（`#ff6b4a` 到 `#f97316` 渐变）发光进度条。
  * 完美支持自适应深色模式，并在支持 CSS 动画的邮件客户端（如 Apple Mail / iOS Mail）中提供微光淡入与平滑伸展微动画。

---

## 2. 验证与测试计划 (Verification Plan)

### 自动化单元测试 (Automated Tests)
* **时间戳与 URL 编码测试**：
  * 在 `internal/fetcher/jsonapi_test.go` 中，编写针对 13 位毫秒级时间戳解析的测试用例。
  * 编写针对 Query 参数自动转义（含中文热搜词占位符）的测试用例。
  * 执行命令：`go test ./...` 确保所有单元测试 100% 通过。

### 视觉与渲染手动校验 (Manual Verification)
1. **本地测试模式构建**：
   * 在终端运行：
     ```bash
     go build -o newspocket.exe ./cmd/newspocket
     .\newspocket.exe --test
     ```
2. **输出预览**：
   * 打开生成的 `output.html`，校验顶部的 **Cinematic Dusk 风格来源抓取占比进度条** 是否在不同屏幕尺寸下均能完美自适应。
   * 分别在“浅色模式”与“系统深色模式”下，通过浏览器控制台模拟或手动切屏，确认其背景及霓虹线能柔和响应，没有排版变形或色彩突兀。
