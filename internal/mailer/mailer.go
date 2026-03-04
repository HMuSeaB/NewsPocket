// Package mailer 负责 SMTP SSL 邮件发送。
package mailer

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"
	"net/smtp"
	"os"
	"strings"
)

// Config 邮件配置，从环境变量读取
type Config struct {
	Host       string
	Port       string
	User       string
	Password   string
	Recipients []string
}

// LoadFromEnv 从环境变量读取邮件配置
func LoadFromEnv() (*Config, error) {
	user := os.Getenv("EMAIL_USER")
	pass := os.Getenv("EMAIL_PASS")
	host := os.Getenv("EMAIL_HOST")
	port := os.Getenv("EMAIL_PORT")

	if user == "" || pass == "" {
		return nil, fmt.Errorf("缺失环境变量 EMAIL_USER 或 EMAIL_PASS")
	}

	if host == "" {
		host = "smtp.qq.com"
	}
	if port == "" {
		port = "465"
	}

	// 解析收件人
	recipientEnv := os.Getenv("EMAIL_TO")
	var recipients []string
	if recipientEnv != "" {
		for _, r := range strings.Split(recipientEnv, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				recipients = append(recipients, r)
			}
		}
	}
	if len(recipients) == 0 {
		recipients = []string{user}
	}

	return &Config{
		Host:       host,
		Port:       port,
		User:       user,
		Password:   pass,
		Recipients: recipients,
	}, nil
}

// SendHTML 发送 HTML 邮件
func SendHTML(cfg *Config, subject, htmlContent string) error {
	slog.Info("连接 SMTP 服务器", "host", cfg.Host, "port", cfg.Port)

	addr := net.JoinHostPort(cfg.Host, cfg.Port)

	// 构建邮件内容 (MIME)
	header := fmt.Sprintf("From: NewsPocket <%s>\r\n", cfg.User)
	header += fmt.Sprintf("To: %s\r\n", strings.Join(cfg.Recipients, ","))
	header += fmt.Sprintf("Subject: %s\r\n", subject)
	header += "MIME-Version: 1.0\r\n"
	header += "Content-Type: text/html; charset=UTF-8\r\n"
	header += "\r\n"

	message := []byte(header + htmlContent)

	// SSL/TLS 连接
	tlsConfig := &tls.Config{
		ServerName: cfg.Host,
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("TLS 连接失败: %w", err)
	}

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("SMTP 客户端创建失败: %w", err)
	}
	defer client.Close()

	// 认证
	auth := smtp.PlainAuth("", cfg.User, cfg.Password, cfg.Host)
	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP 认证失败: %w", err)
	}

	// 发送
	if err := client.Mail(cfg.User); err != nil {
		return fmt.Errorf("MAIL FROM 失败: %w", err)
	}

	for _, rcpt := range cfg.Recipients {
		if err := client.Rcpt(rcpt); err != nil {
			return fmt.Errorf("RCPT TO (%s) 失败: %w", rcpt, err)
		}
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA 命令失败: %w", err)
	}

	if _, err := w.Write(message); err != nil {
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}

	if err := w.Close(); err != nil {
		return fmt.Errorf("关闭数据流失败: %w", err)
	}

	slog.Info("邮件发送成功", "recipients", cfg.Recipients)
	return nil
}
