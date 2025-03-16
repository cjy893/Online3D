package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SplatViewer(c *gin.Context) {
	workID := c.Query("id")
	if workID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "work_id不能为空",
		})
		return
	}
	c.Redirect(http.StatusFound, "/web/index.html?id="+workID)
}
