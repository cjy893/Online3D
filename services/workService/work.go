package workService

import (
	"context"
	"fmt"
	"log"
	"myapp/config"
	"myapp/database"
	"myapp/models"
	"myapp/services/websocket"
	"myapp/utils"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type InitInfo struct {
	VideoID    uint   `json:"id"`
	WorkName   string `json:"workName"`
	IsPublic   bool   `json:"isPublic"`
	CoverUrl   string `json:"coverUrl"`
	Iterations string `json:"iterations"`
}

type TransferInfo struct {
	WorkID     uint   `json:"id"`
	WorkName   string `json:"work_name"`
	IsPublic   bool   `json:"is_public"`
	Iterations string `json:"iterations"`
}

type WorkInfo struct {
	WorkID   uint   `json:"work_id"`
	WorkName string `json:"workName"`
	Status   string `json:"status"`
}

type Processor struct {
	TrainerPath      string
	PythonPath       string
	BaseOutputFolder string
	OutputFolder     string
	FPS              int
	Iterations       string
}

// NewVideoProcessor 创建并初始化一个新的VideoProcessor实例。
// 该函数无需参数。
// 返回值是一个指向VideoProcessor实例的指针，以及一个错误值（如果有）。
func NewProcessor(iterations string) (*Processor, error) {
	// 获取项目根目录的路径。
	projectRoot := utils.GetProjectRoot()

	// 拼接项目根目录与训练脚本相对路径，得到完整的训练脚本路径。
	trainerPath := utils.SafeJoin(projectRoot, "3DGS/gaussian-splatting")
	// 如果训练脚本路径为空，则返回错误。
	if trainerPath == "" {
		return nil, fmt.Errorf("invalid trainer path")
	}

	// 返回一个新的VideoProcessor实例，包含了一系列预设的属性值。
	return &Processor{
		TrainerPath:      trainerPath,
		PythonPath:       utils.SafeJoin(projectRoot, config.Conf.PythonPath),
		BaseOutputFolder: utils.SafeJoin(projectRoot, "output"),
		OutputFolder:     "",
		FPS:              12,
		Iterations:       iterations,
	}, nil
}

func (vp *Processor) RunFfmpeg(videoPath string) error {
	cmd := exec.Command("python", filepath.Join(vp.TrainerPath, "video_to_image.py"), "-v", videoPath, "--fps", strconv.Itoa(vp.FPS))

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w", err)
	}

	return nil
}

func (vp *Processor) RunColmap(dataPath string) error {
	modelName := filepath.Base(dataPath)
	cmd := exec.Command("xvfb-run", "-a", "python", filepath.Join(vp.TrainerPath, "convert.py"), "-s", "temp", "-m", modelName)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("colmap failed: %w", err)
	}

	return nil
}

// runTraining 运行训练程序以处理指定路径的视频。
// 该函数接受视频路径和输出文件夹路径作为参数。
// 它返回训练过程中生成的输出路径或者错误信息（如果有）。
func (vp *Processor) Reconstruction(dataPath string) error {
	// 构建运行训练脚本的命令。
	modelPath := filepath.Join(vp.BaseOutputFolder, uuid.NewString())
	cmd := exec.Command("python", filepath.Join(vp.TrainerPath, "train.py"),
		"--source_path", filepath.Join(dataPath, "undistorted"),
		"--model_path", modelPath,
		"--iterations", vp.Iterations,
		"--save_iterations", vp.Iterations,
		"--checkpoint_iterations", vp.Iterations,
		"--resolution", "1")

	// 准备缓冲区以存储命令的输出。
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 打印训练开始的信息。
	fmt.Printf("Starting training process for video: %s\n", dataPath)
	fmt.Println(cmd.Args)

	// 执行命令并处理错误（如果有）。
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("training failed: %w", err)
	}

	vp.OutputFolder = modelPath
	return nil
}

func (vp *Processor) Stylize(dataPath, stylePath string) error {
	modelPath := filepath.Join(vp.BaseOutputFolder, uuid.NewString())
	cmd := exec.Command("python", filepath.Join(vp.TrainerPath, "stylize.py"),
		"--source_path", filepath.Join(dataPath),
		"--model_path", modelPath,
		"--start_checkpoint", filepath.Join(dataPath, "checkpoint", "chkpnt.pth"),
		"--style_img", stylePath,
		"--iterations", vp.Iterations,
		"--resolution", "1")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 打印训练开始的信息。
	fmt.Printf("Starting stylizing process for video: %s\n", dataPath)
	fmt.Println(cmd.String())

	// 执行命令并处理错误（如果有）。
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("stylizing failed: %w", err)
	}

	vp.OutputFolder = modelPath
	return nil
}

// Splat 是一个方法，用于将给定路径下的.ply文件转换为.splat文件。
// 它依赖于一个Python脚本进行实际的转换过程。
// 参数:
//
//	workPath - 指定的工作路径，用于查找.ply文件。
//
// 返回值:
//
//	如果转换过程中遇到任何错误，则返回错误。
func (vp *Processor) Splat() error {
	// 尝试在指定的工作路径中找到.ply文件。
	plyPath, err := findPlyPath(vp.Iterations, vp.OutputFolder)
	if err != nil {
		// 如果找不到.ply文件，返回错误。
		return fmt.Errorf("fail to find .ply file: %v", err)
	}

	// 构建执行Python转换脚本的命令。
	// 使用VideoProcessor实例中指定的Python解释器。
	cmd := exec.Command("python", "3DGS/gaussian-splatting/splat.py", plyPath)
	// 添加环境变量以确保Python脚本可以找到所需的库。
	cmd.Env = append(os.Environ(), fmt.Sprintf("PYTHONPATH=%s", "3DGS/gaussian-splatting/envs/gaussian_splatting"))

	// 执行命令并检查是否有错误发生。
	if err := cmd.Run(); err != nil {
		// 如果执行命令时出错，返回错误。
		return fmt.Errorf("fail to convert to splat file:%w", err)
	}

	// 如果一切顺利，返回nil表示没有发生错误。
	return nil
}

func findPlyPath(iterations, filePath string) (string, error) {
	if _, err := os.Stat(filePath + "/point_cloud/iteration_" + iterations + "/point_cloud.ply"); err != nil {
		return "", fmt.Errorf("fail to find .ply file: %v", err)
	}
	return filePath + "/point_cloud/iteration_" + iterations + "/point_cloud.ply", nil
}

// getParentID 返回父作品的ID
// 如果origin.Parent为nil或origin.Parent.ID为nil，则返回origin.ID的指针
// 否则返回origin.Parent.ID
func getParentID(origin *models.Work) *uint {
	if origin.Parent == nil {
		return &origin.ID
	}
	return origin.ParentID
}

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

func QueryWorkInfo(userID uint) ([]WorkInfo, error) {
	var workInfos []WorkInfo
	if err := config.Conf.DB.Model(&models.Work{}).
		Where("user_id=?", userID).
		Select("id as work_id, work_name, status").
		Scan(&workInfos).Error; err != nil {

		return nil, err
	}

	return workInfos, nil
}

func QueryWork(workID uint) (models.Work, error) {
	var origin models.Work
	if err := config.Conf.DB.Where("id = ?", workID).First(&origin).Error; err != nil {
		return models.Work{}, err
	}

	return origin, nil
}

func NewWork(transferInfo TransferInfo, origin models.Work) (models.Work, error) {
	var work models.Work
	err := config.Conf.DB.Transaction(func(tx *gorm.DB) error {
		work = models.Work{
			UserID:     origin.UserID,
			WorkName:   transferInfo.WorkName,
			Status:     "processing",
			IsPublic:   transferInfo.IsPublic,
			Iterations: transferInfo.Iterations,
			ParentID:   getParentID(&origin),
		}
		return tx.Create(&work).Error
	})
	if err != nil {
		return models.Work{}, err
	}

	return work, nil
}

func PrepareData(parentID uint) (string, error) {
	dataPath := filepath.Join("transfer_tmp", uuid.NewString())
	err := database.RetrieveFromBucketWithDir(fmt.Sprintf("%d", parentID), dataPath)
	if err != nil {
		return "", err
	}
	dataPath += fmt.Sprintf("/%d", parentID)
	return dataPath, nil
}

func SaveModel(outputFolder, iterations string, workID uint) error {
	modelPath := filepath.Join(outputFolder, fmt.Sprintf("point_cloud/iteration_%s/point_cloud.ply", iterations))
	model, err := os.Open(modelPath)
	if err != nil {
		return err
	}
	defer model.Close()

	targetModelPath := fmt.Sprintf("%d/point_cloud/model.ply", workID)
	if err := database.StoreInBucket(targetModelPath, model); err != nil {
		return err
	}
	return nil
}

func SaveCheckpoint(outputFolder, iterations string, workID uint) error {
	chkpntPath := filepath.Join(outputFolder, fmt.Sprintf("chkpnt%s.pth", iterations))
	chkpnt, err := os.Open(chkpntPath)
	if err != nil {
		return err
	}
	defer chkpnt.Close()

	targetChkpntPath := fmt.Sprintf("%d/checkpoint/chkpnt.pth", workID)
	if err := database.StoreInBucket(targetChkpntPath, chkpnt); err != nil {
		return err
	}
	return nil
}

// updateWorkStatus 更新工作的状态。
// 参数:
//
//	workID - 工作的唯一标识符。
//	status - 工作的新状态。
//	errorLog - 工作执行过程中遇到的错误日志。
//	startTime - 工作开始的时间。
//
// 返回值:
//
//	如果更新过程中发生错误，则返回错误。
func UpdateWorkStatus(workID uint, status, errorLog string, startTime time.Time) error {
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

func ProcessTasks(ctx context.Context, TaskQueue chan *websocket.Task) {
	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, processor exiting")
			return
		case task, ok := <-TaskQueue:
			if !ok {
				log.Println("Task queue channel closed")
				return
			}
			if task != nil {
				// 处理任务的代码保持不变
				go func(t *websocket.Task) {
					var result string
					var err error

					switch t.Type {
					case "init_model":
						result, err = processInitModelTask(t)
					case "transfer":
						result, err = processTransferTask(t)
					default:
						result = "unknown_task_type"
						err = fmt.Errorf("unknown task type: %s", t.Type)
					}

					if config.Conf.Hub != nil {
						message := websocket.Message{
							Status:  result,
							Message: result,
							Type:    t.Type,
							WorkID:  t.WorkID,
							Time:    time.Now(),
						}
						if err != nil {
							message.Error = err.Error()
						}
						config.Conf.Hub.BroadcastToUser(t.UserID, message)
					}
				}(task)
			}
		}
	}
}

func processInitModelTask(task *websocket.Task) (string, error) {
	initInfo := task.Data.(InitInfo)

	// 从存储桶中检索视频文件
	videoPath := fmt.Sprintf("videos/%d.mp4", initInfo.VideoID)
	videoPath, err := database.RetrieveFromBucket(videoPath)
	if err != nil {
		UpdateWorkStatus(task.WorkID, "failed", fmt.Sprintf("fail to retrieve video:%v", err), time.Now())
		return "failed", fmt.Errorf("fail to find video:%v", err)
	}
	defer os.RemoveAll(filepath.Dir(videoPath))

	// 创建处理器实例
	processor, err := NewProcessor(initInfo.Iterations)
	if err != nil {
		UpdateWorkStatus(task.WorkID, "failed", fmt.Sprintf("fail to train the model:%v", err), time.Now())
		return "failed", fmt.Errorf("fail to train the model:%v", err)
	}

	startTime := time.Now()
	// 提取视频帧
	if err := processor.RunFfmpeg(videoPath); err != nil {
		UpdateWorkStatus(task.WorkID, "failed", fmt.Sprintf("fail to generate pics:%v", err), startTime)
		return "failed", fmt.Errorf("fail to generate pics:%v", err)
	}

	dataPath := filepath.Dir(videoPath)
	// 运行COLMAP进行相机参数估计
	if err := processor.RunColmap(dataPath); err != nil {
		UpdateWorkStatus(task.WorkID, "failed", fmt.Sprintf("fail to estimate the camera:%v", err), startTime)
		return "failed", fmt.Errorf("fail to estimate the camera:%v", err)
	}

	// 执行三维重建
	if err := processor.Reconstruction(dataPath); err != nil {
		UpdateWorkStatus(task.WorkID, "failed", fmt.Sprintf("fail to reconstruction:%v", err), startTime)
		return "failed", fmt.Errorf("fail to reconstruction:%v", err)
	}

	// 将处理结果存储到存储桶
	if err := database.StoreInBucketWIthDir(fmt.Sprintf("%d", task.WorkID), dataPath+"/undistorted"); err != nil {
		UpdateWorkStatus(task.WorkID, "failed", fmt.Sprintf("fail to store data:%v", err), startTime)
		return "failed", fmt.Errorf("fail to store data:%v", err)
	}

	// 获取并存储点云模型文件
	if err := SaveModel(processor.OutputFolder, initInfo.Iterations, task.WorkID); err != nil {
		UpdateWorkStatus(task.WorkID, "failed", fmt.Sprintf("fail to save model:%v", err), startTime)
		return "failed", fmt.Errorf("fail to save model:%v", err)
	}
	// 获取并存储检查点文件
	if err := SaveCheckpoint(processor.OutputFolder, initInfo.Iterations, task.WorkID); err != nil {
		UpdateWorkStatus(task.WorkID, "failed", fmt.Sprintf("fail to save checkpoint:%v", err), startTime)
		return "failed", fmt.Errorf("fail to save checkpoint:%v", err)
	}

	// 更新作品状态为完成
	UpdateWorkStatus(task.WorkID, "completed", "", startTime)

	return "completed", nil
}

func processTransferTask(task *websocket.Task) (string, error) {
	taskData := task.Data.(struct {
		TransferInfo   TransferInfo `json:"transfer_info"`
		StyleImagePath string       `json:"style_image_path"`
		WorkID         uint         `json:"work_id"`
	})

	work, err := QueryWork(task.WorkID)
	// 从存储桶检索原始作品数据到本地
	dataPath, err := PrepareData(*work.ParentID)
	if err != nil {
		UpdateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to stylize:%v", err), time.Now())
		return "failed", fmt.Errorf("fail to find parent:%v", err)
	}
	defer os.RemoveAll(filepath.Dir(dataPath))

	// 初始化处理器
	processor, err := NewProcessor(taskData.TransferInfo.Iterations)
	if err != nil {
		UpdateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to stylize:%v", err), time.Now())
		return "failed", fmt.Errorf("fail to train the model:%v", err)
	}

	// 执行风格迁移处理
	startTime := time.Now()
	if err := processor.Stylize(dataPath, taskData.StyleImagePath); err != nil {
		UpdateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to stylize:%v", err), startTime)
		return "failed", fmt.Errorf("fail to stylize:%v", err)
	}

	// 保存处理后的点云模型文件到存储桶
	if err := SaveModel(processor.OutputFolder, taskData.TransferInfo.Iterations, work.ID); err != nil {
		UpdateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to stylize:%v", err), startTime)
		return "failed", fmt.Errorf("fail to save model:%v", err)
	}

	// 获取并存储检查点文件
	if err := SaveCheckpoint(processor.OutputFolder, taskData.TransferInfo.Iterations, work.ID); err != nil {
		UpdateWorkStatus(work.ID, "failed", fmt.Sprintf("fail to stylize:%v", err), startTime)
		return "failed", fmt.Errorf("fail to save checkpoint:%v", err)
	}

	// 更新状态为完成
	UpdateWorkStatus(work.ID, "completed", "", startTime)

	return "completed", nil
}
