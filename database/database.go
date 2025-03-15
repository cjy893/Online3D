package database

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"myapp/config"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
)

func StoreInBucket(id, ftype string, file *os.File) error {
	// 1. 添加文件格式校验
	ext := filepath.Ext(file.Name())
	if ext != ".mp4" && ext != ".splat" {
		return fmt.Errorf("unsupported file format: %s, only .mp4 and .splat allowed", ext)
	}

	// 2. 设置大文件分块参数（64MB分块）
	partSize := uint64(64 * 1024 * 1024)
	opts := minio.PutObjectOptions{
		ContentType:      getContentType(ext), // 3. 自定义ContentType处理
		PartSize:         partSize,
		DisableMultipart: false,
	}

	// 4. 使用流式上传避免内存溢出
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("fail to reset file pointer:%w", err)
	}

	// 5. 通过Reader接口实现流式上传
	_, err := config.Conf.MINIO.PutObject(
		context.Background(),
		config.Conf.BucketName,
		ftype+id+ext,
		file,
		-1, // 使用-1让minio自动检测文件大小
		opts,
	)
	if err != nil {
		return fmt.Errorf("fail to upload file:%w", err)
	}
	return nil
}

func RetrieveFromBucket(id string) (string, error) {
	// 检查对象是否存在
	_, err := config.Conf.MINIO.StatObject(context.Background(), config.Conf.BucketName, id, minio.StatObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("object %s not found: %w", id, err)
	}

	// 创建保存目录（自动处理多级目录）
	if err := os.MkdirAll("temp/"+id, 0755); err != nil {
		return "", fmt.Errorf("failed to create directories: %w", err)
	}

	fileuuid := uuid.New().String()
	fileName := "temp/" + fileuuid + "/" + id
	// 创建本地文件
	file, err := os.Create(fileName)
	if err != nil {
		return "", fmt.Errorf("failed to create local file: %w", err)
	}
	defer file.Close()

	// 获取对象流
	obj, err := config.Conf.MINIO.GetObject(context.Background(), config.Conf.BucketName, id, minio.GetObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get object: %w", err)
	}
	defer obj.Close()

	bufWriter := bufio.NewWriterSize(file, 64*1024*1024) // 64MB缓冲区
	defer bufWriter.Flush()

	//带进度监控的拷贝
	if _, err := io.CopyBuffer(bufWriter, obj, make([]byte, 4*1024*1024)); err != nil { // 4MB buffer
		return "", fmt.Errorf("failed to save object content: %w", err)
	}

	return fileName, nil
}

func getContentType(ext string) string {
	switch ext {
	case ".mp4":
		return "video/mp4"
	case ".splat":
		return "application/octet-stream" // 自定义类型
	default:
		return "application/octet-stream"
	}
}
