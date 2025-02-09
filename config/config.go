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
		UploadPath: getEnvWithDefault("UPLOAD_PATH", "./uploads"),
		PythonPath: os.Getenv("PYTHON_PATH"),
		OutputPath: getEnvWithDefault("OUTPUT_PATH", "./outputs"),
		ServerPort: "8080",
		DSN:        os.Getenv("DB_DSN"),
	}

	db, err := gorm.Open(mysql.Open(Conf.DSN), &gorm.Config{})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	db.AutoMigrate(&models.User{}, &models.Video{}, &models.Work{})
	Conf.DB = db // 将数据库实例存入 AppConfig
}

func getEnvWithDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
