package utils

import (
	"errors"
	"myapp/config"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// GenerateToken 生成用户认证令牌
// 参数:
//
//	userID uint - 用户ID，用于标识令牌的拥有者
//
// 返回值:
//
//	string - 生成的JWT令牌字符串
//	error - 错误信息，如果生成令牌失败
func GenerateToken(userID uint) (string, error) {
	// 创建一个新的JWT令牌，并附带指定的声明
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": userID,                                // "sub" 主题声明，存储用户ID
		"exp": time.Now().Add(time.Hour * 24).Unix(), // "exp" 过期时间声明，设置令牌在发行后24小时过期
	})

	// 使用HS256算法和配置的密钥对令牌进行签名并返回签名后的令牌字符串
	// config.Conf.JWTSecret 是应用的密钥，用于签名JWT令牌
	return token.SignedString([]byte(config.Conf.JWTSecret))
}

// ParseToken 解析JWT令牌，提取用户ID。
// 该函数接受一个JWT字符串作为输入，验证并解析该字符串以获取用户ID。
// tokenString: 待解析的JWT字符串。
// 返回值: uint类型的用户ID和一个错误（如果解析过程中发生错误）。
func ParseToken(tokenString string) (uint, error) {
	// 使用密钥解析JWT。
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// 返回用于签名验证的密钥。
		return []byte(config.Conf.JWTSecret), nil
	})
	if err != nil {
		// 如果解析过程中出现错误，返回错误。
		return 0, err
	}

	// 将令牌声明转换为MapClaims类型，并检查令牌的有效性。
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		// 如果转换失败或令牌无效，返回签名无效错误。
		return 0, jwt.ErrSignatureInvalid
	}

	// 从声明中提取"sub"字段，即用户ID，并进行类型断言。
	sub, ok := claims["sub"].(float64)
	if !ok {
		// 如果类型断言失败，返回错误。
		return 0, errors.New("invalid subject type")
	}

	// 将用户ID转换为uint类型，并返回。
	return uint(sub), nil
}
