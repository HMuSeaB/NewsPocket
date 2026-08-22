package renderer

import (
	"strings"
	"testing"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/parser"
)

func TestRenderWithoutAISummary(t *testing.T) {
	t.Parallel()

	r := New()
	data := TemplateData{
		Title:         "NewsPocket 晨报",
		Date:          "2026年08月22日 星期六",
		TotalCount:    2,
		SourceCount:   1,
		CategoryCount: 1,
		Sections: []parser.CategorySection{
			{
				Index: 1,
				Name:  "行业动态",
				Groups: []parser.SourceGroup{
					{
						Name:      "36氪",
						Collapsed: false,
						Items: []parser.NewsItem{
							{
								Title:   "测试资讯标题",
								Time:    "2026-08-22 10:00",
								Summary: "这是一条测试摘要",
								Link:    "https://example.com/news/1",
								Source:  "36氪",
							},
						},
					},
				},
			},
		},
	}

	htmlContent, err := r.Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(htmlContent, "测试资讯标题") {
		t.Errorf("rendered HTML does not contain item title")
	}

	if strings.Contains(htmlContent, "今日核心要闻 1 分钟速读") {
		t.Errorf("expected no AI Digest container when AISummary is empty")
	}
}

func TestRenderWithAISummary(t *testing.T) {
	t.Parallel()

	r := New()
	rawSummary := "1. **【AI 引擎上线】** NewsPocket 现已支持每日核心要闻速读。\n2. **【体验升级】** 桌面端全面支持 OPML 导入与实时预览。"
	formattedSummary := FormatAISummaryToHTML(rawSummary)

	data := TemplateData{
		Title:         "NewsPocket 晨报 (带 AI 速览)",
		Date:          "2026年08月22日 星期六",
		TotalCount:    1,
		SourceCount:   1,
		CategoryCount: 1,
		AISummary:     formattedSummary,
		Sections: []parser.CategorySection{
			{
				Index: 1,
				Name:  "科技生活",
				Groups: []parser.SourceGroup{
					{
						Name: "少数派",
						Items: []parser.NewsItem{
							{
								Title:   "效率神器推荐",
								TimeObj: time.Now(),
								Summary: "测试介绍",
								Link:    "https://example.com/2",
								Source:  "少数派",
							},
						},
					},
				},
			},
		},
	}

	htmlContent, err := r.Render(data)
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	if !strings.Contains(htmlContent, "今日核心要闻 1 分钟速读") {
		t.Errorf("expected AI Digest header in HTML")
	}

	if !strings.Contains(htmlContent, "AI 引擎上线") {
		t.Errorf("expected AI highlight title in HTML")
	}
}
