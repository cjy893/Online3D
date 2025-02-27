package handlers

import (
	"myapp/config"
	"myapp/models"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func UploadVideo(c *gin.Context) {
	userID, _ := c.Get("userID")
	var user models.User
	if err := config.Conf.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "未找到用户"})
		return
	}

	file, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败"})
		return
	}

	// 生成唯一文件名
	ext := filepath.Ext(file.Filename)
	newFileName := uuid.New().String()
	fileDirPath := filepath.Join(config.Conf.UploadPath, newFileName)
	filePath := filepath.Join(fileDirPath, newFileName+ext)
	filePathAbs, _ := filepath.Abs(filePath)

	if err := os.MkdirAll(fileDirPath, 0755); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败"})
		return
	}

	// 保存文件
	if err := c.SaveUploadedFile(file, filePathAbs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存失败"})
		return
	}

	// 创建视频记录（使用短事务）
	var video models.Video
	err = config.Conf.DB.Transaction(func(tx *gorm.DB) error {
		video = models.Video{
			VideoID:  newFileName,
			UserName: user.Username,
			FileName: file.Filename,
			FilePath: filePathAbs,
			Status:   "uploaded",
		}
		return tx.Create(&video).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create video record"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Video uploaded successfully",
		"video_id": video.ID,
	})
}

func GetVideo(c *gin.Context) {
	var video models.Video
	videoID := c.Param("id")

	if config.Conf.DB.First(&video, videoID).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Video not found"})
		return
	}

	// 返回相对路径
	fileName := filepath.Base(video.FilePath)
	c.JSON(http.StatusOK, gin.H{
		"video":    video,
		"view_url": "/view/" + videoID,
		"file_url": "/uploads/" + videoID + "/" + fileName, // 通过Static路由访问
	})
}
