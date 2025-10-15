package handlers

import (
	"fmt"
	"log"
	"myapp/agent"
	"myapp/config"
	"myapp/database"
	"myapp/models"
	"myapp/services"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// InitModel 初始化视频模型并进行处理
// 参数:
//
//	c *gin.Context: Gin框架的上下文对象，用于处理HTTP请求和响应
func InitModel(c *gin.Context) {
	//获取初始化模型信息
	var initInfo struct {
		VideoID    uint   `json:"id"`
		WorkName   string `json:"workName"`
		IsPublic   bool   `json:"isPublic"`
		CoverUrl   string `json:"coverUrl"`
		Iterations string `json:"iterations"`
	}
	if err := c.ShouldBindJSON(&initInfo); err != nil {
		// 如果解析JSON失败，返回错误响应
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	// 找到video信息
	var video models.Video
	if err := config.Conf.DB.Where("id=?", initInfo.VideoID).First(&video).Error; err != nil {
		// 如果找不到视频，返回错误响应
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Video Not Found",
		})
		return
	}

	// 创建work记录
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
		// 如果创建work记录失败，返回错误响应
		c.JSON(http.StatusInternalServerError, gin.H{
			"init error": "Failed to initialize video model",
			"videoid":    initInfo.VideoID,
		})
		return
	}

	videoPath := filepath.Join("videos", fmt.Sprintf("%d.mp4", initInfo.VideoID))
	videoPath, err = database.RetrieveFromBucket(videoPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to find video:%v", err),
		})
		return
	}
	defer os.RemoveAll(filepath.Dir(videoPath))

	// 执行training
	processor, err := services.NewProcessor(initInfo.Iterations)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"init error": "fail to train the model",
		})
		return
	}

	startTime := time.Now()
	if err := processor.RunFfmpeg(videoPath); err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to generate pics:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to generate pics:%v", err),
		})
		return
	}

	dataPath := filepath.Dir(videoPath)
	if err := processor.RunColmap(dataPath); err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to estimate the camera:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to estimate the camera:%v", err),
		})
	}

	if err := processor.Reconstruction(dataPath); err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to reconstruction:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to reconstruction:%v", err),
		})
		return
	}

	if err := database.StoreInBucketWIthDir(fmt.Sprintf("%d", work.ID), dataPath); err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to store data:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to store data:%v", err),
		})
	}

	model, err := os.Open(filepath.Join(processor.OutputFolder, fmt.Sprintf("point_cloud/iteration_%s/point_cloud.ply", initInfo.Iterations)))
	if err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to find model:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to find model:%v", err),
		})
		return
	}
	if err := database.StoreInBucket(filepath.Join(fmt.Sprintf("%d", work.ID), dataPath, "point_cloud", "model.ply"), model); err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to store model:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to store model:%v", err),
		})
	}

	chkpnt, err := os.Open(filepath.Join(processor.OutputFolder, fmt.Sprintf("checkpoint%s.pth", initInfo.Iterations)))
	if err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to find checkpoint:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to find checkpoint:%v", err),
		})
		return
	}

	if err := database.StoreInBucket(filepath.Join(fmt.Sprintf("%d", work.ID), dataPath, "checkpoint", "chkpnt.pth"), chkpnt); err != nil {
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
		"message": "Model initialization and processing completed successfully",
	})
}

func Transfer(c *gin.Context) {
	var transferInfo struct {
		WorkID     uint   `json:"id"`
		WorkName   string `json:"workName"`
		Style      string `json:"style"`
		Weight     string `json:"weight"`
		Iterations string `json:"iterations"`
	}
	if err := c.ShouldBindJSON(&transferInfo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	var origin models.Work
	if err := config.Conf.DB.Where("id = ?", transferInfo.WorkID).First(&origin).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Work Not Found",
		})
		return
	}

	var work models.Work
	err := config.Conf.DB.Transaction(func(tx *gorm.DB) error {
		work = models.Work{
			UserID:     origin.UserID,
			WorkName:   transferInfo.WorkName,
			Status:     "processing",
			Iterations: transferInfo.Iterations,
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

	err = database.RetrieveFromBucketWithDir(fmt.Sprintf("%d", transferInfo.WorkID), "transfer_tmp")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to find work:%v", err),
		})
		return
	}

	processor, err := services.NewProcessor(transferInfo.Iterations)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"init error": "fail to train the model",
		})
		return
	}

	stylePath := "img/style.png"
	dataPath := filepath.Join("transfer_tmp", fmt.Sprintf("%d", transferInfo.WorkID))

	startTime := time.Now()
	if err := processor.Stylize(dataPath, stylePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to stylize:%v", err),
		})
		return
	}

	model, err := os.Open(filepath.Join(processor.OutputFolder, fmt.Sprintf("point_cloud/iteration_%s/point_cloud.ply", transferInfo.Iterations)))
	if err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to find model:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to find model:%v", err),
		})
		return
	}
	if err := database.StoreInBucket(filepath.Join(fmt.Sprintf("%d", work.ID), dataPath, "point_cloud", "model.ply"), model); err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to store model:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to store model:%v", err),
		})
	}

	chkpnt, err := os.Open(filepath.Join(processor.OutputFolder, fmt.Sprintf("checkpoint%s.pth", transferInfo.Iterations)))
	if err != nil {
		_ = updateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to find checkpoint:%v", err), startTime)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("fail to find checkpoint:%v", err),
		})
		return
	}

	if err := database.StoreInBucket(filepath.Join(fmt.Sprintf("%d", work.ID), dataPath, "checkpoint", "chkpnt.pth"), chkpnt); err != nil {
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

func TransferByAIAgent(c *gin.Context) {
	var transferInfo struct {
		UserInput string `json:"userInput"`
	}
	if err := c.ShouldBindJSON(&transferInfo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	resp, err := agent.ServerAgent.Invoke(c.Request.Context(), transferInfo.UserInput)
	response := gin.H{
		"message": resp,
	}
	if err != nil {
		response["error"] = err.Error()
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	c.JSON(http.StatusOK, response)
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
	user, ok := checkUser(c)
	if !ok {
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

// GetWorkPath 获取作品的文件路径
// 该函数首先尝试根据ID从数据库中获取作品信息，然后根据作品的文件路径寻找对应的.splat文件
// 如果.splat文件不存在，则尝试寻找.ply文件并将其转换为.splat文件
// 参数:
//
//	c *gin.Context - Gin框架的上下文，用于处理HTTP请求和响应
func GetWork(c *gin.Context) {
	workID := c.Query("id")

	splatPath := filepath.Join("temp", workID, workID+".splat")
	splatPath, err := database.RetrieveFromBucket(splatPath)
	defer func() {
		err := os.RemoveAll(filepath.Dir(splatPath))
		if err != nil {
			log.Println("Failed to remove temporary directory:", err)
		}
	}()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("fail to retrieve work: %v", err)})
		return
	}
	defer os.RemoveAll(filepath.Dir(splatPath))
	c.File(splatPath)
}

func ShowWork(c *gin.Context) {
	user, ok := checkUser(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证的用户"})
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
