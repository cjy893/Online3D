package handlers

import (
	"fmt"
	"myapp/config"
	"myapp/models"
	"myapp/services"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// InitModel 初始化视频模型并进行处理
// 参数:
//
//	c *gin.Context: Gin框架的上下文对象，用于处理HTTP请求和响应
func InitModel(c *gin.Context) {
	// 获取URL参数
	videoID := c.Param("id")
	fileName := c.Param("file_name")
	iterations := c.Param("iterations")

	// 将视频ID转换为uint64类型
	uVideoID, err := strconv.ParseUint(videoID, 10, 32)
	if err != nil {
		// 如果转换失败，返回错误响应
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "id is not a number",
		})
		return
	}

	// 找到video信息
	var video models.Video
	if err := config.Conf.DB.Where("id=?", uVideoID).First(&video).Error; err != nil {
		// 如果找不到视频，返回错误响应
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Video Not Found",
		})
		return
	}

	// 创建work记录
	var work models.Work
	err = config.Conf.DB.Transaction(func(tx *gorm.DB) error {
		work = models.Work{
			UserName:   video.UserName,
			FileName:   fileName,
			Status:     "processing",
			Iterations: iterations,
		}
		return tx.Create(&work).Error
	})
	if err != nil {
		// 如果创建work记录失败，返回错误响应
		c.JSON(http.StatusInternalServerError, gin.H{
			"init error": "Failed to initialize video model",
			"videoid":    uVideoID,
		})
		return
	}

	// 执行training
	processor, err := services.NewVideoProcessor(iterations)
	if err != nil {
		// 如果处理失败，更新work状态并返回错误响应
		if updateErr := updateWorkStatus(work.ID, "process failed", "", err.Error(), time.Now()); updateErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status error": updateErr.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"init error": "fail to train the model",
		})
		return
	}

	startTime := time.Now()
	if err := processor.ProcessVideo(&video, processor); err != nil {
		// 如果视频处理失败，更新work状态并返回错误响应
		if updateErr := updateWorkStatus(work.ID, "process failed", processor.OutputFolder, err.Error(), startTime); updateErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status error": updateErr.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "fail to train the model",
		})
		return
	}

	// 执行splat
	if err := processor.Splat(processor.OutputFolder); err != nil {
		// 如果splat操作失败，更新work状态并返回错误响应
		if updateErr := updateWorkStatus(work.ID, "process failed", processor.OutputFolder, err.Error(), startTime); updateErr != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"status error": updateErr.Error(),
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "fail to splat",
		})
		return
	}

	// 更新状态为完成
	if updateErr := updateWorkStatus(work.ID, "completed", processor.OutputFolder, "", startTime); updateErr != nil {
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
func updateWorkStatus(workID uint, status, outputFolder, errorLog string, startTime time.Time) error {
	// 使用事务来更新工作状态，确保数据的一致性。
	err := config.Conf.DB.Transaction(func(tx *gorm.DB) error {
		// 初始化要更新的字段。
		updates := map[string]interface{}{"status": status}

		// 当工作完成或失败时，更新处理时间和文件路径。
		if status == "completed" || status == "splat failed" {
			updates["process_time"] = int(time.Since(startTime).Seconds())
			updates["file_path"] = outputFolder
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

// GetWorkPath 获取作品的文件路径
// 该函数首先尝试根据ID从数据库中获取作品信息，然后根据作品的文件路径寻找对应的.splat文件
// 如果.splat文件不存在，则尝试寻找.ply文件并将其转换为.splat文件
// 参数:
//
//	c *gin.Context - Gin框架的上下文，用于处理HTTP请求和响应
func GetWork(c *gin.Context) {
	// 初始化一个Work结构体实例
	var work models.Work
	// 从请求参数中获取作品ID
	workID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		//如果id解析失败，返回400错误
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "id is not a number",
		})
		return
	}

	// 尝试从数据库中获取作品信息
	if config.Conf.DB.First(&work, workID).Error != nil {
		// 如果作品不存在，返回404错误
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Work Not Found",
		})
		return
	}

	// 尝试寻找.splat文件
	baseFilePath := filepath.Join(work.FilePath, "point_cloud", "iteration_"+work.Iterations)
	splatPath := filepath.Join(baseFilePath, "point_cloud.splat")
	if _, err := os.Stat(splatPath); err == nil {
		//如果找到.splat文件，返回文件
		c.File(splatPath)
		return
	}

	// 如果未找到.splat文件，继续寻找.ply文件并转换为.splat文件
	c.JSON(http.StatusContinue, gin.H{
		"continue": ".splat file not found, continue to find .ply file and convert to .splat file",
	})

	// 寻找.ply文件
	plyPath := filepath.Join(baseFilePath, "point_cloud.ply")
	if _, err := os.Stat(plyPath); err != nil {
		// 如果.ply文件不存在，返回404错误
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Ply File Not Found",
		})
		return
	}

	// 将.ply文件转换为.splat文件
	splatPath, err = splat(plyPath)
	if err != nil {
		// 如果转换过程中出现错误，返回400错误
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 返回转换后的.splat文件
	c.File(splatPath)
}

// splat函数负责将给定的工作路径下的数据转换为"splat"格式。
// 这个函数首先生成一个唯一的"splat"名称，然后在配置的SplatPath下创建一个目录，
// 并在这个目录中创建一个名为".splat"的文件。接着，它使用Python脚本将工作路径下的数据
// 转换为"splat"格式，并将输出保存在刚创建的".splat"文件中。
//
// 参数:
//
//	workPath - string类型，表示需要转换的数据的工作路径。
//
// 返回值:
//
//	string类型，表示转换后的"splat"文件的绝对路径。
//	error类型，如果转换过程中发生错误，则返回该错误。
func splat(workPath string) (string, error) {
	// 尝试在指定的工作路径中找到.ply文件。
	plyPath := filepath.Join(workPath, "point_cloud.splat")
	if _, err := os.Stat(plyPath); err != nil {
		return "", fmt.Errorf("fail to find .ply file: %v", err)
	}

	// 生成.splat文件的路径，通过替换.ply文件的扩展名实现。
	splatPath := strings.Split(plyPath, ".")[0]
	splatPath += ".splat"

	// 构建执行Python转换脚本的命令。
	// 使用VideoProcessor实例中指定的Python解释器。
	cmd := exec.Command(config.Conf.PythonPath+"/Python.exe", "web/convert.py", plyPath, "--output", splatPath)
	// 添加环境变量以确保Python脚本可以找到所需的库。
	cmd.Env = append(os.Environ(), fmt.Sprintf("PYTHONPATH=%s", "3DGS/gaussian-splatting/envs/gaussian_splatting"))

	// 执行命令并检查是否有错误发生。
	if err := cmd.Run(); err != nil {
		// 如果执行命令时出错，返回错误。
		return "", fmt.Errorf("fail to convert to splat file:%w", err)
	}

	// 如果一切顺利，返回nil表示没有发生错误。
	return splatPath, nil
}
