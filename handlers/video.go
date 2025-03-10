package handlers

import (
	"log"
	"myapp/config"
	"myapp/models"
	"myapp/services"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	// 异步处理视频（不包裹在事务中）
	go processVideo(video.ID, user.Username, video.FileName)

	c.JSON(http.StatusCreated, gin.H{
		"message":  "Video uploaded successfully",
		"video_id": video.ID,
	})
}

func processVideo(videoID uint, userName, fileName string) {
	// 短事务1：标记为"processing"（快速提交）
	var video models.Video
	err := config.Conf.DB.Transaction(func(tx *gorm.DB) error {
		// 锁定记录并更新状态
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			First(&video, videoID).Error; err != nil {
			return err
		}
		return tx.Model(&video).Update("status", "processing").Error
	})
	if err != nil {
		log.Printf("Failed to start processing video %d: %v", videoID, err)
		return
	}

	// 执行耗时操作（不在事务中）
	processor, err := services.NewVideoProcessor()
	if err != nil {
		updateVideoStatus(videoID, "failed", err.Error(), time.Now())
		return
	}

	startTime := time.Now()
	if err := processor.Process(&video, processor); err != nil {
		updateVideoStatus(videoID, "failed", err.Error(), startTime)
		return
	} else {
		updateVideoStatus(videoID, "completed", "", startTime)
		_ = config.Conf.DB.Transaction(func(tx *gorm.DB) error {
			work := models.Work{
				UserName: userName,
				FileName: fileName,
				FilePath: processor.OutputFolder,
			}
			return tx.Create(&work).Error
		})
	}
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
		"file_url": "/uploads/" + fileName, // 通过Static路由访问
	})
}

func updateVideoStatus(videoID uint, status, errorLog string, startTime time.Time) {
	err := config.Conf.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{"status": status}
		if status == "completed" {
			updates["process_time"] = int(time.Since(startTime).Seconds())
		}
		if errorLog != "" {
			updates["error_log"] = errorLog
		}
		return tx.Model(&models.Video{}).Where("id = ?", videoID).Updates(updates).Error
	})
	if err != nil {
		log.Printf("Failed to update video %d status: %v", videoID, err)
	}
}
