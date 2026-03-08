package dao

import (
	"gorm.io/gorm"

	"github.com/rightcapital/notification-service/internal/model"
)

// TargetConfigDAO 目标系统配置数据访问对象
type TargetConfigDAO struct {
	db *gorm.DB
}

// NewTargetConfigDAO 创建目标系统配置DAO实例
func NewTargetConfigDAO() *TargetConfigDAO {
	return &TargetConfigDAO{db: GetDB()}
}

// GetBySystemCode 根据系统编码查询配置
func (d *TargetConfigDAO) GetBySystemCode(systemCode string) (*model.TargetSystemConfig, error) {
	var config model.TargetSystemConfig
	err := d.db.Where("system_code = ? AND is_active = 1", systemCode).First(&config).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &config, err
}

// ListAllActive 查询所有激活的配置
func (d *TargetConfigDAO) ListAllActive() ([]*model.TargetSystemConfig, error) {
	var configs []*model.TargetSystemConfig
	err := d.db.Where("is_active = 1").Find(&configs).Error
	return configs, err
}
