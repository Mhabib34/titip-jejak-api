package repository

import (
	"context"
	"titip-jejak-api/internal/dto"
	"titip-jejak-api/internal/model"

	"github.com/google/uuid"
)

type NotificationRepository interface {
	Create(ctx context.Context, notification *model.Notification) error
	FindByUserID(ctx context.Context, userID uuid.UUID, query dto.GetNotificationsQuery) ([]model.Notification, int64, error)
	CountUnread(ctx context.Context, userID uuid.UUID) (int64, error)
	MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
}
