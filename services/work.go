package services

import (
	"fmt"
	"myapp/config"
	"myapp/models"
	"myapp/utils"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/google/uuid"
)

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
		"--resolution 1")

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 打印训练开始的信息。
	fmt.Printf("Starting stylizing process for video: %s\n", dataPath)

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
func GetParentID(origin *models.Work) *uint {
	if origin.Parent == nil {
		return &origin.ID
	}
	return origin.ParentID
}
