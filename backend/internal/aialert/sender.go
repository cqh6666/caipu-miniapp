package aialert

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"fmt"
	"mime"
	"net"
	"net/smtp"
	"strconv"
	"strings"
)

type SMTPSender struct{}

func NewSMTPSender() *SMTPSender {
	return &SMTPSender{}
}

func (s *SMTPSender) Send(ctx context.Context, request SendRequest) error {
	cfg := request.Config.Normalized()
	if err := cfg.ValidateForSend(); err != nil {
		return err
	}

	address := net.JoinHostPort(cfg.SMTPHost, strconv.Itoa(cfg.SMTPPort))
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return fmt.Errorf("连接 SMTP 服务器失败: %w", err)
	}
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	if cfg.SMTPPort == 465 {
		tlsConn := tls.Client(conn, &tls.Config{
			ServerName: cfg.SMTPHost,
			MinVersion: tls.VersionTLS12,
		})
		if err := tlsConn.Handshake(); err != nil {
			_ = conn.Close()
			return fmt.Errorf("SMTP TLS 握手失败: %w", err)
		}
		conn = tlsConn
	}

	client, err := smtp.NewClient(conn, cfg.SMTPHost)
	if err != nil {
		_ = conn.Close()
		return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
	}
	defer client.Close()

	if cfg.SMTPPort != 465 {
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{
				ServerName: cfg.SMTPHost,
				MinVersion: tls.VersionTLS12,
			}); err != nil {
				return fmt.Errorf("升级 SMTP TLS 失败: %w", err)
			}
		}
	}

	if ok, _ := client.Extension("AUTH"); !ok {
		return errors.New("SMTP 服务器不支持 AUTH")
	}
	if err := client.Auth(smtp.PlainAuth("", cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPHost)); err != nil {
		return fmt.Errorf("SMTP 鉴权失败: %w", err)
	}
	if err := client.Mail(cfg.FromAddress()); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}
	for _, recipient := range cfg.Recipients() {
		if err := client.Rcpt(recipient); err != nil {
			return fmt.Errorf("设置收件人失败: %w", err)
		}
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}
	message := buildMessage(cfg, request.Subject, request.Body)
	if _, err := writer.Write(message); err != nil {
		_ = writer.Close()
		return fmt.Errorf("发送邮件正文失败: %w", err)
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("提交邮件内容失败: %w", err)
	}
	if err := client.Quit(); err != nil {
		return fmt.Errorf("关闭 SMTP 会话失败: %w", err)
	}
	return nil
}

func buildMessage(cfg Config, subject, body string) []byte {
	recipients := cfg.Recipients()
	encodedBody := wrapBase64(base64.StdEncoding.EncodeToString([]byte(strings.TrimSpace(body) + "\n")))
	headers := []string{
		fmt.Sprintf("From: %s", cfg.FromAddress()),
		fmt.Sprintf("To: %s", strings.Join(recipients, ", ")),
		fmt.Sprintf("Subject: %s", mime.QEncoding.Encode("UTF-8", strings.TrimSpace(subject))),
		"MIME-Version: 1.0",
		`Content-Type: text/plain; charset="UTF-8"`,
		"Content-Transfer-Encoding: base64",
		"",
		encodedBody,
	}
	return []byte(strings.Join(headers, "\r\n"))
}

func wrapBase64(value string) string {
	if value == "" {
		return ""
	}
	const lineWidth = 76
	lines := make([]string, 0, len(value)/lineWidth+1)
	for len(value) > lineWidth {
		lines = append(lines, value[:lineWidth])
		value = value[lineWidth:]
	}
	if value != "" {
		lines = append(lines, value)
	}
	return strings.Join(lines, "\r\n")
}
