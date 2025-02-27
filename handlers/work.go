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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func InitModel(c *gin.Context) {
	videoID := c.Param("id")
	fileName := c.Param("file_name")

	uVideoID, err := strconv.ParseUint(videoID, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "id is not a number",
		})
		return
	}

	// 找到video信息
	var video models.Video
	if err := config.Conf.DB.Where("id=?", uint(uVideoID)).First(&video).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Video Not Found",
		})
		return
	}

	// 创建work记录
	var work models.Work
	err = config.Conf.DB.Transaction(func(tx *gorm.DB) error {
		work = models.Work{
			UserName: video.UserName,
			FileName: fileName,
			Status:   "processing",
		}
		return tx.Create(&work).Error
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"init error": "Failed to initialize video model",
			"videoid":    uVideoID,
		})
		return
	}

	// 执行training
	processor, err := services.NewVideoProcessor()
	if err != nil {
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

	c.JSON(http.StatusOK, gin.H{
		"message": "Model initialization and processing completed successfully",
	})
}

func updateWorkStatus(workID uint, status, outputFolder, errorLog string, startTime time.Time) error {
	err := config.Conf.DB.Transaction(func(tx *gorm.DB) error {
		updates := map[string]interface{}{"status": status}
		if status == "completed" || status == "splat failed" {
			updates["process_time"] = int(time.Since(startTime).Seconds())
			updates["file_path"] = outputFolder
		}
		if errorLog != "" {
			updates["error_log"] = errorLog
		}
		return tx.Model(&models.Work{}).Where("id = ?", workID).Updates(updates).Error
	})
	if err != nil {
		return fmt.Errorf("status update error: %v", err)
	}
	return nil
}

func GetWorkPath(c *gin.Context) {
	var work models.Work
	workID := c.Param("id")

	if config.Conf.DB.First(&work, workID).Error != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Work Not Found",
		})
		return
	}

	if filePath, err := findSplatPath(work.FilePath); err == nil {
		c.JSON(http.StatusOK, gin.H{
			"filePath": filePath,
		})
		return
	}

	c.JSON(http.StatusContinue, gin.H{
		"continue": ".splat file not found, continue to find .ply file and convert to .splat file",
	})

	filepath, err := findPlyPath(work.FilePath)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Ply File Not Found",
		})
		return
	}

	splatPath, err := splat(filepath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"filePath": splatPath,
	})
}

func splat(workPath string) (string, error) {
	splatName := uuid.New().String()
	splatDirPath := filepath.Join(config.Conf.SplatPath, splatName)
	splatPath := filepath.Join(splatDirPath, ".splat")
	splatPathAbs, _ := filepath.Abs(splatPath)
	if err := os.Mkdir(splatDirPath, 0755); err != nil {
		return "", err
	}

	cmd := exec.Command("C:/Users/Administrator/anaconda3/envs/gaussian_splatting/python.exe", "web/convert.py", workPath, "--output", splatPathAbs)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PYTHONPATH=%s", "3DGS/gaussian-splatting/envs/gaussian_splatting"))
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("fail to convert to splat file:%w", err)
	}
	return splatPathAbs, nil
}

func findPlyPath(filePath string) (string, error) {
	if _, err := os.Stat(filePath + "/point_cloud/iteration_30000/point_cloud.ply"); err != nil {
		if _, err = os.Stat(filePath + "/point_cloud/iteration_7000/point_cloud.ply"); err != nil {
			if _, err = os.Stat(filePath + "input.ply"); err != nil {
				return "", err
			} else {
				filePath += "input.ply"
			}
		} else {
			filePath += "/point_cloud/iteration_7000/point_cloud.ply"
		}
	} else {
		filePath += "/point_cloud/iteration_7000/point_cloud.ply"
	}
	return filePath, nil
}

func findSplatPath(filePath string) (string, error) {
	if _, err := os.Stat(filePath + "/point_cloud/iteration_30000/point_cloud.splat"); err != nil {
		if _, err = os.Stat(filePath + "/point_cloud/iteration_7000/point_cloud.splat"); err != nil {
			if _, err = os.Stat(filePath + "input.splat"); err != nil {
				return "", err
			} else {
				filePath += "input.splat"
			}
		} else {
			filePath += "/point_cloud/iteration_7000/point_cloud.splat"
		}
	} else {
		filePath += "/point_cloud/iteration_7000/point_cloud.splat"
	}
	return filePath, nil
}
