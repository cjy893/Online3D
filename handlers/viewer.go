package handlers

import (
	"github.com/gin-gonic/gin"
)

func SplatViewer(c *gin.Context) {
	c.File("./web/index.html")
}
