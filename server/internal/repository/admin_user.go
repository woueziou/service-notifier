package repository

import (
	"context"
	"fmt"

	"woueziou/notifier/internal/model"
	"gorm.io/gorm"
)

type AdminUserRepo struct {
	db *gorm.DB
}

func NewAdminUserRepo(db *gorm.DB) *AdminUserRepo {
	return &AdminUserRepo{db: db}
}

// FindByEmail looks up an admin user by email.
func (r *AdminUserRepo) FindByEmail(ctx context.Context, email string) (*model.AdminUser, error) {
	var user model.AdminUser
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("find admin user by email: %w", err)
	}
	return &user, nil
}

// List returns all admin users ordered by creation time.
func (r *AdminUserRepo) List(ctx context.Context) ([]model.AdminUser, error) {
	var users []model.AdminUser
	err := r.db.WithContext(ctx).Order("created_at ASC").Find(&users).Error
	if err != nil {
		return nil, fmt.Errorf("list admin users: %w", err)
	}
	return users, nil
}

// Create inserts a new admin user.
func (r *AdminUserRepo) Create(ctx context.Context, email string, role model.AdminRole, createdBy *string) (*model.AdminUser, error) {
	user := &model.AdminUser{
		Email:     email,
		Role:      role,
		CreatedBy: createdBy,
	}
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return nil, fmt.Errorf("create admin user: %w", err)
	}
	return user, nil
}

// Delete removes an admin user by ID.
func (r *AdminUserRepo) Delete(ctx context.Context, id string) error {
	result := r.db.WithContext(ctx).Delete(&model.AdminUser{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("delete admin user: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("admin user not found")
	}
	return nil
}

// Count returns the total number of admin users.
func (r *AdminUserRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.AdminUser{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count admin users: %w", err)
	}
	return count, nil
}
