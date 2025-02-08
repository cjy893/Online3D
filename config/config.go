// config.go
package config

import (
	"os"

	"gorm.io/gorm"
)

type AppConfig struct {
	DB         *gorm.DB
	JWTSecret  string
	UploadPath string
	PythonPath string
	OutputPath string
	ServerPort string
}

var Conf AppConfig

func LoadConfig() {
	Conf = AppConfig{
		JWTSecret:  os.Getenv("JWT_SECRET"),
		UploadPath: getEnvWithDefault("UPLOAD_PATH", "./uploads"),
		PythonPath: os.Getenv("PYTHON_PATH"),
		OutputPath: getEnvWithDefault("OUTPUT_PATH", "./outputs"),
		ServerPort: "8080",
	}
}

func getEnvWithDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
