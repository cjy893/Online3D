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

func checkUser(c *gin.Context) (*models.User, bool) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证的用户"})
		return nil, false
	}

	var user models.User
	if err := config.Conf.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "未找到用户"})
		return nil, false
	}
	return &user, true
}

func UploadVideo(c *gin.Context) {
	user, ok := checkUser(c)
	if !ok {
		return
	}

	file, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败"})
		return
	}

	title := c.PostForm("title")
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "标题不能为空"})
		return
	}

	// 生成唯一文件名
	ext := filepath.Ext(file.Filename)
	dirUUID := uuid.New().String()
	fileUUID := uuid.New().String()
	fileDirPath := filepath.Join(config.Conf.UploadPath, dirUUID)
	filePath := filepath.Join(fileDirPath, fileUUID+ext)

	if err := os.MkdirAll(fileDirPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建目录"})
		return
	}

	// 保存文件
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存失败"})
		return
	}

	// 创建视频记录（使用短事务）
	var video models.Video
	err = config.Conf.DB.Transaction(func(tx *gorm.DB) error {
		video = models.Video{
			UserName: user.Username,
			Title:    title,
			FilePath: filePath,
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

func ShowVideo(c *gin.Context) {
	user, ok := checkUser(c)
	if !ok {
		return
	}

	var videos []models.Video
	if err := config.Conf.DB.Where("user_id=?", user.ID).Find(&videos).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "No videos found for this user"})
		return
	}

	videoInfos := make([]gin.H, len(videos))
	for i, video := range videos {
		videoInfos[i] = gin.H{
			"video_id": video.ID,
			"title":    video.Title,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"videos": videoInfos,
	})
}
