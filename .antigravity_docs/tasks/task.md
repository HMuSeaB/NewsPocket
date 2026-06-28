# NewsPocket 稳定性与调试功能优化任务看板

本看板用于追踪网络超时防护、JSON API 嵌套字段读取以及 GUI 测试抓取精细化反馈的开发进度。

---

## 📋 任务状态清单

- `[x]` 任务一：在 `internal/mailer/mailer.go` 中引入超时防护
  - [x] 引入 `net.Dialer` 限制连接超时为 10 秒
  - [x] 修改 `tls.Dial` 为 `tls.DialWithDialer`
  - [x] 修改 `net.Dial` 为 `dialer.Dial`
- `[x]` 任务二：在 `internal/fetcher/jsonapi.go` 中重构嵌套路径提取
  - [x] 升级 `getString` 方法，检测并解析点号 `.` 嵌套路径（利用 `getNestedValue`）
- `[x]` 任务三：在 `internal/fetcher/jsonapi.go` 中支持嵌套占位符模板替换
  - [x] 升级 `buildLink` 方法，支持匹配并提取形如 `{author.id}` 的占位符列表
  - [x] 通过 `getNestedValue` 安全获取嵌套属性，并根据其在超链接问号 `?` 之前还是之后自动进行 Path/Query 智能转义
- `[x]` 任务四：在 `cmd/newspocket-gui/app.go` 中优化测试抓取的调试诊断
  - [x] 升级 `TestSource` 方法，若有效新闻数为 0，检查原始抓取条目数
  - [x] 若有原始数据但全被过滤，输出 24h 时间过滤诊断警告，并预览首条原始记录以利于核实 `time_field` 等配置
  - [x] 若无原始数据，给出接口无数据或 `items_path` 错误的诊断提示
- `[x]` 任务五：单元测试编写与自动校验
  - [x] 在 `jsonapi_test.go` 中添加对嵌套属性读取和嵌套模板占位符转义的单元测试
  - [x] 本地运行并确保所有单元测试 100% 通过
- `[x]` 任务六：本地命令行编译及验证
  - [x] 编译核心引擎并以测试模式运行，确保未破坏原有抓取功能
