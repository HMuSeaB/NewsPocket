// NewsPocket CLI 入口
// 协调 配置加载 → 并发抓取 → 内容解析 → 模板渲染 → 邮件发送 的完整流程
package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/ai"
	"github.com/HMuSeaB/NewsPocket/internal/config"
	"github.com/HMuSeaB/NewsPocket/internal/digest"
	"github.com/HMuSeaB/NewsPocket/internal/mailer"
	"github.com/HMuSeaB/NewsPocket/internal/timeutil"
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

	// 2. 执行晨报生成全流程（抓取 → 解析 → 分组 → AI 速览 → 渲染）
	titleSuffix := ""
	if *testMode {
		titleSuffix = " (测试)"
	}

	result, err := digest.Build(cfg, ai.NewClientFromSettings(cfg.Settings.AI), digest.Options{
		FetchTimeout: time.Duration(*timeout) * time.Second,
		TitleSuffix:  titleSuffix,
	})
	if err != nil {
		// 无内容属于正常业务场景（如所有源恰好都无更新），不算故障
		if errors.Is(err, digest.ErrNoSources) || errors.Is(err, digest.ErrNoResults) || errors.Is(err, digest.ErrNoItems) {
			slog.Warn("流程提前结束", "reason", err)
			os.Exit(0)
		}
		slog.Error("晨报生成失败", "error", err)
		os.Exit(1)
	}

	for _, fs := range result.FailedSources {
		slog.Warn("本次有源抓取失败", "source", fs.Name, "url", fs.URL, "reason", fs.Reason)
	}

	// 3. 输出：测试模式写文件 / 生产模式发邮件
	if *testMode {
		if err := os.WriteFile("output.html", []byte(result.HTML), 0644); err != nil {
			slog.Error("写出测试文件失败", "error", err)
			os.Exit(1)
		}
		slog.Info("测试文件已生成: output.html",
			"total", result.TotalCount,
			"sources", result.SourceCount,
			"failed_sources", len(result.FailedSources),
		)
	} else {
		mailCfg, err := mailer.LoadFromEnv()
		if err != nil {
			slog.Error("邮件配置错误", "error", err)
			os.Exit(1)
		}

		subject := fmt.Sprintf("NewsPocket 每日简报 - %s", timeutil.TodayString())
		if err := mailer.SendHTML(mailCfg, subject, result.HTML); err != nil {
			slog.Error("邮件发送失败", "error", err)
			os.Exit(1)
		}
	}

	slog.Info("=== NewsPocket 运行完成 ===")
}
