package engine

import (
	"context"
	"testing"
)

// mockEngine implements EmailEngine for testing.
type mockEngine struct {
	sendFunc func(ctx context.Context, msg *EmailMessage) error
}

func (m *mockEngine) Send(ctx context.Context, msg *EmailMessage) error {
	return m.sendFunc(ctx, msg)
}

func TestEmailEngine_Interface(t *testing.T) {
	// Ensure SMTPEngine satisfies the interface
	var _ EmailEngine = (*SMTPEngine)(nil)
}

func TestMockEngine_Success(t *testing.T) {
	engine := &mockEngine{
		sendFunc: func(_ context.Context, msg *EmailMessage) error {
			if msg.From == "" {
				t.Error("From should not be empty")
			}
			if len(msg.To) == 0 {
				t.Error("To should not be empty")
			}
			return nil
		},
	}

	err := engine.Send(context.Background(), &EmailMessage{
		From:    "test@example.com",
		To:      []string{"user@example.com"},
		Subject: "Test",
		Body:    "<p>Hello</p>",
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestMockEngine_Error(t *testing.T) {
	expected := "smtp connection refused"
	engine := &mockEngine{
		sendFunc: func(_ context.Context, msg *EmailMessage) error {
			return &emailError{msg: expected}
		},
	}

	err := engine.Send(context.Background(), &EmailMessage{
		From: "test@example.com",
		To:   []string{"user@example.com"},
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != expected {
		t.Errorf("expected %q, got %q", expected, err.Error())
	}
}

type emailError struct{ msg string }

func (e *emailError) Error() string { return e.msg }

func TestEmailMessage_Validation(t *testing.T) {
	tests := []struct {
		name    string
		msg     *EmailMessage
		wantErr bool
	}{
		{
			name:    "valid message",
			msg:     &EmailMessage{From: "a@b.com", To: []string{"c@d.com"}, Subject: "Hi", Body: "Hello"},
			wantErr: false,
		},
		{
			name:    "empty from",
			msg:     &EmailMessage{From: "", To: []string{"c@d.com"}, Subject: "Hi", Body: "Hello"},
			wantErr: true,
		},
		{
			name:    "empty to",
			msg:     &EmailMessage{From: "a@b.com", To: []string{}, Subject: "Hi", Body: "Hello"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine := &mockEngine{
				sendFunc: func(_ context.Context, msg *EmailMessage) error {
					if msg.From == "" {
						return &emailError{"from is required"}
					}
					if len(msg.To) == 0 {
						return &emailError{"to is required"}
					}
					return nil
				},
			}
			err := engine.Send(context.Background(), tt.msg)
			if (err != nil) != tt.wantErr {
				t.Errorf("Send() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

// TestEmailMessage_Struct tests the EmailMessage struct fields.
func TestEmailMessage_Struct(t *testing.T) {
	msg := &EmailMessage{
		From:    "sender@example.com",
		To:      []string{"recipient@example.com", "another@example.com"},
		Subject: "Test Subject",
		Body:    "<html><body>Test</body></html>",
	}

	if msg.From != "sender@example.com" {
		t.Errorf("unexpected From: %s", msg.From)
	}
	if len(msg.To) != 2 {
		t.Errorf("expected 2 recipients, got %d", len(msg.To))
	}
	if msg.Subject != "Test Subject" {
		t.Errorf("unexpected Subject: %s", msg.Subject)
	}
	if msg.Body != "<html><body>Test</body></html>" {
		t.Errorf("unexpected Body: %s", msg.Body)
	}
}
