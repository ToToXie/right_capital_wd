package task

import (
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"github.com/rightcapital/notification-service/config"
	"github.com/rightcapital/notification-service/internal/dao"
	"github.com/rightcapital/notification-service/internal/service"
)

// RetryTask 重试任务
type RetryTask struct {
	cron            *cron.Cron
	notificationDAO *dao.NotificationDAO
	deliveryService *service.DeliveryService
}

// NewRetryTask 创建重试任务实例
func NewRetryTask() *RetryTask {
	return &RetryTask{
		cron:            cron.New(cron.WithSeconds()),
		notificationDAO: dao.NewNotificationDAO(),
		deliveryService: service.NewDeliveryService(),
	}
}

// Start 启动重试任务
func (t *RetryTask) Start() {
	cfg := config.Get().Retry
	_, err := t.cron.AddFunc(cfg.Cron, func() {
		t.runRetry(cfg.BatchSize)
	})
	if err != nil {
		zap.L().Fatal("Add retry cron job failed", zap.Error(err))
	}

	t.cron.Start()
	zap.L().Info("Retry task started", zap.String("cron", cfg.Cron))
}

// Stop 停止重试任务
func (t *RetryTask) Stop() {
	t.cron.Stop()
	zap.L().Info("Retry task stopped")
}

// runRetry 执行重试逻辑
func (t *RetryTask) runRetry(batchSize int) {
	zap.L().Debug("Start retry task run")

	// 查询待重试的通知
	notifications, err := t.notificationDAO.ListPendingRetry(batchSize)
	if err != nil {
		zap.L().Error("List pending retry notifications failed", zap.Error(err))
		return
	}

	if len(notifications) == 0 {
		zap.L().Debug("No pending retry notifications")
		return
	}

	zap.L().Info("Found pending retry notifications", zap.Int("count", len(notifications)))

	// 并发执行重试
	for _, notification := range notifications {
		go func(n *model.Notification) {
			if err := t.deliveryService.Deliver(n); err != nil {
				zap.L().Warn("Retry delivery failed",
					zap.String("biz_id", n.BizID),
					zap.Int("retry_count", n.RetryCount),
					zap.Error(err))
			} else {
				zap.L().Info("Retry delivery success",
					zap.String("biz_id", n.BizID),
					zap.Int("retry_count", n.RetryCount))
			}
		}(notification)
	}

	zap.L().Debug("Retry task run completed")
}
