package handlers

import (
	"encoding/json"
	"fmt"
	"myapp/config"
	"myapp/database"
	"myapp/models"
	"myapp/services/websocket"
	"myapp/services/workService"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InitModel 处理3D模型初始化请求
//
// 该函数接收一个包含视频ID、作品名称、可见性设置、封面URL和迭代次数的JSON请求，
// 然后通过视频数据进行3D模型重建处理，包括视频帧提取、相机参数估计、三维重建等步骤，
// 最终将生成的点云模型和检查点文件存储到存储桶中。
//
// 参数:
//   - c: Gin框架的上下文对象，用于处理HTTP请求和响应
//
// JSON参数:
//   - id: 视频ID
//   - workName: 作品名称
//   - isPublic: 是否公开作品
//   - coverUrl: 封面图片URL
//   - iterations: 迭代次数
func InitModel(c *gin.Context) {
	var initInfo workService.InitInfo
	if err := c.ShouldBindJSON(&initInfo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	var video models.Video
	if err := config.Conf.DB.Where("id=?", initInfo.VideoID).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Video Not Found",
		})
		return
	}

	var work models.Work
	err := config.Conf.DB.Transaction(func(tx *gorm.DB) error {
		work = models.Work{
			UserID:     video.UserID,
			WorkName:   initInfo.WorkName,
			Status:     "processing",
			Iterations: initInfo.Iterations,
		}
		return tx.Create(&work).Error
	})
	if err != nil {
		workService.UpdateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to initialize video model:%v", err), time.Now())
		c.JSON(http.StatusInternalServerError, gin.H{
			"init error": "Failed to initialize video model",
			"videoid":    initInfo.VideoID,
		})
		return
	}

	task := websocket.Task{
		Data:      initInfo,
		ID:        uuid.New().String(),
		StartTime: time.Now(),
		Type:      "stylize",
		UserID:    work.UserID,
		WorkID:    work.ID,
	}

	workService.ProcessTasks(&task)

	c.JSON(http.StatusOK, gin.H{
		"message": "Model initialization and processing completed successfully",
	})
}

// Transfer 处理模型风格迁移请求
//
// 该函数接收一个包含源作品ID、新作品名称以及风格迁移参数的JSON请求，
// 同时需要上传一张风格图片，然后对指定的作品执行风格迁移操作。
//
// 参数:
//   - c: Gin框架的上下文对象，用于处理HTTP请求和响应
//
// JSON参数:
//   - id: 源作品的ID
//   - work_name: 新生成作品的名称
//   - style: 风格类型（未在代码中使用）
//   - weight: 权重参数（未在代码中使用）
//   - iterations: 迭代次数
//
// 文件参数:
//   - style_img: 用于风格迁移的风格图像文件
func Transfer(c *gin.Context) {
	jsonData := c.PostForm("data")

	var transferInfo workService.TransferInfo
	if err := json.Unmarshal([]byte(jsonData), &transferInfo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON data"})
		return
	}

	// 获取并保存上传的风格图像
	styleIMGFile, err := c.FormFile("style_img")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Style image is required"})
		return
	}

	styleIMGPath := filepath.Join("transfer_img_tmp", uuid.NewString()+".jpg")
	if err := c.SaveUploadedFile(styleIMGFile, styleIMGPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to save style image:%v", err),
		})
		return
	}
	defer os.Remove(styleIMGPath)

	// 查找原始作品信息
	origin, err := workService.QueryWork(transferInfo.WorkID)

	// 创建新的作品记录
	work, err := workService.NewWork(transferInfo, origin)
	if err != nil {
		// 如果创建work记录失败，返回错误响应
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   fmt.Sprintf("fail to create new work record:%v", err),
			"work_id": transferInfo.WorkID,
		})
		return
	}

	taskData := struct {
		TransferInfo   workService.TransferInfo `json:"transfer_info"`
		StyleImagePath string                   `json:"style_image_path"`
		WorkID         uint                     `json:"work_id"`
	}{
		TransferInfo:   transferInfo,
		StyleImagePath: styleIMGPath,
		WorkID:         work.ID,
	}

	task := websocket.Task{
		Data:      taskData,
		ID:        uuid.New().String(),
		StartTime: time.Now(),
		Type:      "transfer",
		UserID:    work.UserID,
		WorkID:    work.ID,
	}

	workService.ProcessTasks(&task)

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"message": "Model transfer completed successfully",
	})
}

// TODO
func UploadWork(c *gin.Context) {
	user, err := workService.CheckUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	file, err := c.FormFile("work")
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
	defer os.RemoveAll(filePath)

	tx := config.Conf.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var work = models.Work{
		UserID:   user.ID,
		Status:   "completed",
		WorkName: title,
	}
	if err := tx.Create(&work).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to upload work:%v", err),
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
	if err := database.StoreInBucket(fmt.Sprintf("%d", work.ID), fileReader); err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to upload work:%v", err),
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
		"message": "Work uploaded successfully",
		"work_id": work.ID,
	})
}

// GetWork 根据ID获取作品的点云模型文件并返回给客户端
// 参数:
//
//	c *gin.Context: Gin框架的上下文对象，用于处理HTTP请求和响应
//
// 该函数会:
// 1. 从请求参数中获取作品ID
// 2. 构造点云模型文件在存储中的路径
// 3. 从存储中检索该文件
// 4. 将文件发送给客户端
// 5. 函数返回后清理临时文件
func GetWork(c *gin.Context) {
	workID := c.Query("id")

	plyPath := filepath.Join(fmt.Sprintf("%s", workID), "point_cloud", "model.ply")
	plyPath, err := database.RetrieveFromBucket(plyPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("fail to retrieve work: %v", err)})
		return
	}
	defer os.Remove(plyPath)
	c.File(plyPath)
}

func ShowWork(c *gin.Context) {
	user, err := workService.CheckUser(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	workInfos, err := workService.QueryWorkInfo(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to query work: %v", err),
		})
		return
	}

	if len(workInfos) == 0 {
		// 如果没有作品记录，返回空数组
		c.JSON(http.StatusOK, gin.H{
			"message": "当前没有作品记录",
			"works":   []interface{}{},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "作品查询成功",
		"works":   workInfos,
	})
}
