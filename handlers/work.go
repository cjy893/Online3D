package handlers

import (
	"encoding/json"
	"fmt"
	"myapp/config"
	"myapp/database"
	"myapp/models"
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
	var initInfo struct {
		VideoID    uint   `json:"id"`
		WorkName   string `json:"workName"`
		IsPublic   bool   `json:"isPublic"`
		CoverUrl   string `json:"coverUrl"`
		Iterations string `json:"iterations"`
	}
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
		updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to initialize video model:%v", err), time.Now())
		c.JSON(http.StatusInternalServerError, gin.H{
			"init error": "Failed to initialize video model",
			"videoid":    initInfo.VideoID,
		})
		return
	}

	// 从存储桶中检索视频文件
	videoPath := fmt.Sprintf("videos/%d.mp4", video.ID)
	videoPath, err = database.RetrieveFromBucket(videoPath)
	if err != nil {
		updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to retrieve video:%v", err), time.Now())
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to find video:%v", err),
		})
		return
	}
	defer os.RemoveAll(filepath.Dir(videoPath))

	// 创建处理器实例
	processor, err := workService.NewProcessor(initInfo.Iterations)
	if err != nil {
		updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to train the model:%v", err), time.Now())
		c.JSON(http.StatusInternalServerError, gin.H{
			"init error": "fail to train the model",
		})
		return
	}

	startTime := time.Now()
	// 提取视频帧
	if err := processor.RunFfmpeg(videoPath); err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to generate pics:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to generate pics:%v", err),
		})
		return
	}

	dataPath := filepath.Dir(videoPath)
	// 运行COLMAP进行相机参数估计
	if err := processor.RunColmap(dataPath); err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to estimate the camera:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to estimate the camera:%v", err),
		})
	}

	// 执行三维重建
	if err := processor.Reconstruction(dataPath); err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to reconstruction:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to reconstruction:%v", err),
		})
		return
	}

	// 将处理结果存储到存储桶
	if err := database.StoreInBucketWIthDir(fmt.Sprintf("%d", work.ID), dataPath+"/undistorted"); err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to store data:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to store data:%v", err),
		})
	}

	// 获取并存储点云模型文件
	modelPath := filepath.Join(processor.OutputFolder, fmt.Sprintf("point_cloud/iteration_%s/point_cloud.ply", initInfo.Iterations))
	model, err := os.Open(modelPath)
	if err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to find model:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to find model:%v", err),
		})
		return
	}

	targetModelPath := fmt.Sprintf("%d/point_cloud/model.ply", work.ID)
	if err := database.StoreInBucket(targetModelPath, model); err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to store model:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to store model:%v", err),
		})
	}

	// 获取并存储检查点文件
	chkpntPath := filepath.Join(processor.OutputFolder, fmt.Sprintf("chkpnt%s.pth", initInfo.Iterations))
	chkpnt, err := os.Open(chkpntPath)
	if err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to find checkpoint:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to find checkpoint:%v", err),
		})
		return
	}

	targetChkpntPath := fmt.Sprintf("%d/checkpoint/chkpnt.pth", work.ID)
	if err := database.StoreInBucket(targetChkpntPath, chkpnt); err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to store checkpoint:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to store checkpoint:%v", err),
		})
	}

	// 更新作品状态为完成
	if updateErr := updateWorkStatus(work.ID, "completed", "", startTime); updateErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status error": updateErr.Error(),
		})
		return
	}

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

	var transferInfo struct {
		WorkID     uint   `json:"id"`
		WorkName   string `json:"work_name"`
		IsPublic   bool   `json:"is_public"`
		Iterations string `json:"iterations"`
	}
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
	var origin models.Work
	if err := config.Conf.DB.Where("id = ?", transferInfo.WorkID).First(&origin).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Work Not Found",
		})
		return
	}

	// 创建新的作品记录
	var work models.Work
	err = config.Conf.DB.Transaction(func(tx *gorm.DB) error {
		work = models.Work{
			UserID:     origin.UserID,
			WorkName:   transferInfo.WorkName,
			Status:     "processing",
			IsPublic:   transferInfo.IsPublic,
			Iterations: transferInfo.Iterations,
			ParentID:   workService.GetParentID(&origin),
		}
		return tx.Create(&work).Error
	})
	if err != nil {
		// 如果创建work记录失败，返回错误响应
		c.JSON(http.StatusInternalServerError, gin.H{
			"init error": "Failed to transfer model",
			"work_id":    transferInfo.WorkID,
		})
		return
	}

	// 从存储桶检索原始作品数据到本地
	dataPath := filepath.Join("transfer_tmp", uuid.NewString())
	err = database.RetrieveFromBucketWithDir(fmt.Sprintf("%d", *work.ParentID), dataPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to find work:%v", err),
		})
		return
	}
	dataPath += fmt.Sprintf("/%d", *work.ParentID)
	defer os.RemoveAll(filepath.Dir(dataPath))

	// 初始化处理器
	processor, err := workService.NewProcessor(transferInfo.Iterations)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"init error": "fail to train the model",
		})
		return
	}

	// 执行风格迁移处理
	startTime := time.Now()
	if err := processor.Stylize(dataPath, styleIMGPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to stylize:%v", err),
		})
		return
	}

	// 保存处理后的点云模型文件到存储桶
	modelPath := filepath.Join(processor.OutputFolder, fmt.Sprintf("point_cloud/iteration_%s/point_cloud.ply", transferInfo.Iterations))
	model, err := os.Open(modelPath)
	if err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to find model:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to find model:%v", err),
		})
		return
	}

	targetModelPath := fmt.Sprintf("%d/point_cloud/model.ply", work.ID)
	if err := database.StoreInBucket(targetModelPath, model); err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to store model:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to store model:%v", err),
		})
	}

	// 获取并存储检查点文件
	chkpntPath := filepath.Join(processor.OutputFolder, fmt.Sprintf("chkpnt%s.pth", transferInfo.Iterations))
	chkpnt, err := os.Open(chkpntPath)
	if err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to find checkpoint:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to find checkpoint:%v", err),
		})
		return
	}

	targetChkpntPath := fmt.Sprintf("%d/checkpoint/chkpnt.pth", work.ID)
	if err := database.StoreInBucket(targetChkpntPath, chkpnt); err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to store checkpoint:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to store checkpoint:%v", err),
		})
	}

	// 更新状态为完成
	if updateErr := updateWorkStatus(work.ID, "completed", "", startTime); updateErr != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status error": updateErr.Error(),
		})
		return
	}

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"message": "Model transfer completed successfully",
	})
}

// updateWorkStatus 更新工作的状态。
// 参数:
//
//	workID - 工作的唯一标识符。
//	status - 工作的新状态。
//	outputFolder - 工作输出文件的文件夹路径。
//	errorLog - 工作执行过程中遇到的错误日志。
//	startTime - 工作开始的时间。
//
// 返回值:
//
//	如果更新过程中发生错误，则返回错误。
func updateWorkStatus(workID uint, status, errorLog string, startTime time.Time) error {
	// 使用事务来更新工作状态，确保数据的一致性。
	err := config.Conf.DB.Transaction(func(tx *gorm.DB) error {
		// 初始化要更新的字段。
		updates := map[string]interface{}{"status": status}

		// 当工作完成或失败时，更新处理时间和文件路径。
		if status == "completed" || status == "splat failed" {
			updates["process_time"] = int(time.Since(startTime).Seconds())
		}

		// 如果有错误日志，则更新错误日志字段。
		if errorLog != "" {
			updates["error_log"] = errorLog
		}

		// 执行更新操作。
		return tx.Model(&models.Work{}).Where("id = ?", workID).Updates(updates).Error
	})

	// 如果更新过程中发生错误，返回详细的错误信息。
	if err != nil {
		return fmt.Errorf("status update error: %v", err)
	}

	// 更新成功，返回nil表示没有发生错误。
	return nil
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

	var workInfos []struct {
		WorkID   uint   `json:"work_id"`
		WorkName string `json:"workName"`
		Status   string `json:"status"`
	}

	if err := config.Conf.DB.Model(&models.Work{}).
		Where("user_id=?", user.ID).
		Select("id as work_id, work_name, status").
		Scan(&workInfos).Error; err != nil {
		// 如果数据库查询失败，返回错误响应
		c.JSON(http.StatusInternalServerError, gin.H{"error": "作品查询失败"})
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
