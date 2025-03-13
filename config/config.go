// config.go
package config

import (
	"myapp/models"
	"os"

	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type AppConfig struct {
	DB         *gorm.DB
	MINIO      *minio.Client
	JWTSecret  string
	BucketName string
	AccessKey  string
	SecretKey  string
	EndPoint   string
	PythonPath string
	SplatPath  string
	ServerPort string
	DSN        string
}

var Conf AppConfig

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		panic("environment loading error")
	}
	Conf = AppConfig{
		JWTSecret:  os.Getenv("JWT_SECRET"),
		BucketName: os.Getenv("BUCKET_NAME"),
		AccessKey:  os.Getenv("ACCESS_KEY"),
		SecretKey:  os.Getenv("SECRET_KEY"),
		EndPoint:   os.Getenv("END_POINT"),
		PythonPath: os.Getenv("PYTHON_PATH"),
		SplatPath:  os.Getenv("SPLAT_PATH"),
		ServerPort: "8080",
		DSN:        os.Getenv("DB_DSN"),
	}

	db, err := gorm.Open(mysql.Open(Conf.DSN), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	if err := db.AutoMigrate(&models.User{}, &models.Video{}, &models.Work{}); err != nil {
		panic("Database migration failed: " + err.Error())
	}
	Conf.DB = db // 将数据库实例存入 AppConfig

	minioClient, err := minio.New(Conf.EndPoint, &minio.Options{
		Creds: credentials.NewStaticV4(Conf.AccessKey, Conf.SecretKey, ""),
	})
	if err != nil {
		panic("failed to connect minio: " + err.Error())
	}

	Conf.MINIO = minioClient
}
