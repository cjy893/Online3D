package handlers

import (
	"myapp/config"
	"myapp/models"
	"myapp/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Register 处理用户注册请求。
// 参数: c *gin.Context - Gin框架的上下文，用于处理HTTP请求和响应。
func Register(c *gin.Context) {
	var user models.User
	// 将请求体绑定到User模型，如果失败则返回错误信息。
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err})
		return
	}

	// 开启事务
	tx := config.Conf.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查用户名是否已存在（使用事务内的查询）
	var existing models.User
	if err := tx.Where("account = ?", user.Account).First(&existing).Error; err == nil {
		tx.Rollback()
		c.JSON(http.StatusConflict, gin.H{"error": "用户名已存在"})
		return
	}

	//检查邮箱是否已存在
	var existingEmail models.User
	if err := tx.Where("email = ?", user.Email).First(&existingEmail).Error; err == nil {
		tx.Rollback()
		c.JSON(http.StatusConflict, gin.H{"error": "邮箱已被注册"})
		return
	}

	// 哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "密码加密失败"})
		return
	}
	user.Password = string(hashedPassword)

	// 创建用户记录
	if err := tx.Create(&user).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户创建失败"})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "事务提交失败"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "用户注册成功"})
}

// Login 处理用户登录请求
// 参数: c *gin.Context 上下文，用于处理HTTP请求和响应
func Login(c *gin.Context) {
	// 定义凭据结构体，用于解析请求中的用户名/邮箱和密码
	var credentials struct {
		Identifier string `json:"identifier"` // 用户名或邮箱
		Password   string `json:"password"`
	}

	// 尝试从请求中解析JSON格式的凭据
	if err := c.ShouldBindJSON(&credentials); err != nil {
		// 如果解析失败，返回400错误响应
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 定义用户变量，用于存储从数据库中查询到的用户信息
	var user models.User
	// 根据提供的标识符（用户名或邮箱）查询用户信息
	if err := config.Conf.DB.Where("account = ? OR email = ?", credentials.Identifier, credentials.Identifier).First(&user).Error; err != nil {
		// 如果查询失败或用户不存在，返回401错误响应
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名/邮箱错误"})
		return
	}

	// 验证用户密码是否正确
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(credentials.Password)); err != nil {
		// 如果密码不正确，返回401错误响应
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	// 生成用户Token
	token, err := utils.GenerateToken(user.ID)
	if err != nil {
		// 如果Token生成失败，返回500错误响应
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Token generation failed"})
		return
	}

	// 返回生成的Token
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// DeleteUser 删除用户
// 该函数接收一个 gin.Context 参数，其中包含用户ID
// 函数通过数据库事务删除用户，并返回删除结果
func DeleteUser(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("userID")
	if !exists {
		// 如果用户未认证，返回未授权错误
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证的用户"})
		return
	}

	var user models.User
	if err := config.Conf.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		// 如果查询用户失败，返回内部服务器错误
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询用户失败"})
		return
	}

	// 开启事务
	tx := config.Conf.DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			// 如果发生异常，回滚事务
			tx.Rollback()
		}
	}()

	// 删除用户
	result := tx.Where("id = ?", userID).Delete(&models.User{})
	if result.Error != nil {
		// 如果删除过程中发生错误，回滚事务并返回错误信息
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除用户失败"})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		// 如果事务提交失败，回滚事务并返回错误信息
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "事务提交失败"})
		return
	}

	// 返回删除成功信息
	c.JSON(http.StatusOK, gin.H{"message": "用户删除成功"})
}
