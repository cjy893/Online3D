package handlers

import (
	"myapp/agent"
	"myapp/services/workService"
	"net/http"

	"github.com/gin-gonic/gin"
)

func ChatHandler(c *gin.Context) {
	user, err := workService.CheckUser(c)
	if err != nil {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return
	}

	var req struct {
		Content string `json:"content"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "Invalid request"})
		return
	}

	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// 从上下文中获取 user ID
	userID := user.ID

	// 调用 Stream 方法，传入响应 Writer 和用户 ID
	err = agent.ServerAgent.Stream(c.Request.Context(), req.Content, userID, c.Writer)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
		return
	}
}