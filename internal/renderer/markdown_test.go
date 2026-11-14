package renderer

import (
	"strings"
	"testing"

	"github.com/HMuSeaB/NewsPocket/internal/parser"
)

func TestRenderMarkdown(t *testing.T) {
	r := New()

	testData := TemplateData{
		Title:         "NewsPocket 晨报 - 2026-11-14",
		Date:          "2026-11-14",
		TotalCount:    2,
		SourceCount:   1,
		CategoryCount: 1,
		Sections: []parser.CategorySection{
			{
				Index: 1,
				Name:  "行业动态",
				Groups: []parser.SourceGroup{
					{
						Name: "Hacker News",
						Items: []parser.NewsItem{
							{
								Title:   "Go 1.25 Released",
								Link:    "https://go.dev/doc/go1.25",
								Time:    "08:00",
								Summary: "Go 1.25 brings new memory management features.",
								Score:   4.8,
							},
							{
								Title:   "Show HN: NewsPocket v2.2",
								Link:    "https://github.com/HMuSeaB/NewsPocket",
								Time:    "08:30",
								Summary: "A lightweight news digest system.",
							},
						},
					},
				},
			},
		},
		SourceStats: []SourceStat{
			{Name: "Hacker News", Count: 2, Percentage: 100},
		},
	}

	rawAI := "1. **Go 1.25** 发布新版本，带来显著内存优化。\n2. **NewsPocket** 发布自动化早报更新。"

	md := r.RenderMarkdown(testData, rawAI)

	// 验证关键段落存在
	if !strings.Contains(md, "# 📰 NewsPocket 晨报 - 2026-11-14") {
		t.Errorf("Markdown 应包含主标题")
	}
	if !strings.Contains(md, "⚡ 今日 AI 核心要闻速览") {
		t.Errorf("Markdown 应包含 AI 速览卡片")
	}
	if !strings.Contains(md, "[Go 1.25 Released](https://go.dev/doc/go1.25)") {
		t.Errorf("Markdown 应包含条目链接")
	}
	if !strings.Contains(md, "⭐ 认知评分: 4.8") {
		t.Errorf("Markdown 应包含评分展示")
	}
	if !strings.Contains(md, "| Hacker News | 2 | 100% |") {
		t.Errorf("Markdown 应包含信源分布表格")
	}
}
