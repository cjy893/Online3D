package config

import (
	"myapp/models"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func ConnectDB() {
	// 依赖已加载的配置（需先调用 LoadConfig）
	dsn := os.Getenv("DB_DSN") // 如果 DB_DSN 未在 LoadConfig 中加载，需补充到 AppConfig
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}

	db.AutoMigrate(&models.User{}, &models.Video{}, &models.Work{})
	Conf.DB = db // 将数据库实例存入 AppConfig
}
