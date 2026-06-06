package engine

import (
	"bytes"
	"context"
	"fmt"
	"net/smtp"
	"strings"
)

type SMTPEngine struct {
	Host string
	Port int
	User string
	Pass string
	From string
}

func NewSMTPEngine(host string, port int, user, pass, from string) *SMTPEngine {
	return &SMTPEngine{
		Host: host,
		Port: port,
		User: user,
		Pass: pass,
		From: from,
	}
}

func (e *SMTPEngine) Send(ctx context.Context, msg *EmailMessage) error {
	addr := fmt.Sprintf("%s:%d", e.Host, e.Port)

	// Build MIME message
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("From: %s\r\n", msg.From))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(msg.To, ", ")))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", msg.Subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(msg.Body)

	// Authenticate if credentials provided
	var auth smtp.Auth
	if e.User != "" && e.Pass != "" {
		auth = smtp.PlainAuth("", e.User, e.Pass, e.Host)
	}

	return smtp.SendMail(addr, auth, msg.From, msg.To, buf.Bytes())
}
