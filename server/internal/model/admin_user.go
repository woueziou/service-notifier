package model

import "time"

type AdminRole string

const (
	RoleSuperAdmin AdminRole = "super_admin"
	RoleAdmin      AdminRole = "admin"
	RoleViewer     AdminRole = "viewer"
)

type AdminUser struct {
	ID        string    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	Email     string    `gorm:"uniqueIndex;not null" json:"email"`
	Role      AdminRole `gorm:"type:varchar(20);not null;default:'viewer'" json:"role"`
	CreatedBy *string   `gorm:"type:uuid" json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (AdminUser) TableName() string {
	return "admin_users"
}
