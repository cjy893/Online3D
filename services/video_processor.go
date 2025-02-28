package services

import (
	"bufio"
	"bytes"
	"fmt"
	"myapp/config"
	"myapp/models"
	"myapp/utils"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type VideoProcessor struct {
	TrainerPath       string
	PythonPath        string
	BaseOutputFolder  string
	OutputFolder      string
	PythonInterpreter string
	FPS               int
}

// NewVideoProcessor 创建并初始化一个新的VideoProcessor实例。
// 该函数无需参数。
// 返回值是一个指向VideoProcessor实例的指针，以及一个错误值（如果有）。
func NewVideoProcessor() (*VideoProcessor, error) {
	// 获取项目根目录的路径。
	projectRoot := utils.GetProjectRoot()

	// 拼接项目根目录与训练脚本相对路径，得到完整的训练脚本路径。
	trainerPath := utils.SafeJoin(projectRoot, "3DGS/gaussian-splatting/train_video.py")
	// 如果训练脚本路径为空，则返回错误。
	if trainerPath == "" {
		return nil, fmt.Errorf("invalid trainer path")
	}

	// 返回一个新的VideoProcessor实例，包含了一系列预设的属性值。
	return &VideoProcessor{
		TrainerPath:       trainerPath,
		PythonPath:        utils.SafeJoin(projectRoot, "3DGS/gaussian-splatting/envs/gaussian_splatting"),
		BaseOutputFolder:  utils.SafeJoin(projectRoot, "output"),
		OutputFolder:      "",
		PythonInterpreter: "C:/Users/Administrator/anaconda3/envs/gaussian_splatting/python.exe",
		FPS:               2,
	}, nil
}

// ProcessVideo 处理视频文件。
// 该方法负责执行视频的训练过程，并将输出结果保存在安全的输出目录中。
// 参数:
//
//	video: 指向待处理的视频模型的指针。
//	processor: 指向当前视频处理器的指针，用于记录和操作处理过程的输出目录。
//
// 返回值:
//
//	如果处理过程中发生错误，则返回错误。
func (vp *VideoProcessor) ProcessVideo(video *models.Video, processor *VideoProcessor) error {
	// 生成唯一输出目录
	outputFolder := utils.SafeJoin(vp.BaseOutputFolder, "output")

	// 执行训练命令
	var err error
	processor.OutputFolder, err = vp.runTraining(video.FilePath, outputFolder)
	if err != nil {
		return err
	}
	// 注意：此处不再直接更新数据库，由外层统一处理状态
	return nil
}

// runTraining 运行训练程序以处理指定路径的视频。
// 该函数接受视频路径和输出文件夹路径作为参数。
// 它返回训练过程中生成的输出路径或者错误信息（如果有）。
func (vp *VideoProcessor) runTraining(videoPath, outputFolder string) (string, error) {
	// 获取上传路径的绝对路径，用于后续的路径验证。
	uploadPathAbs, _ := filepath.Abs(config.Conf.UploadPath)

	// 检查视频路径是否为绝对路径并且以上传路径开头。
	if !filepath.IsAbs(videoPath) || !strings.HasPrefix(videoPath, uploadPathAbs) {
		return "", fmt.Errorf("invalid video path")
	}

	// 构建运行训练脚本的命令。
	cmd := exec.Command(vp.PythonInterpreter, vp.TrainerPath, "--video", videoPath)

	// 添加PYTHONPATH环境变量以确保脚本能找到所需的模块。
	cmd.Env = append(os.Environ(), fmt.Sprintf("PYTHONPATH=%s", vp.PythonPath))

	// 准备缓冲区以存储命令的输出。
	var stdoutBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = os.Stderr

	// 打印训练开始的信息。
	fmt.Printf("Starting training process for video: %s\n", videoPath)

	// 执行命令并处理错误（如果有）。
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("training failed: %w", err)
	}

	// 解析输出路径
	scanner := bufio.NewScanner(&stdoutBuf)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "Output folder:") {
			parts := strings.Split(scanner.Text(), ": ")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1])[0:19], nil
			}
		}
	}
	// 如果未找到特定的输出路径，则返回提供的输出文件夹路径。
	return outputFolder, nil
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
func (vp *VideoProcessor) Splat(workPath string) error {
	// 尝试在指定的工作路径中找到.ply文件。
	plyPath, err := findPlyPath(workPath)
	if err != nil {
		// 如果找不到.ply文件，返回错误。
		return fmt.Errorf("fail to find .ply file: %v", err)
	}

	// 生成.splat文件的路径，通过替换.ply文件的扩展名实现。
	splatPath := strings.Split(plyPath, ".")[0]
	splatPath += ".splat"

	// 构建执行Python转换脚本的命令。
	// 使用VideoProcessor实例中指定的Python解释器。
	cmd := exec.Command(vp.PythonInterpreter, "web/convert.py", plyPath, "--output", splatPath)
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
