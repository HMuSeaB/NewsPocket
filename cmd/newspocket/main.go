// NewsPocket CLI 入口
// 协调 配置加载 → 并发抓取 → 内容解析 → 跨日去重 → 模板渲染 → 多渠道推送(邮件/Webhook) 的完整流程
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/HMuSeaB/NewsPocket/internal/ai"
	"github.com/HMuSeaB/NewsPocket/internal/config"
	"github.com/HMuSeaB/NewsPocket/internal/digest"
	"github.com/HMuSeaB/NewsPocket/internal/history"
	"github.com/HMuSeaB/NewsPocket/internal/mailer"
	"github.com/HMuSeaB/NewsPocket/internal/pusher"
	"github.com/HMuSeaB/NewsPocket/internal/timeutil"
)

func main() {
	// 命令行参数
	configPath := flag.String("config", "config/sources.json", "配置文件路径")
	testMode := flag.Bool("test", false, "测试模式：生成 output.html / output.md，不执行实际外发")
	timeout := flag.Int("timeout", 20, "单个源的抓取超时时间(秒)")
	outputMD := flag.String("output-md", "", "同时导出 Markdown 格式到指定文件（如 output.md）")
	historyPath := flag.String("history", "", "跨日持久化去重文件路径（如 cache/history.json，留空则自动检测）")
	retentionDays := flag.Int("retention", 7, "跨日历史去重保留天数（默认 7 天）")
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

	// 2. 初始化跨日历史去重存储 (若指定或存在 cache/history.json)
	var histStore *history.Store
	dbPath := *historyPath
	if dbPath == "" {
		if _, err := os.Stat("cache/history.json"); err == nil {
			dbPath = "cache/history.json"
		}
	}
	if dbPath != "" {
		slog.Info("启用跨日历史去重", "path", dbPath, "retention_days", *retentionDays)
		store, hErr := history.OpenStore(dbPath, *retentionDays)
		if hErr != nil {
			slog.Warn("打开历史去重文件失败，本次将跳过跨日去重", "error", hErr)
		} else {
			histStore = store
		}
	}

	// 3. 执行晨报生成全流程（抓取 → 解析清洗 → 跨日去重 → 分组 → AI 速览 → HTML/MD 渲染）
	titleSuffix := ""
	if *testMode {
		titleSuffix = " (测试)"
	}

	result, err := digest.Build(cfg, ai.NewClientFromSettings(cfg.Settings.AI), digest.Options{
		FetchTimeout: time.Duration(*timeout) * time.Second,
		TitleSuffix:  titleSuffix,
		HistoryStore: histStore,
	})
	if err != nil {
		if errors.Is(err, digest.ErrNoSources) || errors.Is(err, digest.ErrNoResults) || errors.Is(err, digest.ErrNoItems) {
			slog.Warn("流程提前结束", "reason", err)
			os.Exit(0)
		}
		slog.Error("晨报生成失败", "error", err)
		os.Exit(1)
	}

	// 4. 打印信源抓取健康度概览面板
	printHealthSummary(result.SourceReports)

	// 5. 输出处理：测试模式写出文件 / 生产模式多渠道分发
	if *testMode {
		if err := os.WriteFile("output.html", []byte(result.HTML), 0644); err != nil {
			slog.Error("写出测试 HTML 文件失败", "error", err)
			os.Exit(1)
		}
		slog.Info("测试文件已生成: output.html",
			"total", result.TotalCount,
			"sources", result.SourceCount,
			"failed_sources", len(result.FailedSources),
		)

		mdTarget := *outputMD
		if mdTarget == "" {
			mdTarget = "output.md"
		}
		if err := os.WriteFile(mdTarget, []byte(result.Markdown), 0644); err != nil {
			slog.Warn("写出 Markdown 测试文件失败", "path", mdTarget, "error", err)
		} else {
			slog.Info("Markdown 文件已生成", "path", mdTarget)
		}
	} else {
		// 生产模式：若有指定 --output-md，也同步归档一份
		if *outputMD != "" {
			_ = os.WriteFile(*outputMD, []byte(result.Markdown), 0644)
			slog.Info("Markdown 已归档", "path", *outputMD)
		}

		subject := fmt.Sprintf("NewsPocket 每日简报 - %s", timeutil.TodayString())
		dispatchedAny := false

		// A. 邮件推送（若环境变量配置了 EMAIL_USER）
		if os.Getenv("EMAIL_USER") != "" {
			mailCfg, err := mailer.LoadFromEnv()
			if err != nil {
				slog.Error("邮件配置错误", "error", err)
			} else {
				if err := mailer.SendHTML(mailCfg, subject, result.HTML); err != nil {
					slog.Error("邮件发送失败", "error", err)
				} else {
					slog.Info("📧 邮件发送成功", "to", mailCfg.Recipients)
					dispatchedAny = true
				}
			}
		}

		// B. 多通道 Webhook 推送（企业微信 / 飞书 / 钉钉 / Telegram / 通用 Webhook）
		p := pusher.New(pusher.LoadFromEnv())
		if p.HasActiveChannels() {
			slog.Info("开始推送至 Webhook 即时通信渠道...")
			pushResults := p.PushAll(context.Background(), pusher.Payload{
				Title:      subject,
				Markdown:   result.Markdown,
				TotalCount: result.TotalCount,
				AIDigest:   result.AIDigest,
			})
			for ch, pErr := range pushResults {
				if pErr == nil {
					dispatchedAny = true
					slog.Info("🤖 即时消息通道推送成功", "channel", ch)
				}
			}
		}

		if !dispatchedAny {
			slog.Warn("未配置任何有效的外发渠道 (EMAIL_USER / WECOM_WEBHOOK / FEISHU_WEBHOOK / DINGTALK_WEBHOOK / TELEGRAM_BOT_TOKEN)。若需调试请使用 --test 选项。")
		}
	}

	slog.Info("=== NewsPocket 运行完成 ===")
}

// printHealthSummary 格式化输出抓取健康度概览
func printHealthSummary(reports []digest.SourceReport) {
	fmt.Println("\n" + strings.Repeat("=", 68))
	fmt.Printf("%-24s | %-10s | %-8s | %-12s\n", "新闻源名称", "耗时", "收录数", "状态")
	fmt.Println(strings.Repeat("-", 68))

	for _, rep := range reports {
		status := "✅ OK"
		if rep.Error != "" {
			status = "❌ FAIL"
		}
		// 截短名称以防终端溢出
		name := rep.Name
		if len([]rune(name)) > 14 {
			name = string([]rune(name)[:13]) + "…"
		}
		durStr := fmt.Sprintf("%dms", rep.Duration.Milliseconds())
		fmt.Printf("%-22s | %-10s | %-8d | %-12s\n", name, durStr, rep.Items, status)
	}
	fmt.Println(strings.Repeat("=", 68) + "\n")
}
