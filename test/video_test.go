package test

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"myapp/config"
	"myapp/database"
	"myapp/handlers"
	"myapp/models"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestVideoUpload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	if config.Conf.DB == nil {
		// 使用测试数据库配置
		dsn := "root:@1919810ysxB@tcp(localhost:3306)/online3d?charset=utf8mb4&loc=PRC&parseTime=true&allowNativePasswords=true"
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			t.Fatal("Failed to connect to test database:", err)
		}
		config.Conf.DB = db
	}

	minioClient, err := minio.New("objectstorageapi.hzh.sealos.run", &minio.Options{
		Creds: credentials.NewStaticV4("swsqe2yx", "vqgztfbr4xp4vtxd", ""),
	})
	if err != nil {
		t.Fatal("Failed to initialize MinIO client:", err)
	}
	config.Conf.MINIO = minioClient
	config.Conf.BucketName = "swsqe2yx-online3d"

	// tempFile, err := os.CreateTemp("./", "test_video.mp4")
	// assert.NoError(t, err, "Failed to create temporary video file")
	// defer os.Remove(tempFile.Name())

	// _, err = tempFile.Write([]byte("fake video content"))
	// assert.NoError(t, err, "Failed to write video content to temporary file")
	// tempFile.Close()

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	videoFile, err := os.Open("video/test_video.mp4")
	assert.NoError(t, err, "Failed to open temp file")
	defer videoFile.Close()

	videoPart, err := writer.CreateFormFile("video", "test_video.mp4")
	assert.NoError(t, err, "Failed to create video part")
	_, err = io.Copy(videoPart, videoFile)
	assert.NoError(t, err, "Failed to write video content to video part")

	err = writer.WriteField("title", "Test Video")
	assert.NoError(t, err, "Failed to write title field")

	err = writer.WriteField("is_public", "true")
	assert.NoError(t, err, "Failed to write is_public field")

	err = writer.Close()
	assert.NoError(t, err, "Failed to close multipart writer")

	// 创建测试上下文
	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/user/video/upload", body)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// 创建Gin上下文
	c, _ := gin.CreateTestContext(w)
	c.Request = req

	// 模拟认证中间件的行为：设置userID到上下文中
	// 创建一个假的用户ID用于测试
	c.Set("userID", uint(1))

	// 直接调用处理函数
	handlers.UploadVideo(c)

	respBody, _ := io.ReadAll(w.Body)
	assert.Equal(t, http.StatusCreated, w.Code, string(respBody))

	// 检查数据库中是否创建了视频记录
	var video models.Video
	result := config.Conf.DB.Where("title = ? AND user_id = ?", "Test Video", 1).First(&video)
	assert.NoError(t, result.Error, "Video should be saved in database")
	assert.Equal(t, true, video.IsPublic, "Video should be public")
	assert.Equal(t, 1, int(video.UserID), "Video should belong to test user")

	path, err := database.RetrieveFromBucket(fmt.Sprintf("%s%d%s", "videos/", video.ID, ".mp4"))
	assert.NoError(t, err)
	_, err = os.Stat(path)
	assert.NoError(t, err)

	// 清理测试数据
	// config.Conf.DB.Delete(&video)
	// var count int64
	// config.Conf.DB.Model(&models.Video{}).Where("id = ?", video.ID).Count(&count)
	// assert.NotEqual(t, int64(0), count, "Video should be deleted from database")
}
