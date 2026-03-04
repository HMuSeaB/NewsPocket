// NewsPocket CLI 入口
// 协调 配置加载 → 并发抓取 → 内容解析 → 模板渲染 → 邮件发送 的完整流程
package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/config"
	"github.com/HMuSeaB/NewsPocket/internal/fetcher"
	"github.com/HMuSeaB/NewsPocket/internal/mailer"
	"github.com/HMuSeaB/NewsPocket/internal/parser"
	"github.com/HMuSeaB/NewsPocket/internal/renderer"
)

func main() {
	// 命令行参数
	configPath := flag.String("config", "config/sources.json", "配置文件路径")
	testMode := flag.Bool("test", false, "测试模式：生成 output.html 不发送邮件")
	timeout := flag.Int("timeout", 20, "单个源的抓取超时时间(秒)")
	flag.Parse()

	// 配置日志
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	slog.Info("=== NewsPocket 开始运行 ===")

	// 1. 加载配置
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		slog.Error("加载配置文件失败", "error", err)
		os.Exit(1)
	}

	sources := cfg.EnabledSources()
	slog.Info("配置加载完成",
		"total", len(cfg.Sources),
		"enabled", len(sources),
	)

	if len(sources) == 0 {
		slog.Warn("没有启用的源，程序结束")
		os.Exit(0)
	}

	// 2. 并发抓取
	f := fetcher.New(time.Duration(*timeout) * time.Second)
	results := f.FetchAll(sources)

	if len(results) == 0 {
		slog.Warn("未抓取到任何内容，程序结束")
		os.Exit(0)
	}

	// 3. 解析和清洗
	p := parser.New(cfg.Settings.SummaryMaxLength, cfg.Settings.HoursLookback)
	allItems := p.ParseAll(results, cfg.Settings.MaxItemsPerSource)

	if len(allItems) == 0 {
		slog.Warn("解析后无有效内容（可能所有内容都已过期），程序结束")
		os.Exit(0)
	}

	// 4. 分组统计
	sections := parser.GroupByCategory(allItems, cfg.Sources)

	// 统计来源数
	sourceSet := make(map[string]struct{})
	for _, item := range allItems {
		sourceSet[item.Source] = struct{}{}
	}

	slog.Info("统计信息",
		"total", len(allItems),
		"sources", len(sourceSet),
		"categories", len(sections),
	)

	// 5. 渲染 + 发送
	beijing := time.FixedZone("CST", 8*3600)
	today := time.Now().In(beijing).Format("2006年01月02日 Monday")

	// 确定模板目录（相对于可执行文件或当前目录）
	templateDir := resolveTemplateDir()

	r := renderer.New(templateDir)

	titleSuffix := ""
	if *testMode {
		titleSuffix = " (测试)"
	}

	data := renderer.TemplateData{
		Title:         fmt.Sprintf("NewsPocket 晨报 - %s%s", today, titleSuffix),
		Date:          today,
		TotalCount:    len(allItems),
		SourceCount:   len(sourceSet),
		CategoryCount: len(sections),
		Sections:      sections,
	}

	htmlContent, err := r.Render(data)
	if err != nil {
		slog.Error("模板渲染失败", "error", err)
		os.Exit(1)
	}

	if *testMode {
		// 测试模式：输出到文件
		if err := os.WriteFile("output.html", []byte(htmlContent), 0644); err != nil {
			slog.Error("写出测试文件失败", "error", err)
			os.Exit(1)
		}
		slog.Info("测试文件已生成: output.html")
	} else {
		// 生产模式：发送邮件
		mailCfg, err := mailer.LoadFromEnv()
		if err != nil {
			slog.Error("邮件配置错误", "error", err)
			os.Exit(1)
		}

		subject := fmt.Sprintf("NewsPocket 每日简报 - %s", today)
		if err := mailer.SendHTML(mailCfg, subject, htmlContent); err != nil {
			slog.Error("邮件发送失败", "error", err)
			os.Exit(1)
		}
	}

	slog.Info("=== NewsPocket 运行完成 ===")
}

// resolveTemplateDir 按优先级查找模板目录：
// 1. 可执行文件同级的 templates/
// 2. 当前工作目录的 templates/
func resolveTemplateDir() string {
	// 尝试可执行文件所在目录
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Join(filepath.Dir(exe), "templates")
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}

	// 回退到工作目录
	return "templates"
}
