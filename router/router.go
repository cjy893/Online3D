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
	router.PUT("/register", handlers.Register)
	router.PUT("/login", handlers.Login)

	//认证路由
	auth := router.Group("/user")
	auth.Use(middleware.AuthMiddleware())
	{
		auth.PUT("/video/upload", handlers.UploadVideo)
		auth.PUT("/work/init", handlers.InitModel)
		auth.GET("/video/get/:id", handlers.GetVideo)
		auth.GET("/work/get", handlers.GetWorkPath)
		auth.GET("/work/view", handlers.GetViwer)
		auth.DELETE("/delete", handlers.DeleteUser)
	}

	router.Static("/uploads", config.Conf.UploadPath)
	router.Static("/output", config.Conf.OutputPath)
	return router
}
