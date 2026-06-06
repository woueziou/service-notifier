package model

import "time"

type Consumer struct {
	ID                 string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Name               string    `gorm:"uniqueIndex;not null" json:"name"`
	EmailPrefix        string    `gorm:"not null" json:"email_prefix"`
	SenderEmail        string    `gorm:"not null" json:"sender_email"`
	APIKeyHash         string    `gorm:"not null" json:"-"`
	HMACSecretEncrypted string   `gorm:"type:text" json:"-"`
	Active             bool      `gorm:"default:true" json:"active"`
	Suspended          bool      `gorm:"default:false" json:"suspended"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	Jobs               []Job     `gorm:"foreignKey:ConsumerID" json:"jobs,omitempty"`
}

func (Consumer) TableName() string {
	return "consumers"
}
