package Online3D

import (
	"errors"
	"fmt"

	nativeMysql "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const dsn = "root:%401919810ysxB@tcp(localhost:3306)/test?charset=utf8mb4&loc=PRC&parseTime=true"

type Account struct {
	Name     string `gorm:"column:name;uniqueIndex;size:20"`
	Account  string `gorm:"column:name;uniqueIndex;size:33"`
	Password string `gorm:"column:password;size:60"`
}

func (Account) TableName() string {
	return "accounts"
}

func (a *Account) BeforeCreate(tx *gorm.DB) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(a.Password), bcrypt.DefaultCost)
	if err != nil {
		logrus.Errorf("加密失败:%v", err)
		return err
	}
	a.Password = string(hashed)
	return nil
}

type Work struct {
	Account  string `gorm:"column:account;primaryKey;size:33"`
	WorkName string `gorm:"column:workname;primaryKey;size:60"`
	WorkID   string `gorm:"column:workid;size:10"`
}

func (Work) TableName() string {
	return "works"
}

// 初始化数据库连接池
func InitDB() (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		logrus.Errorf("数据库连接失败:%v", err)
		return nil, err
	}

	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	return db, nil
}

// 注册用户账号操作
func Regist(db *gorm.DB, name, password string) error {
	// 生成标准账号格式
	account := fmt.Sprintf("%s@Online3D.com", name)

	newUser := Account{
		Name:     name,
		Account:  account,
		Password: password, // 钩子会自动加密
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&newUser).Error; err != nil {
			if isDuplicateKeyError(err) {
				logrus.Warnf("账号已存在: %s", account)
				return fmt.Errorf("账号已存在")
			}
			logrus.Errorf("注册失败: %v", err)
			return fmt.Errorf("注册失败")
		}
		return nil
	})
}

// 登录用户账号操作
func Login(db *gorm.DB, account, password string) error {
	var user Account
	err := db.Where("account = ?", account).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		logrus.Warnf("账号不存在: %s", account)
		return fmt.Errorf("账号不存在")
	}
	if err != nil {
		logrus.Errorf("登录查询失败: %v", err)
		return fmt.Errorf("系统错误")
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(password),
	); err != nil {
		logrus.Warnf("密码错误: %s", account)
		return fmt.Errorf("密码错误")
	}

	return nil
}

// 添加作品
func AddWork(db *gorm.DB, account, workName, workId string) error {
	return db.Transaction(func(tx *gorm.DB) error {
		work := Work{
			Account:  account,
			WorkName: workName,
			WorkID:   workId,
		}

		if err := tx.Create(&work).Error; err != nil {
			if isDuplicateKeyError(err) {
				logrus.Warnf("作品已存在: %s/%s", account, workName)
				return fmt.Errorf("作品已存在")
			}
			logrus.Errorf("创建作品失败: %v", err)
			return fmt.Errorf("系统错误")
		}
		return nil
	})
}

// 查询作品
func QueryWork(db *gorm.DB, account, workName string) (string, error) {
	var work Work
	err := db.Model(&Work{}).
		Where("account = ? AND workname = ?", account, workName).
		First(&work).
		Error

	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		logrus.Warnf("作品不存在: %s/%s", account, workName)
		return "", fmt.Errorf("作品不存在")
	case err != nil:
		logrus.Errorf("查询作品失败: %v", err)
		return "", fmt.Errorf("系统错误")
	}

	return work.WorkID, nil
}

// 辅助函数：判断是否是唯一键冲突错误
func isDuplicateKeyError(err error) bool {
	var mysqlErr *nativeMysql.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
