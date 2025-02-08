package router

import (
	"myapp/config"
	"myapp/handlers"
	"myapp/middleware"

	"github.com/gin-gonic/gin"
)

func RouterConfig() *gin.Engine {
	router := gin.Default()

	//公共路由
	router.POST("/register", handlers.Register)
	router.POST("/login", handlers.Login)

	auth := router.Group("/user")
	auth.Use(middleware.AuthMiddleware())
	{
		auth.POST("/upload")
		auth.GET("/video/:id")
		auth.GET("/work/:id")
	}

	router.Static("/uploads", config.Conf.UploadPath)
	router.Static("/output", config.Conf.OutputPath)
	return router
}
