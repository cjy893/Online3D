package handlers

import (
	"fmt"
	"myapp/config"
	"myapp/database"
	"myapp/models"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// checkUser 检查并返回当前请求的用户信息
// 参数:
//
//	c *gin.Context: Gin框架的上下文对象，用于处理HTTP请求和响应
//
// 返回值:
//
//	*models.User: 用户信息的指针，如果用户存在且验证通过
//	bool: 表示是否成功获取到用户信息
func checkUser(c *gin.Context) (*models.User, bool) {
	// 尝试从上下文中获取用户ID，如果不存在，则返回未认证的用户错误
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证的用户"})
		return nil, false
	}

	// 初始化用户模型
	var user models.User

	// 使用用户ID查询数据库中的用户信息，如果查询失败，则返回内部服务器错误
	if err := config.Conf.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "未找到用户"})
		return nil, false
	}

	// 成功获取用户信息，返回用户信息和成功标志
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

	ext := filepath.Ext(file.Filename)
	fileUUID := uuid.New().String()
	filePath := filepath.Join("temp", fileUUID, fileUUID+ext)

	if err := c.SaveUploadedFile(file, filePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存失败"})
		return
	}
	defer os.RemoveAll(filepath.Dir(filePath))

	tx := config.Conf.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var video = models.Video{
		UserID: user.ID,
		Title:  title,
	}
	if err := tx.Create(&video).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to upload video:%v", err),
		})
		return
	}

	fileReader, err := os.Open(filePath)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to open file:%v", err),
		})
		return
	}
	defer fileReader.Close()
	if err := database.StoreInBucket(fmt.Sprintf("%d", video.ID), "video", fileReader); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to upload video:%v", err),
		})
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to commit :%v", err),
		})
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Video uploaded successfully",
		"video_id": video.ID,
	})
}

// ShowVideo 处理用户视频列表请求，验证用户身份后查询数据库并返回视频信息
// 参数说明:
//   - c: *gin.Context Gin框架上下文对象，用于处理HTTP请求和响应
//
// 功能流程:
//   - 执行用户身份验证
//   - 查询当前用户关联的视频数据
//   - 返回标准化JSON响应
func ShowVideo(c *gin.Context) {
	// 用户身份验证检查
	user, ok := checkUser(c)
	if !ok {
		return
	}

	var videoInfos []struct {
		VideoID uint   `json:"video_id"`
		Title   string `json:"title"`
	}
	// 数据库查询操作：获取当前用户的视频ID和标题
	if err := config.Conf.DB.Model(&models.Video{}).
		Where("user_id = ?", user.ID).
		Select("id as video_id, title").
		Scan(&videoInfos).Error; err != nil {
		// 数据库查询错误处理
		c.JSON(http.StatusInternalServerError, gin.H{"error": "视频查询失败"})
		return
	}

	//如果没有视频记录，返回空数组
	if len(videoInfos) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"message": "当前没有视频记录",
			"videos":  []interface{}{},
		})
		return
	}

	// 成功返回视频数据
	c.JSON(http.StatusOK, gin.H{
		"message": "视频查询成功",
		"videos":  videoInfos,
	})
}
