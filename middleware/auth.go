package middleware

import (
	"myapp/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware 返回一个基于JWT的认证中间件。
// 该中间件用于验证和解析请求头中的认证信息，以确保用户请求的合法性。
// 它主要执行以下操作：
// 1. 从请求头中获取认证令牌。
// 2. 验证令牌的有效性。
// 3. 将解析出的用户ID设置到请求上下文中，供后续处理函数使用。
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取请求头中的Authorization字段值
		tokenString := c.GetHeader("Authorization")
		// 如果请求头中没有Authorization字段，中断请求处理，并返回错误信息
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		// 解析令牌，获取用户ID
		userID, err := utils.ParseToken(tokenString)
		// 如果令牌解析失败，中断请求处理，并返回错误信息
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// 将解析出的用户ID设置到请求上下文中，以便后续处理函数使用
		c.Set("userID", userID)
		// 继续执行后续的处理函数
		c.Next()
	}
}
