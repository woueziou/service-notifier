package engine

import "context"

type EmailMessage struct {
	From    string
	To      []string
	Subject string
	Body    string // HTML body
}

// EmailEngine defines the interface for sending emails.
// Only SMTP is implemented initially; SES/SendGrid can be added later.
type EmailEngine interface {
	Send(ctx context.Context, msg *EmailMessage) error
}
