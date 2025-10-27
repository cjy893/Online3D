package authService

import (
	"fmt"
	"myapp/config"
	"myapp/models"

	"golang.org/x/crypto/bcrypt"
)

func CreateUser(user *models.User) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user.Password = string(hashedPassword)
	return config.Conf.DB.Create(&user).Error
}

func CheckUser(user models.User) error {
	if err := config.Conf.DB.Where("account = ?", user.Account).First(&user).Error; err != nil {
		return fmt.Errorf("用户名已存在:%v", err)
	}
	if err := config.Conf.DB.Where("email = ?", user.Email).First(&user).Error; err != nil {
		return fmt.Errorf("邮箱已注册:%v", err)
	}
	return nil
}

func GetUserByIdentifier(identifier string) (models.User, error) {
	// 定义用户变量，用于存储从数据库中查询到的用户信息
	var user models.User
	// 根据提供的标识符（用户名或邮箱）查询用户信息
	if err := config.Conf.DB.Where("account = ? OR email = ?", identifier, identifier).First(&user).Error; err != nil {
		return models.User{}, fmt.Errorf("用户不存在:%v", err)
	}

	return user, nil
}
