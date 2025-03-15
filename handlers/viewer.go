package handlers

import (
	"github.com/gin-gonic/gin"
)

func GetViwer(c *gin.Context) {
	c.File("web/index.html")
}
