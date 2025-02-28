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

	//认证路由
	auth := router.Group("/user")
	auth.Use(middleware.AuthMiddleware())
	{
		auth.POST("/video/upload", handlers.UploadVideo)
		auth.POST("/work/:id/:file_name/init", handlers.InitModel)
		auth.GET("/video/", handlers.ShowVideo)
		auth.GET("/work/get", handlers.GetWorkPath)
		auth.GET("/work/view", handlers.GetViwer)
		auth.DELETE("/:id/delete", handlers.DeleteUser)
	}

	router.Static("/uploads", config.Conf.UploadPath)
	router.Static("/output", config.Conf.OutputPath)
	return router
}
