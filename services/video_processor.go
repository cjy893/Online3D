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
	TrainerPath      string
	ViewerPath       string
	PythonPath       string
	BaseOutputFolder string
	OutputFolder     string
}

func NewVideoProcessor() (*VideoProcessor, error) {
	projectRoot := utils.GetProjectRoot()

	trainerPath := utils.SafeJoin(projectRoot, "3DGS/gaussian-splatting/train_video.py")
	if trainerPath == "" {
		return nil, fmt.Errorf("invalid trainer path")
	}

	viewerPath := utils.SafeJoin(projectRoot, "3DGS/gaussian-splatting/SIBR_viewer.py")
	if viewerPath == "" {
		return nil, fmt.Errorf("invalid viewer path")
	}

	return &VideoProcessor{
		TrainerPath:      trainerPath,
		ViewerPath:       viewerPath,
		PythonPath:       utils.SafeJoin(projectRoot, "3DGS/gaussian-splatting/envs/gaussian_splatting"),
		BaseOutputFolder: utils.SafeJoin(projectRoot, "output"),
		OutputFolder:     "",
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

	cmd := exec.Command("python3", vp.TrainerPath, "--input", videoPath, "--output", outputFolder)
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
