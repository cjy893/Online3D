package videoService

import (
	"fmt"
	"mime/multipart"
	"myapp/config"
	"myapp/database"
	"myapp/models"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func CheckUser(c *gin.Context) (*models.User, error) {
	// 尝试从上下文中获取用户ID，如果不存在，则返回未认证的用户错误
	userID, exists := c.Get("userID")
	if !exists {
		return nil, fmt.Errorf("未认证的用户")
	}

	// 初始化用户模型
	var user models.User

	if err := config.Conf.DB.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("用户不存在:%v", err)
	}

	return &user, nil
}

func SaveVideo(c *gin.Context, file *multipart.FileHeader) (string, error) {
	ext := filepath.Ext(file.Filename)
	videoFileUUID := uuid.New().String()
	videoFilePath := filepath.Join("temp", videoFileUUID, videoFileUUID+ext)

	if err := c.SaveUploadedFile(file, videoFilePath); err != nil {
		return "", fmt.Errorf("保存视频文件失败:%v", err)
	}
	return videoFilePath, nil
}

func SaveCover(c *gin.Context, file *multipart.FileHeader) (string, error) {
	ext := filepath.Ext(file.Filename)
	coverFileUUID := uuid.New().String()
	coverFilePath := filepath.Join("temp", coverFileUUID, coverFileUUID+ext)

	if err := c.SaveUploadedFile(file, coverFilePath); err != nil {
		return "", fmt.Errorf("保存封面文件失败:%v", err)
	}
	return coverFilePath, nil
}

func CreateVideo(video *models.Video, videoPath string) error {
	tx := config.Conf.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := createVideo(video); err != nil {
		tx.Rollback()
		return err
	}
	if err := uploadVideo(videoPath, video.ID); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func createVideo(video *models.Video) error {
	return config.Conf.DB.Create(video).Error
}

func uploadVideo(videoPath string, videoID uint) error {
	fileReader, err := os.Open(videoPath)
	if err != nil {
		return fmt.Errorf("打开视频文件失败:%v", err)
	}
	defer fileReader.Close()

	if err := database.StoreInBucket(fmt.Sprintf("videos/%d.mp4", videoID), fileReader); err != nil {
		return fmt.Errorf("上传视频文件失败:%v", err)
	}

	return nil
}
