package handlers

import (
	"fmt"
	"myapp/config"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func GetViwer(c *gin.Context) {
	filePath := config.Conf.ViewerPath + "index.html"

	content, err := os.ReadFile(filePath)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("fail to load viewer: %v", err))
		return
	}

	c.Data(http.StatusOK, "text/html", content)
}
