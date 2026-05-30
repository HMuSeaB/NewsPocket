# NewsPocket 项目优化与更新审查报告

本报告对 NewsPocket 项目（基于 Golang 及 Wails 构建的新闻早报聚合助手）的源码与架构进行了深度审查。我们针对系统稳定性、兼容性及健壮性，发现了以下几个极具价值的优化和更新方向。

---

## 1. 核心发现与优化建议

### 🔍 发现一：JSON API 毫秒级时间戳（13位）解析缺失
- **现状分析**：
  在 [jsonapi.go](file:///d:/4rchive/Code/NewsPocket/internal/fetcher/jsonapi.go#L169-L203) 的 `parseTimeString` 函数中，系统仅支持 10 位的 Unix 时间戳（秒级）：
  ```go
  // 尝试 Unix 时间戳（秒）
  if len(s) == 10 {
      var ts int64
      if _, err := fmt.Sscanf(s, "%d", &ts); err == nil && ts > 1000000000 {
          return time.Unix(ts, 0).UTC()
      }
  }
  ```
  然而，在实际对接现代 JSON API 时（如 Java 后端默认返回、新浪微博 API 或各类社交平台接口），时间戳通常以 **13 位毫秒** 形式存在。如果 API 返回 13 位毫秒时间戳，解析器将直接失败并退避为零值（`未知时间`）。
- **优化建议**：
  在 `parseTimeString` 中增加对 13 位毫秒时间戳的智能识别与解析支持：
  ```go
  // 尝试 Unix 时间戳（毫秒）
  if len(s) == 13 {
      var ts int64
      if _, err := fmt.Sscanf(s, "%d", &ts); err == nil && ts > 1000000000000 {
          return time.UnixMilli(ts).UTC()
      }
  }
  ```

---

### 🔍 发现二：邮件客户端中 JS `<script>` 的安全与兼容性风险
- **现状分析**：
  In [email.gohtml](file:///d:/4rchive/Code/NewsPocket/internal/renderer/templates/email.gohtml#L806-L884) 模板底部，直接引入了 Chart.js 并通过 `<script>` 进行数据可视化渲染：
  ```html
  <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
  <script>
      document.addEventListener('DOMContentLoaded', function() { ... })
  </script>
  ```
  尽管对图表区域使用了 `<!--[if !mso]><!-->` 屏蔽了 Outlook，但是在绝大多数主流 Web 邮件客户端（Gmail, Outlook.com, QQ邮箱, 网易邮箱等）中：
  1. 出于极其严格的安全隔离考虑，**所有的 `<script>` 标签和行内 JS 都会被邮件网关强制剥离或禁用**。
  2. 含有 `<script>` 的邮件极易被识别为恶意欺诈邮件，从而直接丢进垃圾箱（SPAM）。
- **优化建议**：
  - **静态数据降级 (No-JS Fallback)**：即便 `<script>` 被剥离，我们应当确保页面依然能美观展示。例如，在 Canvas 图表框上方，若无 JS 支持，显示一个优雅的静态文字统计列表，或者在 CSS 中使用纯 HTML+CSS 技术（如 `<table width="100%">` 配合背景色条）实现无需 JS 的简易条形统计图。
  - **清晰提示**：图表区加注友好文案：“*提示：如需体验交互式统计图表，请在浏览器中打开此邮件*”。

---

### 🔍 发现三：JSON API `buildLink` 替换字符的 URL 编码（URL Encode）安全隐患
- **现状分析**：
  在 [jsonapi.go](file:///d:/4rchive/Code/NewsPocket/internal/fetcher/jsonapi.go#L131-L150) 的 `buildLink` 函数中，通过 URL 模板（如 `link_template`）直接替换占位符的值：
  ```go
  link := linkTemplate
  for key, value := range item {
      placeholder := "{" + key + "}"
      if strings.Contains(link, placeholder) {
          link = strings.ReplaceAll(link, placeholder, fmt.Sprintf("%v", value))
      }
  }
  ```
  但是，如果替换变量包含非 ASCII 字符（如微博热搜中的中文字符、特殊符号等），直接拼接出的 URL 会是类似 `https://s.weibo.com/weibo?q=%23苹果发布会%23`。
  一些邮件客户端或老旧浏览器在渲染此类超链接时，可能会发生**截断、乱码，甚至链接失效**。
- **优化建议**：
  在将值替换进 Query 参数时，应该使用 Go 标准库 `net/url` 对变量的值进行安全的 URL 编码转义：
  ```go
  import "net/url"
  
  // 对于可能包含中文字符的 placeholder，进行安全转义
  valStr := fmt.Sprintf("%v", value)
  escapedVal := url.QueryEscape(valStr) // 或在合适场景应用转义
  ```
  为了避免将整个链接结构转义（例如当占位符在 Path 路径中时），我们可以智能地只对参数部分进行 Query 编码，或者在模板中支持特定的格式指示符，例如 `{word|url}`。

---

## 2. 后续更新步骤建议

上述三项改动均为**低侵入性、高收益**的安全和健壮性更新：
1. **时间戳与 URL 转义修复**：可在 `internal/fetcher/` 模块中轻松无损重构，并附带完善的单元测试覆盖。
2. **邮件模板静态统计降级**：升级 `email.gohtml` 样式结构，确保任何邮件客户端都能稳定兼容，极大降低被误判为 SPAM 的风险。

请告诉我您更希望优先改进哪一部分，我将为您准备好正式的代码修改方案！
