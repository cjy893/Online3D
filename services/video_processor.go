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

func NewVideoProcessor() (*VideoProcessor, error) {
	projectRoot := utils.GetProjectRoot()

	trainerPath := utils.SafeJoin(projectRoot, "3DGS/gaussian-splatting/train_video.py")
	if trainerPath == "" {
		return nil, fmt.Errorf("invalid trainer path")
	}

	return &VideoProcessor{
		TrainerPath:       trainerPath,
		PythonPath:        utils.SafeJoin(projectRoot, "3DGS/gaussian-splatting/envs/gaussian_splatting"),
		BaseOutputFolder:  utils.SafeJoin(projectRoot, "output"),
		OutputFolder:      "",
		PythonInterpreter: "C:/Users/Administrator/anaconda3/envs/gaussian_splatting/python.exe",
		FPS:               2,
	}, nil
}

func (vp *VideoProcessor) Process(video *models.Video, processor *VideoProcessor) error {
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

func (vp *VideoProcessor) runTraining(videoPath, outputFolder string) (string, error) {
	uploadPathAbs, _ := filepath.Abs(config.Conf.UploadPath)
	if !filepath.IsAbs(videoPath) || !strings.HasPrefix(videoPath, uploadPathAbs) {
		return "", fmt.Errorf("invalid video path")
	}

	cmd := exec.Command(vp.PythonInterpreter, vp.TrainerPath, "--video", videoPath)
	cmd.Env = append(os.Environ(), fmt.Sprintf("PYTHONPATH=%s", vp.PythonPath))
	var stdoutBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = os.Stderr

	fmt.Printf("Starting training process for video: %s\n", videoPath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("training failed: %w", err)
	}

	// 解析输出路径
	scanner := bufio.NewScanner(&stdoutBuf)
	for scanner.Scan() {
		if strings.Contains(scanner.Text(), "Output folder:") {
			parts := strings.Split(scanner.Text(), ": ")
			if len(parts) > 1 {
				return strings.TrimSpace(parts[1]), nil
			}
		}
	}
	return outputFolder, nil
}
