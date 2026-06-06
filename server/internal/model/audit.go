package model

import "time"

type AuditLog struct {
	ID         string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	ConsumerID string    `gorm:"index;not null"`
	IP         string    `gorm:"type:varchar(45)"`
	Endpoint   string    `gorm:"type:varchar(255)"`
	Method     string    `gorm:"type:varchar(10)"`
	StatusCode int       `gorm:"type:smallint"`
	JobID      string    `gorm:"type:varchar(255)"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
