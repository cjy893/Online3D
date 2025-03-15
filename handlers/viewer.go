package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SplatViewer(c *gin.Context) {
	workID := c.Query("work_id")
	if workID == "" {
		c.String(http.StatusBadRequest, "work_id is required")
		return
	}
	c.File("./web/index.html")
}
