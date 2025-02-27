package handlers

import (
	"myapp/config"
	"myapp/models"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func GetWork(c *gin.Context) {
	var work models.Work
	workID := c.Param("id")

	if config.Conf.DB.First(&work, workID).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Work Not Found",
		})
		return
	}

	filePath := filepath.Base(work.FileName)
	c.JSON(http.StatusOK, gin.H{
		"work":     work,
		"view_url": "/view/" + workID,
		"file_url": "/file/" + filePath,
	})
}
