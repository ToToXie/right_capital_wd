package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"
)

// NotificationStatus 通知状态枚举
type NotificationStatus int

const (
	StatusPending NotificationStatus = 0 // 待投递
	StatusSuccess NotificationStatus = 1 // 投递成功
	StatusFailed  NotificationStatus = 2 // 投递失败
	StatusDead    NotificationStatus = 3 // 死信
)

// Notification 通知主表模型
type Notification struct {
	ID             uint64                `json:"id" gorm:"primaryKey;autoIncrement"`
	BizID          string                `json:"biz_id" gorm:"uniqueIndex;size:64;not null"`
	BizType        string                `json:"biz_type" gorm:"size:32;not null"`
	TargetSystem   string                `json:"target_system" gorm:"size:32;not null"`
	Content        JSON                  `json:"content" gorm:"type:json;not null"`
	Status         NotificationStatus    `json:"status" gorm:"default:0;not null"`
	RetryCount     int                   `json:"retry_count" gorm:"default:0;not null"`
	MaxRetryCount  int                   `json:"max_retry_count" gorm:"default:3;not null"`
	NextRetryTime  *time.Time            `json:"next_retry_time"`
	LastError      string                `json:"last_error" gorm:"size:1024"`
	CreatedAt      time.Time             `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt      time.Time             `json:"updated_at" gorm:"autoUpdateTime"`
}

// TargetSystemConfig 目标系统配置表模型
type TargetSystemConfig struct {
	ID            uint64 `json:"id" gorm:"primaryKey;autoIncrement"`
	SystemCode    string `json:"system_code" gorm:"uniqueIndex;size:32;not null"`
	SystemName    string `json:"system_name" gorm:"size:64;not null"`
	Endpoint      string `json:"endpoint" gorm:"size:255;not null"`
	Method        string `json:"method" gorm:"size:10;default:'POST';not null"`
	Headers       JSON   `json:"headers" gorm:"type:json"`
	BodyTemplate  JSON   `json:"body_template" gorm:"type:json"`
	RetryStrategy JSON   `json:"retry_strategy" gorm:"type:json"`
	Timeout       int    `json:"timeout" gorm:"default:5000;not null"`
	RateLimit     int    `json:"rate_limit" gorm:"default:100;not null"`
	IsActive      bool   `json:"is_active" gorm:"default:1;not null"`
	CreatedAt     time.Time `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// JSON 自定义JSON类型，用于gorm存储JSON
type JSON json.RawMessage

// Value 实现driver.Valuer接口
func (j JSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return json.RawMessage(j).MarshalJSON()
}

// Scan 实现sql.Scanner接口
func (j *JSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	result := json.RawMessage{}
	if err := json.Unmarshal(bytes, &result); err != nil {
		return err
	}
	*j = JSON(result)
	return nil
}

// MarshalJSON 自定义JSON序列化
func (j JSON) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return json.RawMessage(j).MarshalJSON()
}

// UnmarshalJSON 自定义JSON反序列化
func (j *JSON) UnmarshalJSON(data []byte) error {
	result := json.RawMessage{}
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}
	*j = JSON(result)
	return nil
}
