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

func StoreInBucketWIthDir(id, dir string) error {
	baseDir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	return filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error accessing path %s: %w", path, err)
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return fmt.Errorf("error getting relative path for %s: %w", path, err)
		}

		// 打开文件
		file, err := os.Open(path)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", path, err)
		}
		defer file.Close()

		// 生成唯一ID（可以基于文件路径或使用UUID）
		id += "/" + relPath

		// 使用现有函数上传文件
		if err := StoreInBucket(id, file); err != nil {
			return fmt.Errorf("failed to upload file %s: %w", path, err)
		}

		fmt.Printf("Successfully uploaded %s as %s\n", path, id)
		return nil
	})
}

func RetrieveFromBucketWithDir(id, localPath string) error {
	if err := os.MkdirAll(localPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	objectCh := config.Conf.MINIO.ListObjects(context.Background(), config.Conf.BucketName, minio.ListObjectsOptions{
		Prefix:    id,
		Recursive: true,
	})

	for obj := range objectCh {
		if obj.Err != nil {
			return fmt.Errorf("error listing objects: %w", obj.Err)
		}

		relPath := obj.Key
		localFilePath := filepath.Join(localPath, filepath.FromSlash(relPath))

		localDir := filepath.Dir(localFilePath)
		if err := os.MkdirAll(localDir, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
		}

		localFile, err := os.Create(localFilePath)
		if err != nil {
			return fmt.Errorf("failed to create temporary file: %w", err)
		}

		// 获取对象流
		objStream, err := config.Conf.MINIO.GetObject(context.Background(), config.Conf.BucketName, obj.Key, minio.GetObjectOptions{})
		if err != nil {
			localFile.Close()
			return fmt.Errorf("failed to get object %s: %w", obj.Key, err)
		}

		// 复制数据到临时文件
		bufWriter := bufio.NewWriterSize(localFile, 64*1024*1024)               // 64MB缓冲区
		_, err = io.CopyBuffer(bufWriter, objStream, make([]byte, 4*1024*1024)) // 4MB buffer
		objStream.Close()
		localFile.Close()
		bufWriter.Flush()

		if err != nil {
			os.RemoveAll(localDir)
			return fmt.Errorf("failed to save object content: %w", err)
		}
	}

	return nil
}

func StoreInBucket(id string, file *os.File) error {
	//1. 设置大文件分块参数（64MB分块）
	ext := filepath.Ext(file.Name())
	partSize := uint64(64 * 1024 * 1024)
	opts := minio.PutObjectOptions{
		ContentType:      getContentType(ext), //2. 自定义ContentType处理
		PartSize:         partSize,
		DisableMultipart: false,
	}

	// 3. 使用流式上传避免内存溢出
	if _, err := file.Seek(0, 0); err != nil {
		return fmt.Errorf("fail to reset file pointer:%w", err)
	}

	// 4. 通过Reader接口实现流式上传
	_, err := config.Conf.MINIO.PutObject(
		context.Background(),
		config.Conf.BucketName,
		id,
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

	fileuuid := uuid.New().String()
	fileName := "temp/" + fileuuid + "/" + id
	// 创建保存目录（自动处理多级目录）
	if err := os.MkdirAll("temp/"+fileuuid, 0755); err != nil {
		return "", fmt.Errorf("failed to create directories: %w", err)
	}

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
	case ".jpg":
		return "image/jpeg"
	default:
		return "application/octet-stream"
	}
}

// DeleteFolderFromBucket 删除bucket中的整个文件夹
func DeleteFolderFromBucket(id string) error {
	// 列出所有匹配前缀的对象
	objectCh := config.Conf.MINIO.ListObjects(context.Background(), config.Conf.BucketName, minio.ListObjectsOptions{
		Prefix:    id + "/",
		Recursive: true,
	})

	// 收集要删除的对象键
	var objectsToDelete []minio.ObjectInfo
	for object := range objectCh {
		if object.Err != nil {
			return fmt.Errorf("error listing objects: %w", object.Err)
		}
		objectsToDelete = append(objectsToDelete, object)
	}

	// 批量删除对象
	if len(objectsToDelete) > 0 {
		objectsCh := make(chan minio.ObjectInfo)
		go func() {
			defer close(objectsCh)
			for _, obj := range objectsToDelete {
				objectsCh <- obj
			}
		}()

		errorCh := config.Conf.MINIO.RemoveObjects(context.Background(), config.Conf.BucketName, objectsCh, minio.RemoveObjectsOptions{})

		// 检查是否有删除错误
		for e := range errorCh {
			if e.Err != nil {
				return fmt.Errorf("failed to delete object %s: %w", e.ObjectName, e.Err)
			}
		}
	}

	fmt.Printf("Successfully deleted folder %s from bucket\n", id)
	return nil
}
