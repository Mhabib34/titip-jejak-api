package usecase

import (
	"context"
	"titip-jejak-api/internal/dto"

	"github.com/google/uuid"
)

type NotificationUsecase interface {
	GetAll(ctx context.Context, userID uuid.UUID, query dto.GetNotificationsQuery) (*dto.NotificationListData, error)
	MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error
	MarkAllAsRead(ctx context.Context, userID uuid.UUID) error
}
