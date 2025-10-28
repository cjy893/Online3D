package handlers

import (
	"fmt"
	"myapp/config"
	"myapp/models"
	"myapp/services/videoService"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
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
func UploadVideo(c *gin.Context) {
	user, err := videoService.CheckUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	videoFile, err := c.FormFile("video")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "文件上传失败"})
		return
	}

	title := c.PostForm("title")
	if title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "标题不能为空"})
		return
	}

	isPublic := c.PostForm("is_public") == "true"

	videoPath, err := videoService.SaveVideo(c, videoFile)
	defer os.Remove(filepath.Dir(videoPath))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	var video = models.Video{
		UserID:   user.ID,
		Title:    title,
		CoverUrl: "default",
		IsPublic: isPublic,
	}

	if err := videoService.CreateVideo(&video, videoPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to create video:%v", err),
		})
		return
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
	user, err := videoService.CheckUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
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

func SearchVideos(c *gin.Context) {
	q := c.Query("q")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize := 20

	var videos []models.Video
	if err := config.Conf.DB.Where("title LIKE ?", "%"+q+"%").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&videos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "视频查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "视频查询成功",
		"videos":  videos,
	})
}
