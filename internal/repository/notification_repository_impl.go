package repository

import (
	"context"
	"titip-jejak-api/internal/dto"
	"titip-jejak-api/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationRepositoryImpl struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &NotificationRepositoryImpl{db: db}
}

// Create — simpan notifikasi baru. Dipanggil oleh MatchWorker.
func (r *NotificationRepositoryImpl) Create(ctx context.Context, notification *model.Notification) error {
	return r.db.WithContext(ctx).Create(notification).Error
}

// FindByUserID — ambil notifikasi milik userID dengan filter & pagination.
func (r *NotificationRepositoryImpl) FindByUserID(ctx context.Context, userID uuid.UUID, query dto.GetNotificationsQuery) ([]model.Notification, int64, error) {
	page := query.Page
	if page < 1 {
		page = 1
	}
	limit := query.Limit
	if limit < 1 {
		limit = 10
	}

	var notifications []model.Notification
	var total int64

	db := r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ?", userID)

	// Filter is_read jika dikirim
	if query.IsRead != nil {
		db = db.Where("is_read = ?", *query.IsRead)
	}

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if err := db.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error; err != nil {
		return nil, 0, err
	}

	return notifications, total, nil
}

// CountUnread — hitung jumlah notifikasi belum dibaca (untuk badge).
func (r *NotificationRepositoryImpl) CountUnread(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Count(&count).Error
	return count, err
}

// MarkAsRead — tandai satu notifikasi sebagai sudah dibaca.
// Menggunakan userID untuk memastikan notifikasi milik user tersebut.
func (r *NotificationRepositoryImpl) MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("id = ? AND user_id = ?", id, userID).
		Update("is_read", true)

	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// MarkAllAsRead — tandai semua notifikasi milik userID sebagai sudah dibaca.
func (r *NotificationRepositoryImpl) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.Notification{}).
		Where("user_id = ? AND is_read = false", userID).
		Update("is_read", true).Error
}
