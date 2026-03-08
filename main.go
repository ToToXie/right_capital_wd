package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/rightcapital/notification-service/config"
	"github.com/rightcapital/notification-service/internal/api/handler"
	"github.com/rightcapital/notification-service/internal/dao"
	"github.com/rightcapital/notification-service/internal/task"
)

func main() {
	// 初始化配置
	config.Init()

	// 初始化日志
	logger, _ := zap.NewProduction()
	defer logger.Sync()
	zap.ReplaceGlobals(logger)

	// 初始化数据库
	if err := dao.Init(); err != nil {
		zap.L().Fatal("Failed to initialize database", zap.Error(err))
	}

	// 初始化Redis
	// TODO: 实现Redis初始化，用于限流和缓存

	// 启动定时任务
	retryTask := task.NewRetryTask()
	retryTask.Start()
	defer retryTask.Stop()

	// 初始化Gin路由
	r := gin.Default()

	// API路由组
	v1 := r.Group("/api/v1")
	{
		notificationHandler := handler.NewNotificationHandler()
		v1.POST("/notifications", notificationHandler.Create)
		v1.GET("/notifications/:biz_id", notificationHandler.GetStatus)
	}

	// 启动服务
	srv := &http.Server{
		Addr:    config.Get().Server.Port,
		Handler: r,
	}

	// 优雅关闭
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			zap.L().Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	zap.L().Info("Shutting down server...")

	// 5秒超时关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		zap.L().Fatal("Server forced to shutdown", zap.Error(err))
	}

	zap.L().Info("Server exiting")
}
