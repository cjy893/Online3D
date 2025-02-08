package main

import (
	"myapp/config"
	"myapp/router"
	"os"
	"os/signal"
	"syscall"

	"github.com/sirupsen/logrus"
)

func main() {
	config.LoadConfig()

	// 校验必要配置
	if config.Conf.ServerPort == "" {
		logrus.Fatal("服务端口配置缺失")
	}

	// 初始化数据库连接
	config.ConnectDB()
	defer func() {
		if config.Conf.DB != nil {
			sqlDB, err := config.Conf.DB.DB()
			if err != nil {
				logrus.Errorf("fail to get the underlying database connection: %v", err)
				return
			}
			if err := sqlDB.Close(); err != nil {
				logrus.Errorf("fail to close the database: %v", err)
			}
		}
	}()

	// 初始化路由
	router := router.RouterConfig()
	serverAddress := ":" + config.Conf.ServerPort

	// 使用错误通道处理服务启动失败
	errChan := make(chan error)
	go func() {
		if err := router.Run(serverAddress); err != nil {
			logrus.Errorf("fail to run the server: %v", err)
			errChan <- err
		}
	}()

	// 信号监听
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 等待终止信号或启动错误
	select {
	case <-quit:
		logrus.Info("closing server...")
	case err := <-errChan:
		logrus.Errorf("server shut down with error: %v", err)
	}

	logrus.Info("server exit")
}
