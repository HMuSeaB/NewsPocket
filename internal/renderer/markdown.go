package renderer

import (
	"fmt"
	"strings"
)

// RenderMarkdown 将晨报数据渲染为排版优雅的 GitHub Flavored Markdown 文本。
// 适用于：导出 output.md、Webhook (飞书/企业微信/钉钉/Telegram) 推送、以及同步到个人知识库。
func (r *Renderer) RenderMarkdown(data TemplateData, rawAIDigest string) string {
	var sb strings.Builder

	// 1. 标题与头部元数据
	sb.WriteString(fmt.Sprintf("# 📰 %s\n\n", data.Title))
	sb.WriteString(fmt.Sprintf("> 📅 **日期**: %s &nbsp;|&nbsp; 📊 **新闻总数**: %d 条 &nbsp;|&nbsp; 📡 **有效信源**: %d 个 &nbsp;|&nbsp; 📑 **板块**: %d 个\n\n",
		data.Date, data.TotalCount, data.SourceCount, data.CategoryCount))
	sb.WriteString("---\n\n")

	// 2. 今日 AI 核心要闻 1 分钟速读（若有）
	cleanAIDigest := strings.TrimSpace(rawAIDigest)
	if cleanAIDigest != "" {
		sb.WriteString("### ⚡ 今日 AI 核心要闻速览 (1 分钟速读)\n\n")
		sb.WriteString(cleanAIDigest)
		sb.WriteString("\n\n---\n\n")
	}

	// 3. 各板块新闻内容
	for _, section := range data.Sections {
		sb.WriteString(fmt.Sprintf("## 📌 %s\n\n", section.Name))

		for _, group := range section.Groups {
			sb.WriteString(fmt.Sprintf("### 📡 %s (%d)\n\n", group.Name, len(group.Items)))

			for _, item := range group.Items {
				// 标题带链接
				linkTitle := item.Title
				if item.Link != "" {
					linkTitle = fmt.Sprintf("[%s](%s)", item.Title, item.Link)
				}

				// 附加信息：时间、评分
				metaInfo := item.Time
				if item.Score > 0 {
					metaInfo = fmt.Sprintf("%s · ⭐ 认知评分: %.1f", metaInfo, item.Score)
				}

				sb.WriteString(fmt.Sprintf("- **%s** `[%s]`\n", linkTitle, metaInfo))
				if item.Summary != "" && item.Summary != item.Title {
					sb.WriteString(fmt.Sprintf("  > %s\n", strings.ReplaceAll(item.Summary, "\n", " ")))
				}
			}
			sb.WriteString("\n")
		}
	}

	// 4. 统计信息
	if len(data.SourceStats) > 0 {
		sb.WriteString("### 📊 信源分布统计\n\n")
		sb.WriteString("| 来源平台 | 收录条数 | 占比 |\n")
		sb.WriteString("| :--- | :--- | :--- |\n")
		for _, stat := range data.SourceStats {
			sb.WriteString(fmt.Sprintf("| %s | %d | %d%% |\n", stat.Name, stat.Count, stat.Percentage))
		}
		sb.WriteString("\n")
	}

	// 5. 页脚
	sb.WriteString("---\n")
	sb.WriteString("*本简报由 [NewsPocket](https://github.com/HMuSeaB/NewsPocket) 自动化引擎生成 · 轻量、极速、无广告*\n")

	return sb.String()
}
