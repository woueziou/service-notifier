package model

import "time"

type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusDelivered JobStatus = "delivered"
	JobStatusFailed    JobStatus = "failed"
	JobStatusBounced   JobStatus = "bounced"
)

type Job struct {
	ID         string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	ConsumerID string    `gorm:"index;not null" json:"consumer_id"`
	Status     JobStatus `gorm:"type:varchar(20);default:pending" json:"status"`
	To         string    `gorm:"type:text;not null" json:"to"`
	Subject    string    `gorm:"type:varchar(998)" json:"subject"`
	Body       string    `gorm:"type:text" json:"body,omitempty"`
	Error      string    `gorm:"type:text" json:"error,omitempty"`
	DeliveredAt *time.Time `json:"delivered_at,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (Job) TableName() string {
	return "jobs"
}
