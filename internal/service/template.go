package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"text/template"

	"github.com/rightcapital/notification-service/internal/model"
)

// TemplateService 模板转换服务
type TemplateService struct{}

// NewTemplateService 创建模板服务实例
func NewTemplateService() *TemplateService {
	return &TemplateService{}
}

// RenderHeaders 渲染请求头
func (s *TemplateService) RenderHeaders(config *model.TargetSystemConfig, data map[string]interface{}) (map[string]string, error) {
	if config.Headers == nil {
		return make(map[string]string), nil
	}

	var headerTemplate map[string]string
	if err := json.Unmarshal(config.Headers, &headerTemplate); err != nil {
		return nil, fmt.Errorf("parse header template failed: %w", err)
	}

	result := make(map[string]string, len(headerTemplate))
	for key, tplStr := range headerTemplate {
		rendered, err := s.renderTemplate(tplStr, data)
		if err != nil {
			return nil, fmt.Errorf("render header %s failed: %w", key, err)
		}
		result[key] = rendered
	}

	return result, nil
}

// RenderBody 渲染请求体
func (s *TemplateService) RenderBody(config *model.TargetSystemConfig, data map[string]interface{}) (interface{}, error) {
	if config.BodyTemplate == nil {
		return data["content"], nil
	}

	var bodyTemplate interface{}
	if err := json.Unmarshal(config.BodyTemplate, &bodyTemplate); err != nil {
		return nil, fmt.Errorf("parse body template failed: %w", err)
	}

	return s.renderValue(bodyTemplate, data)
}

// renderValue 递归渲染模板值
func (s *TemplateService) renderValue(value interface{}, data map[string]interface{}) (interface{}, error) {
	switch v := value.(type) {
	case string:
		return s.renderTemplate(v, data)
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for key, val := range v {
			rendered, err := s.renderValue(val, data)
			if err != nil {
				return nil, err
			}
			result[key] = rendered
		}
		return result, nil
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			rendered, err := s.renderValue(val, data)
			if err != nil {
				return nil, err
			}
			result[i] = rendered
		}
		return result, nil
	default:
		return v, nil
	}
}

// renderTemplate 渲染单个模板字符串
func (s *TemplateService) renderTemplate(tplStr string, data map[string]interface{}) (string, error) {
	tpl, err := template.New("tpl").Parse(tplStr)
	if err != nil {
		return "", fmt.Errorf("parse template failed: %w", err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template failed: %w", err)
	}

	return buf.String(), nil
}
