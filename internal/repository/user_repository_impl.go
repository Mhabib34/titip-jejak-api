package repository

import (
	"context"
	"titip-jejak-api/internal/exception"
	"titip-jejak-api/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepositoryImpl struct {
	DB *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &UserRepositoryImpl{
		DB: db,
	}
}

func (u *UserRepositoryImpl) Create(ctx context.Context, user *model.User) (*model.User, error) {
	err := u.DB.WithContext(ctx).Create(user).Error
	exception.PanicIfError(err)
	return user, nil
}

func (u *UserRepositoryImpl) FIndByEmail(ctx context.Context, email string) (*model.User, error) {
	var user model.User
	err := u.DB.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (u *UserRepositoryImpl) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	err := u.DB.WithContext(ctx).Where("id = ?", id).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}
