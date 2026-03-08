package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/rightcapital/notification-service/internal/dao"
	"github.com/rightcapital/notification-service/internal/model"
	"github.com/rightcapital/notification-service/internal/service"
)

// NotificationHandler 通知处理器
type NotificationHandler struct {
	notificationDAO *dao.NotificationDAO
	deliveryService *service.DeliveryService
}

// NewNotificationHandler 创建通知处理器实例
func NewNotificationHandler() *NotificationHandler {
	return &NotificationHandler{
		notificationDAO: dao.NewNotificationDAO(),
		deliveryService: service.NewDeliveryService(),
	}
}

// CreateNotificationRequest 创建通知请求
type CreateNotificationRequest struct {
	BizID        string          `json:"biz_id" binding:"required"`
	BizType      string          `json:"biz_type" binding:"required"`
	TargetSystem string          `json:"target_system" binding:"required"`
	Content      json.RawMessage `json:"content" binding:"required"`
}

// Create 创建通知
func (h *NotificationHandler) Create(c *gin.Context) {
	var req CreateNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "Invalid request parameters", "error": err.Error()})
		return
	}

	// 幂等校验
	exist, err := h.notificationDAO.GetByBizID(req.BizID)
	if err != nil {
		zap.L().Error("Check notification exist failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Internal server error"})
		return
	}
	if exist != nil {
		c.JSON(http.StatusOK, gin.H{
			"code": 200,
			"message": "Notification already exists",
			"data": gin.H{
				"biz_id": exist.BizID,
				"status": exist.Status,
			},
		})
		return
	}

	// 创建通知
	notification := &model.Notification{
		BizID:        req.BizID,
		BizType:      req.BizType,
		TargetSystem: req.TargetSystem,
		Content:      model.JSON(req.Content),
		Status:       model.StatusPending,
	}

	if err := h.notificationDAO.Create(notification); err != nil {
		zap.L().Error("Create notification failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Create notification failed"})
		return
	}

	// 异步首次投递
	go func() {
		if err := h.deliveryService.Deliver(notification); err != nil {
			zap.L().Warn("First delivery failed, will retry later", zap.String("biz_id", notification.BizID), zap.Error(err))
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"message": "Notification accepted",
		"data": gin.H{
			"biz_id": notification.BizID,
			"id":     notification.ID,
		},
	})
}

// GetStatus 查询通知状态
func (h *NotificationHandler) GetStatus(c *gin.Context) {
	bizID := c.Param("biz_id")
	if bizID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": "biz_id is required"})
		return
	}

	notification, err := h.notificationDAO.GetByBizID(bizID)
	if err != nil {
		zap.L().Error("Get notification status failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Internal server error"})
		return
	}
	if notification == nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "message": "Notification not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"message": "Success",
		"data": gin.H{
			"biz_id":         notification.BizID,
			"biz_type":       notification.BizType,
			"target_system":  notification.TargetSystem,
			"status":         notification.Status,
			"retry_count":    notification.RetryCount,
			"max_retry_count": notification.MaxRetryCount,
			"next_retry_time": notification.NextRetryTime,
			"last_error":     notification.LastError,
			"created_at":     notification.CreatedAt,
			"updated_at":     notification.UpdatedAt,
		},
	})
}
