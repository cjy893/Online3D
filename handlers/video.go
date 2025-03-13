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
			UserID:   user.ID,
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
