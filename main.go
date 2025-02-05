package main

import (
	"myapp/config"
	"myapp/handlers"
	"myapp/middleware"

	"github.com/gin-gonic/gin"
)

func main() {
	config.LoadConfig() // 加载环境变量配置
	config.ConnectDB()  // 连接数据库

	r := gin.Default()

	// 公共路由
	r.POST("/register", handlers.Register)
	r.POST("/login", handlers.Login)

	// 需要认证的路由
	auth := r.Group("/")
	auth.Use(middleware.AuthMiddleware())
	{
		auth.POST("/upload", handlers.UploadVideo)
		auth.GET("/video/:id", handlers.GetVideo)
	}

	// 文件访问路由
	r.Static("/uploads", config.Conf.UploadPath)
	r.Static("/output", config.Conf.OutputPath)

	r.Run(":8080")
}
