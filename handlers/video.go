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

// UploadVideo 上传视频处理函数
// 该函数负责处理视频上传请求，包括验证用户身份、接收上传文件、保存文件、
// 以及在数据库中创建视频记录
func UploadVideo(c *gin.Context) {
	// 验证用户身份
	user, ok := checkUser(c)
	if !ok {
		return
	}

	// 接收上传文件
	file, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败"})
		return
	}

	// 获取视频标题
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

	// 创建存储目录
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

	// 返回成功响应
	c.JSON(http.StatusCreated, gin.H{
		"message":  "Video uploaded successfully",
		"video_id": video.ID,
	})
}

// ShowVideo 显示用户视频信息
// 该函数首先检查当前上下文中的用户身份，如果用户身份验证失败，则中止执行。
// 接着，从数据库中查询属于该用户的所有视频信息，如果查询失败或未找到视频，则返回错误信息。
// 最后，将视频信息整理为一个切片，其中每个元素包含视频ID和标题，然后以JSON格式返回给客户端。
func ShowVideo(c *gin.Context) {
	// 检查并获取用户信息
	user, ok := checkUser(c)
	if !ok {
		return
	}

	// 初始化一个视频切片，用于存储查询到的视频信息
	var videos []models.Video
	// 查询数据库中属于当前用户的所有视频
	if err := config.Conf.DB.Where("user_id=?", user.ID).Find(&videos).Error; err != nil {
		// 如果查询失败，返回404错误信息
		c.JSON(http.StatusNotFound, gin.H{"error": "No videos found for this user"})
		return
	}

	// 初始化一个gin.H切片，用于存储视频信息的摘要
	videoInfos := make([]gin.H, len(videos))
	// 遍历视频信息，提取并存储视频ID和标题
	for i, video := range videos {
		videoInfos[i] = gin.H{
			"video_id": video.ID,
			"title":    video.Title,
		}
	}

	// 返回视频信息的JSON响应
	c.JSON(http.StatusOK, gin.H{
		"videos": videoInfos,
	})
}
