package handler

import (
	"testing"

	"github.com/flyasky/notifier/internal/model"
)

func TestValidateStruct_ValidSendRequest(t *testing.T) {
	req := &model.SendRequest{
		To:      []string{"user@example.com"},
		Subject: "Hello",
		Body:    "World",
	}
	if msg := ValidateStruct(req); msg != "" {
		t.Errorf("expected no validation errors, got: %s", msg)
	}
}

func TestValidateStruct_MissingRecipients(t *testing.T) {
	req := &model.SendRequest{
		To:      []string{},
		Subject: "Hello",
		Body:    "World",
	}
	if msg := ValidateStruct(req); msg == "" {
		t.Error("expected validation error for empty recipients")
	}
}

func TestValidateStruct_InvalidEmail(t *testing.T) {
	req := &model.SendRequest{
		To:      []string{"not-an-email"},
		Subject: "Hello",
		Body:    "World",
	}
	if msg := ValidateStruct(req); msg == "" {
		t.Error("expected validation error for invalid email")
	}
}

func TestValidateStruct_MultipleInvalidEmails(t *testing.T) {
	req := &model.SendRequest{
		To:      []string{"valid@example.com", "invalid", "also-invalid"},
		Subject: "Hello",
		Body:    "World",
	}
	msg := ValidateStruct(req)
	if msg == "" {
		t.Error("expected validation error for invalid emails")
	}
	t.Logf("validation message: %s", msg)
}

func TestValidateStruct_ValidConsumerRequest(t *testing.T) {
	req := &model.CreateConsumerRequest{
		Name:        "test-app",
		EmailPrefix: "test-noreply",
	}
	if msg := ValidateStruct(req); msg != "" {
		t.Errorf("expected no validation errors, got: %s", msg)
	}
}

func TestValidateStruct_EmptyConsumerName(t *testing.T) {
	req := &model.CreateConsumerRequest{
		Name:        "",
		EmailPrefix: "test-noreply",
	}
	if msg := ValidateStruct(req); msg == "" {
		t.Error("expected validation error for empty name")
	}
}

func TestValidateStruct_EmptyEmailPrefix(t *testing.T) {
	req := &model.CreateConsumerRequest{
		Name:        "test-app",
		EmailPrefix: "",
	}
	if msg := ValidateStruct(req); msg == "" {
		t.Error("expected validation error for empty email prefix")
	}
}

func TestValidateStruct_Nil(t *testing.T) {
	if msg := ValidateStruct(nil); msg == "" {
		t.Error("expected validation error for nil")
	}
}
