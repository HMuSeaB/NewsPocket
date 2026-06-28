# 邮件超时、JSON嵌套提取与GUI可调试性优化计划

本实施计划旨在针对 NewsPocket 系统进行健壮性与易用性优化，包含以下三个核心提升：
1. **SMTP 发送网络超时限制**：防止由于网络无响应导致主流程无限阻塞。
2. **JSON API 嵌套路径与嵌套占位符解析**：支持子项中类似 `detail.title` 的点号路径提取，以及链接模板中 `{detail.id}` 的安全转义替换。
3. **GUI 测试抓取调试可视化**：在有效新闻为 0 时返回精细化的诊断数据，输出第一条原始数据的关键字段解析，方便排查配置缺陷。

---

## 用户评审要求

> [!IMPORTANT]
> 1. 本次对 `jsonapi.go` 的子字段提取方式由扁平查找（如 `item[key]`）升级为支持嵌套路径（`a.b.c`）。这是完全向后兼容的，因为当路径中没有点号时仍会回退到原有的扁平提取。
> 2. `app.go` 对 `TestSource` 返回的数据格式进行了扩展，在没有有效条目时输出 Markdown 诊断文本而非仅仅空列表提示，这需要前端能正常渲染该 Markdown/字符串文本。

---

## 开放性问题

目前暂无未决定的开放性问题。

---

## 拟定变更说明

### 1. 邮件网络层组件 (Mailer)

#### [MODIFY] [mailer.go](file:///d:/4rchive/Code/NewsPocket/internal/mailer/mailer.go)
* 修改 TCP 拨号和 TLS 建立连接的方式：
  * 引入 `net.Dialer` 并将 `Timeout` 设置为 10 秒。
  * 将 `tls.Dial` 修改为 `tls.DialWithDialer`。
  * 将 `net.Dial` 修改为 `dialer.Dial`。

---

### 2. 新闻数据抓取与解析组件 (Fetcher)

#### [MODIFY] [jsonapi.go](file:///d:/4rchive/Code/NewsPocket/internal/fetcher/jsonapi.go)
* 重构 `getString`：
  * 检测 `key` 中是否包含点号 `.`。如果包含，则调用 `getNestedValue` 进行逐级提取；否则沿用普通的 Map 字段提取。
* 重构 `buildLink`：
  * 支持提取 `linkTemplate` 中所有形如 `{...}` 的占位符（例如 `{detail.id}`）。
  * 对每个占位符使用 `getNestedValue` 提取值，并根据其在模板中的位置（Path 还是 Query）智能进行 `url.PathEscape` 或 `url.QueryEscape` 转义。

---

### 3. Wails 桌面交互层 (GUI)

#### [MODIFY] [app.go](file:///d:/4rchive/Code/NewsPocket/cmd/newspocket-gui/app.go)
* 重构 `TestSource` 返回逻辑：
  * 如果 `len(items) == 0`，但 `len(result.Entries) > 0`（即抓取解析到了原始数据，但被时间过滤器排除掉了），构造多行诊断报告，提示可能是 `time_field` 解析或 24 小时过期问题，并输出第一条原始数据的标题、链接、解析时间以及原始字段值。
  * 如果 `len(result.Entries) == 0`，提示可能 `items_path` 配置错误。

---

## 验证计划

### 自动化测试
* 在 `jsonapi_test.go` 中新增测试用例：
  * 验证子项嵌套字段读取（例如 `getString(item, "author.name")`）。
  * 验证链接模板中嵌套占位符替换（例如 `buildLink` 替换 `{author.id}`）。
* 运行以下命令确保所有测试 100% 通过：
  ```powershell
  go test ./internal/...
  ```

### 手动验证
* 在本地测试运行 CLI 并指定 `--test`，检查原有源抓取是否正常运行并输出 `output.html`：
  ```powershell
  go build -o newspocket.exe ./cmd/newspocket
  .\newspocket.exe --test
  ```
