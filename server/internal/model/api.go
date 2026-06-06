package model

// --- Send Email ---

type SendRequest struct {
	To      []string `json:"to" validate:"required,min=1,dive,email"`
	Subject string   `json:"subject" validate:"required,max=998"`
	Body    string   `json:"body" validate:"required"`
}

type SendResponse struct {
	JobID string `json:"job_id"`
	Status string `json:"status"`
}

// --- Consumer ---

type CreateConsumerRequest struct {
	Name        string `json:"name" validate:"required,min=2,max=100"`
	EmailPrefix string `json:"email_prefix" validate:"required,min=2,max=100"`
}

type CreateConsumerResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	EmailPrefix string `json:"email_prefix"`
	SenderEmail string `json:"sender_email"`
	APIKey      string `json:"api_key"` // shown once
}

// --- Error ---

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
