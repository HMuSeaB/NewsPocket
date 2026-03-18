# NewsPocket Review Checklist

这是给自己用的轻量 review 清单，目标不是“形式完整”，而是尽快发现会影响抓取、配置、发信和发版的问题。

## 改动前

- 这次改动有没有明确范围？
- 这次改动更像 `fix`、`refactor` 还是 `release prep`？
- 有没有碰到下面这些高风险区：
  - `config/sources.json` 读写
  - GUI 改配置并保存
  - JSON API 字段映射
  - 时间过滤
  - 邮件发送
  - GitHub Actions / release

## 看 Diff

- 有没有误删配置字段，尤其是 `json_config` 里的字段？
- 有没有把默认行为改掉但没有显式说明？
- 有没有为了修一个问题顺手改 UI 文案、样式、编码或注释？
- 有没有引入和本次需求无关的“顺手优化”？
- 改动是否足够小，小到以后出问题时能快速定位？

## 配置与数据

- RSS 源和 JSON API 源都还能正常走通吗？
- JSON API 的这些字段是否还完整保留：
  - `items_path`
  - `title_field`
  - `summary_field`
  - `time_field`
  - `link_field`
  - `link_template`
- GUI 打开已有配置后，再保存一次，会不会把未知字段删掉？
- 布尔值和默认值会不会在保存时被改写？

## 抓取逻辑

- `GET` / `POST` 方法是否按配置真实发送？
- 没有 body 的 `POST` 会不会被错误降级成 `GET`？
- 自定义 headers 还会不会生效？
- 失败时日志是否足够定位问题？

## 解析逻辑

- `hours_lookback` 对 RSS 和 JSON API 是否一致生效？
- 没有时间字段的数据是否仍然允许保留？
- 摘要清洗、截断后是否还可读？
- 分类和来源分组有没有被打乱？

## 邮件与渲染

- 中文标题 / 邮件主题会不会乱码？
- 模板里的链接、摘要、时间是否都正常显示？
- `output.html` 能否正常生成并打开？

## GUI

- 新增源、切换源、删除源、保存配置是否正常？
- “测试抓取”返回是否和真实配置一致？
- 表单里没显示的字段会不会被保存过程清掉？

## 验证命令

- `go test ./...`
- `npm run build` in `cmd/newspocket-gui/frontend`
- 如涉及 CLI 输出：`go build ./cmd/newspocket`

## 发版前

- 这次改动是否值得发 patch 版？
- release notes 是否说清楚“修了什么”和“影响谁”？
- 是否确认没有把无关 UI 文案或风格改动混进 release？
- tag 之前先看一遍 `git diff --stat`

