package repository

import (
	"context"
	"fmt"

	"woueziou/notifier/internal/auth"
	"woueziou/notifier/internal/model"
	"gorm.io/gorm"
)

type AdminUserRepo struct {
	db *gorm.DB
}

func NewAdminUserRepo(db *gorm.DB) *AdminUserRepo {
	return &AdminUserRepo{db: db}
}

// FindByUsername looks up an admin user by username.
func (r *AdminUserRepo) FindByUsername(ctx context.Context, username string) (*model.AdminUser, error) {
	var user model.AdminUser
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("find admin user: %w", err)
	}
	return &user, nil
}

// Create inserts a new admin user with a bcrypt-hashed password.
func (r *AdminUserRepo) Create(ctx context.Context, username, password string) (*model.AdminUser, error) {
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &model.AdminUser{
		Username:     username,
		PasswordHash: hash,
	}
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, fmt.Errorf("create admin user: %w", err)
	}
	return user, nil
}

// Count returns the total number of admin users.
func (r *AdminUserRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.AdminUser{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count admin users: %w", err)
	}
	return count, nil
}
