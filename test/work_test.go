package test

import (
	"bytes"
	"io"
	"mime/multipart"
	"myapp/config"
	"myapp/handlers"
	"myapp/models"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestModelInit(t *testing.T) {
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

	body := new(bytes.Buffer)
	writer := multipart.NewWriter(body)

	err = writer.WriteField("id", "1")
	assert.NoError(t, err)

	err = writer.WriteField("work_name", "test_work")
	assert.NoError(t, err)

	err = writer.WriteField("is_public", "true")
	assert.NoError(t, err)

	err = writer.WriteField("iterations", "1000")
	assert.NoError(t, err)

	w := httptest.NewRecorder()
	req, err := http.NewRequest("POST", "/user/work/init", body)
	assert.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	c, _ := gin.CreateTestContext(w)
	c.Request = req

	c.Set("userID", uint(1))
	handlers.InitModel(c)

	respBody, _ := io.ReadAll(w.Result().Body)
	assert.Equal(t, http.StatusOK, w.Code, string(respBody))

	var work models.Work
	result := config.Conf.DB.Where("work_name=? AND video_id=?", "test_work", 1).First(&work)
	assert.NoError(t, result.Error, "work not found")
	assert.Equal(t, "completed", work.Status)
	assert.Equal(t, 1, int(work.UserID), "Work should belong to test user")

}

func TestWorkStylize(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/user/work/ai/transfer", handlers.TransferByAIAgent)

	reader := strings.NewReader(`{"image_url": "https://example.com/image.jpg"}`)

	req, err := http.NewRequest("POST", "/user/work/ai/transfer", reader)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
}
