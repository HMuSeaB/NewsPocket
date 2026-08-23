// Package timeutil 提供项目内统一的时区与时间格式化工具。
// 项目面向中文新闻源，展示与无时区字符串解析统一采用北京时间。
package timeutil

import (
	"strconv"
	"strings"
	"time"

	// 内嵌 IANA 时区数据库，保证在无系统 tzdata 的环境
	// (如精简容器、交叉编译的 Windows 二进制) 中 LoadLocation 依然可用。
	_ "time/tzdata"
)

// Beijing 北京时区 (CST, UTC+8)
var Beijing = time.FixedZone("CST", 8*3600)

// FormatBeijing 将时间格式化为北京时间字符串 "2006-01-02 15:04"。
// 零值时间返回 "未知时间"。
func FormatBeijing(t time.Time) string {
	if t.IsZero() {
		return "未知时间"
	}
	return t.In(Beijing).Format("2006-01-02 15:04")
}

// TodayString 返回北京时间的今日日期串，如 "2026年08月22日 Saturday"。
func TodayString() string {
	return time.Now().In(Beijing).Format("2006年01月02日 Monday")
}

// ResolveLocation 将时区配置字符串解析为 *time.Location。
// 支持三种写法，解析失败时回退到北京时区：
//   - IANA 名称，如 "Asia/Shanghai"、"UTC"
//   - 固定小时偏移，如 "+8"、"-5"
//   - 空字符串视为北京时区
func ResolveLocation(name string) *time.Location {
	name = strings.TrimSpace(name)
	if name == "" {
		return Beijing
	}

	if loc, err := time.LoadLocation(name); err == nil {
		return loc
	}

	// 尝试按固定小时偏移解析（如 "+8"、"-5"）
	if offset, err := strconv.Atoi(name); err == nil && offset >= -12 && offset <= 14 {
		return time.FixedZone("UTC"+name, offset*3600)
	}

	return Beijing
}
