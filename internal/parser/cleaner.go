package parser

import (
	"net/url"
	"strings"
)

// trackingParams 常见跟踪、推广与埋点参数清单（小写匹配）
var trackingParams = map[string]bool{
	// Google Analytics / Universal Tracking
	"utm_source":   true,
	"utm_medium":   true,
	"utm_campaign": true,
	"utm_term":     true,
	"utm_content":  true,
	"utm_id":       true,
	"utm_reader":   true,

	// 社交网络与各大平台追踪
	"spm_id_from": true, // Bilibili
	"from_source": true,
	"from":        true,
	"ref":         true,
	"source":      true,
	"feature":     true,
	"refer_flag":  true, // Weibo
	"ouid":        true,
	"fbclid":      true, // Facebook
	"gclid":       true, // Google Ads
	"_hsenc":      true, // HubSpot
	"_hsmi":       true,
	"mc_cid":      true, // MailChimp
	"mc_eid":      true,
	"zanpid":      true,
}

// CleanURL 清洗链接中的跟踪参数，保留核心有效查询参数，规范化 URL
func CleanURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		// 若无法标准解析，退回到简单字符串修剪
		return rawURL
	}

	// 若无 Query 参数，直接去除末尾多余斜杠或碎片
	if parsed.RawQuery == "" {
		return parsed.String()
	}

	query := parsed.Query()
	hasChanges := false

	for key := range query {
		lowerKey := strings.ToLower(key)
		if trackingParams[lowerKey] || strings.HasPrefix(lowerKey, "utm_") {
			query.Del(key)
			hasChanges = true
		}
	}

	if hasChanges {
		parsed.RawQuery = query.Encode()
	}

	return parsed.String()
}
