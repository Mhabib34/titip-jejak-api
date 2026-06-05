package usecase

import (
	"context"
	"titip-jejak-api/internal/dto"
	"titip-jejak-api/internal/exception"
	"titip-jejak-api/internal/helper"
	"titip-jejak-api/internal/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationUsecaseImpl struct {
	repo repository.NotificationRepository
}

func NewNotificationUsecase(repo repository.NotificationRepository) NotificationUsecase {
	return &NotificationUsecaseImpl{repo: repo}
}

func (u *NotificationUsecaseImpl) GetAll(ctx context.Context, userID uuid.UUID, query dto.GetNotificationsQuery) (*dto.NotificationListData, error) {
	notifications, total, err := u.repo.FindByUserID(ctx, userID, query)
	if err != nil {
		return nil, err
	}

	unreadCount, err := u.repo.CountUnread(ctx, userID)
	if err != nil {
		return nil, err
	}

	page := query.Page
	if page < 1 {
		page = 1
	}
	limit := query.Limit
	if limit < 1 {
		limit = 10
	}

	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return &dto.NotificationListData{
		Notifications: helper.ToNotificationResponseList(notifications),
		UnreadCount:   unreadCount,
		Meta: dto.Pagination{
			Page:       page,
			Limit:      limit,
			Total:      int(total),
			TotalPages: totalPages,
		},
	}, nil
}

func (u *NotificationUsecaseImpl) MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	err := u.repo.MarkAsRead(ctx, id, userID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return exception.NewNotFoundError("notifikasi tidak ditemukan")
		}
		return err
	}
	return nil
}

func (u *NotificationUsecaseImpl) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	return u.repo.MarkAllAsRead(ctx, userID)
}
