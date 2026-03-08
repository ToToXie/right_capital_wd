-- 创建数据库
CREATE DATABASE IF NOT EXISTS notification_service DEFAULT CHARACTER SET utf8mb4 DEFAULT COLLATE utf8mb4_unicode_ci;

USE notification_service;

-- 通知主表
CREATE TABLE `notification` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `biz_id` varchar(64) NOT NULL COMMENT '业务唯一ID，幂等用',
  `biz_type` varchar(32) NOT NULL COMMENT '业务类型',
  `target_system` varchar(32) NOT NULL COMMENT '目标系统编码',
  `content` json NOT NULL COMMENT '通知内容',
  `status` tinyint NOT NULL DEFAULT 0 COMMENT '0待投递 1成功 2失败 3死信',
  `retry_count` int NOT NULL DEFAULT 0,
  `max_retry_count` int NOT NULL DEFAULT 3,
  `next_retry_time` datetime DEFAULT NULL,
  `last_error` varchar(1024) DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_biz_id` (`biz_id`),
  KEY `idx_status_next_retry` (`status`, `next_retry_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 目标系统配置表
CREATE TABLE `target_system_config` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `system_code` varchar(32) NOT NULL UNIQUE COMMENT '系统编码',
  `system_name` varchar(64) NOT NULL,
  `endpoint` varchar(255) NOT NULL COMMENT 'API地址',
  `method` varchar(10) NOT NULL DEFAULT 'POST',
  `headers` json DEFAULT NULL COMMENT '请求头模板',
  `body_template` json DEFAULT NULL COMMENT '请求体模板',
  `retry_strategy` json DEFAULT NULL,
  `timeout` int NOT NULL DEFAULT 5000 COMMENT '超时毫秒',
  `rate_limit` int NOT NULL DEFAULT 100 COMMENT '每秒限流',
  `is_active` tinyint NOT NULL DEFAULT 1,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 插入示例配置
INSERT INTO `target_system_config` (`system_code`, `system_name`, `endpoint`, `method`, `headers`, `body_template`, `timeout`, `rate_limit`)
VALUES (
  'example_system',
  '示例第三方系统',
  'https://api.example.com/webhook',
  'POST',
  '{"Content-Type": "application/json", "Authorization": "Bearer ${secret}"}',
  '{"event_type": "${biz_type}", "event_id": "${biz_id}", "data": ${content}}',
  5000,
  100
);
