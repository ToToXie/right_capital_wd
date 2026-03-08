package dao

import (
	"time"

	"gorm.io/gorm"

	"github.com/rightcapital/notification-service/internal/model"
)

// NotificationDAO 通知数据访问对象
type NotificationDAO struct {
	db *gorm.DB
}

// NewNotificationDAO 创建通知DAO实例
func NewNotificationDAO() *NotificationDAO {
	return &NotificationDAO{db: GetDB()}
}

// Create 创建通知
func (d *NotificationDAO) Create(notification *model.Notification) error {
	return d.db.Create(notification).Error
}

// GetByBizID 根据业务ID查询通知
func (d *NotificationDAO) GetByBizID(bizID string) (*model.Notification, error) {
	var notification model.Notification
	err := d.db.Where("biz_id = ?", bizID).First(&notification).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &notification, err
}

// ListPendingRetry 查询待重试的通知
func (d *NotificationDAO) ListPendingRetry(batchSize int) ([]*model.Notification, error) {
	var notifications []*model.Notification
	now := time.Now()
	err := d.db.Where("status IN (?) AND next_retry_time <= ?",
		[]model.NotificationStatus{model.StatusPending, model.StatusFailed},
		now).Limit(batchSize).Find(&notifications).Error
	return notifications, err
}

// UpdateStatus 更新通知状态
func (d *NotificationDAO) UpdateStatus(id uint64, status model.NotificationStatus, lastError string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if lastError != "" {
		updates["last_error"] = lastError
	}
	return d.db.Model(&model.Notification{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateRetry 更新重试信息
func (d *NotificationDAO) UpdateRetry(id uint64, retryCount int, nextRetryTime *time.Time, lastError string) error {
	updates := map[string]interface{}{
		"retry_count":     retryCount,
		"next_retry_time": nextRetryTime,
		"last_error":      lastError,
		"status":          model.StatusFailed,
	}
	return d.db.Model(&model.Notification{}).Where("id = ?", id).Updates(updates).Error
}

// MarkAsDead 标记为死信
func (d *NotificationDAO) MarkAsDead(id uint64, lastError string) error {
	updates := map[string]interface{}{
		"status":     model.StatusDead,
		"last_error": lastError,
	}
	return d.db.Model(&model.Notification{}).Where("id = ?", id).Updates(updates).Error
}
