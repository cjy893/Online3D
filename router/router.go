package router

import (
	"myapp/handlers"
	"myapp/middleware"

	"github.com/gin-gonic/gin"
)

// RouterConfig 配置并返回一个gin路由器实例。
// 该函数负责定义所有的路由规则和中间件使用。
func RouterConfig() *gin.Engine {
	// 创建一个默认的gin路由器实例。
	router := gin.Default()

	// 注册和登录路由，不需要身份验证。
	router.POST("/register", handlers.Register)
	router.POST("/login", handlers.Login)

	// 创建一个带有"/user"前缀的路由组，并应用身份验证中间件。
	auth := router.Group("/user")
	auth.Use(middleware.AuthMiddleware())
	{
		// 需要身份验证的路由规则。
		auth.POST("/video/upload", handlers.UploadVideo)
		auth.POST("/work/init", handlers.InitModel)
		auth.GET("/video/", handlers.ShowVideo)
		auth.GET("/work/", handlers.ShowWork)
		auth.GET("/work/view", handlers.GetViwer)
		auth.GET("/work/path", handlers.GetWork)
		auth.DELETE("/:id/delete", handlers.DeleteUser)
	}

	// 返回配置好的路由器实例。
	return router
}
