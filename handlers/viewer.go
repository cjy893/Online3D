package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func SplatViewer(c *gin.Context) {
	workID := c.Query("id")
	if workID == "" {
		c.String(http.StatusBadRequest, "id is required")
		return
	}
	c.File("./web/index.html")
}
