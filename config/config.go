// config.go
package config

import (
	"myapp/models"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type AppConfig struct {
	DB         *gorm.DB
	JWTSecret  string
	UploadPath string
	PythonPath string
	OutputPath string
	SplatPath  string
	ViewerPath string
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
		UploadPath: os.Getenv("UPLOAD_PATH"),
		PythonPath: os.Getenv("PYTHON_PATH"),
		OutputPath: os.Getenv("OUTPUT_PATH"),
		SplatPath:  os.Getenv("SPLAT_PATH"),
		ViewerPath: os.Getenv("VIEWER_PATH"),
		ServerPort: "8080",
		DSN:        os.Getenv("DB_DSN"),
	}

	db, err := gorm.Open(mysql.Open(Conf.DSN), &gorm.Config{})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	if err := db.AutoMigrate(&models.User{}, &models.Video{}, &models.Work{}); err != nil {
		panic("Database migration failed: " + err.Error())
	}
	Conf.DB = db // 将数据库实例存入 AppConfig
}
