package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/rightcapital/notification-service/internal/dao"
	"github.com/rightcapital/notification-service/internal/model"
)

// DeliveryService 投递服务
type DeliveryService struct {
	notificationDAO *dao.NotificationDAO
	configDAO       *dao.TargetConfigDAO
	templateService *TemplateService
	httpClient      *http.Client
}

// NewDeliveryService 创建投递服务实例
func NewDeliveryService() *DeliveryService {
	return &DeliveryService{
		notificationDAO: dao.NewNotificationDAO(),
		configDAO:       dao.NewTargetConfigDAO(),
		templateService: NewTemplateService(),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Deliver 执行投递
func (s *DeliveryService) Deliver(notification *model.Notification) error {
	// 获取目标系统配置
	config, err := s.configDAO.GetBySystemCode(notification.TargetSystem)
	if err != nil {
		zap.L().Error("Get target system config failed", zap.String("system", notification.TargetSystem), zap.Error(err))
		return s.handleFailure(notification, fmt.Errorf("get target config failed: %w", err))
	}
	if config == nil {
		err := fmt.Errorf("target system %s not found or inactive", notification.TargetSystem)
		zap.L().Error("Target system not found", zap.String("system", notification.TargetSystem))
		return s.notificationDAO.MarkAsDead(notification.ID, err.Error())
	}

	// 准备模板数据
	var content map[string]interface{}
	if err := json.Unmarshal(notification.Content, &content); err != nil {
		zap.L().Error("Parse notification content failed", zap.Error(err))
		return s.handleFailure(notification, fmt.Errorf("parse content failed: %w", err))
	}

	templateData := map[string]interface{}{
		"biz_id":       notification.BizID,
		"biz_type":     notification.BizType,
		"target_system": notification.TargetSystem,
		"content":      content,
		"created_at":   notification.CreatedAt,
	}

	// 渲染请求头
	headers, err := s.templateService.RenderHeaders(config, templateData)
	if err != nil {
		zap.L().Error("Render headers failed", zap.Error(err))
		return s.handleFailure(notification, fmt.Errorf("render headers failed: %w", err))
	}

	// 渲染请求体
	body, err := s.templateService.RenderBody(config, templateData)
	if err != nil {
		zap.L().Error("Render body failed", zap.Error(err))
		return s.handleFailure(notification, fmt.Errorf("render body failed: %w", err))
	}

	// 序列化请求体
	bodyBytes, err := json.Marshal(body)
	if err != nil {
		zap.L().Error("Marshal request body failed", zap.Error(err))
		return s.handleFailure(notification, fmt.Errorf("marshal body failed: %w", err))
	}

	// 创建请求
	req, err := http.NewRequest(config.Method, config.Endpoint, bytes.NewBuffer(bodyBytes))
	if err != nil {
		zap.L().Error("Create HTTP request failed", zap.Error(err))
		return s.handleFailure(notification, fmt.Errorf("create request failed: %w", err))
	}

	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 执行请求
	resp, err := s.httpClient.Do(req)
	if err != nil {
		zap.L().Warn("HTTP request failed", zap.Error(err))
		return s.handleFailure(notification, fmt.Errorf("request failed: %w", err))
	}
	defer resp.Body.Close()

	// 读取响应
	respBody, _ := io.ReadAll(resp.Body)

	// 处理响应
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		// 投递成功
		zap.L().Info("Delivery success",
			zap.String("biz_id", notification.BizID),
			zap.String("target_system", notification.TargetSystem),
			zap.Int("status_code", resp.StatusCode))
		return s.notificationDAO.UpdateStatus(notification.ID, model.StatusSuccess, "")
	}

	// 投递失败
	err = fmt.Errorf("delivery failed with status code %d, response: %s", resp.StatusCode, string(respBody))
	zap.L().Warn("Delivery failed",
		zap.String("biz_id", notification.BizID),
		zap.String("target_system", notification.TargetSystem),
		zap.Int("status_code", resp.StatusCode),
		zap.String("response", string(respBody)))

	// 判断是否需要重试
	if s.shouldRetry(resp.StatusCode) {
		return s.handleFailure(notification, err)
	}

	// 不可重试错误，直接标记为死信
	return s.notificationDAO.MarkAsDead(notification.ID, err.Error())
}

// shouldRetry 判断是否需要重试
func (s *DeliveryService) shouldRetry(statusCode int) bool {
	// 5xx错误、429限流需要重试
	if (statusCode >= 500 && statusCode < 600) || statusCode == 429 {
		return true
	}
	// 其他错误（如400、401、403、404等）不需要重试
	return false
}

// handleFailure 处理投递失败，更新重试信息
func (s *DeliveryService) handleFailure(notification *model.Notification, err error) error {
	newRetryCount := notification.RetryCount + 1

	// 超过最大重试次数，标记为死信
	if newRetryCount >= notification.MaxRetryCount {
		zap.L().Error("Max retry count reached, mark as dead",
			zap.String("biz_id", notification.BizID),
			zap.Int("retry_count", newRetryCount))
		return s.notificationDAO.MarkAsDead(notification.ID, err.Error())
	}

	// 计算下次重试时间（指数退避）
	nextRetryTime := time.Now().Add(time.Duration(1<<newRetryCount) * time.Second)

	// 更新重试信息
	if updateErr := s.notificationDAO.UpdateRetry(notification.ID, newRetryCount, &nextRetryTime, err.Error()); updateErr != nil {
		zap.L().Error("Update retry info failed", zap.Error(updateErr))
		return updateErr
	}

	return err
}
