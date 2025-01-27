package Online3D

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const dsn = "root:%401919810ysxB@tcp(localhost:3306)/test?charset=utf8mb4&loc=PRC&parseTime=true"

// 初始化数据库连接池
func InitDB() (*sql.DB, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		logrus.Errorf("初始化连接池失败: %v", err)
		return nil, err
	}
	if err := db.Ping(); err != nil {
		logrus.Errorf("数据库连接失败: %v", err)
		return nil, err
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	return db, nil
}

// 注册用户账号操作
func Regist(db *sql.DB, name, password string) error {
	//对用户密码进行bcrypt加密，数据库的密码会以哈希的方式储存
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logrus.Errorf("加密失败:%v", err)
		return err
	}

	//初始化事务，并设置结束回滚
	tx, err := db.Begin()
	if err != nil {
		logrus.Errorf("注册失败:%v", err)
		return err
	}
	defer tx.Rollback()

	//使用参数化查询，查询是否已经存在同名账号
	stmt, err := tx.Prepare("SELECT 1 FROM Accounts WHERE name = ?")
	if err != nil {
		logrus.Errorf("查询账号失败:%v", err)
		return err
	}
	defer stmt.Close()

	var exists bool
	err = stmt.QueryRow(name).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		logrus.Errorf("查询账号失败: %v", err)
		return fmt.Errorf("服务暂不可用")
	}
	if exists {
		return fmt.Errorf("账号 %s 已存在", name)
	}

	//生成账号
	var builder strings.Builder
	builder.WriteString(name)
	builder.WriteString("@Online3D.com")
	account := builder.String()

	//注册账号
	stmt, err = tx.Prepare("INSERT INTO Accounts (name, account, password) VALUES (?, ?, ?)")
	if err != nil {
		logrus.Errorf("注册失败:%v", err)
		return err
	}

	_, err = stmt.Exec(name, account, hashedPassword)
	if err != nil {
		logrus.Errorf("注册失败:%v", err)
		return err
	}

	//如果事务提交失败，则返回错误并回滚
	if err := tx.Commit(); err != nil {
		logrus.Errorf("提交事务失败:%v", err)
		return err
	}

	return nil
}

// 登录用户账号操作
func Login(db *sql.DB, account, password string) error {
	stmt, err := db.Prepare("SELECT password FROM Accounts WHERE account = ?")
	if err != nil {
		logrus.Errorf("查询失败:%v", err)
		return err
	}
	defer stmt.Close()

	//查询账号真正的密码，如果不存在账号，则返回错误
	var realPassword string
	err = stmt.QueryRow(account).Scan(&realPassword)
	if err != nil {
		if err == sql.ErrNoRows {
			logrus.Errorf("不存在该账号:%v", err)
			return err
		}
		logrus.Errorf("登录失败:%v", err)
		return err
	}

	//比较真正的密码与输入密码的哈希，判断密码是否正确
	err = bcrypt.CompareHashAndPassword([]byte(realPassword), []byte(password))
	if err != nil {
		logrus.Errorf("密码错误:%v", err)
		return err
	}
	return nil
}

// 添加作品
func AddWork(db *sql.DB, account, workName, workId string) error {
	tx, err := db.Begin()
	if err != nil {
		logrus.Errorf("作品信息记录失败:%v", err)
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare("INSERT INTO Works (account, workname, workid) VALUE (?, ?, ?)")
	if err != nil {
		logrus.Errorf("作品信息记录失败:%v", err)
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(account, workName, workId)
	if err != nil {
		logrus.Errorf("作品信息记录失败:%v", err)
		return err
	}
	return nil
}

func QueryWork(db *sql.DB, account, workName string) (string, error) {
	stmt, err := db.Prepare("SELECT workid FROM Works WHERE account = ? AND workname = ?")
	if err != nil {
		logrus.Errorf("作品信息查询错误:%v", err)
		return "", err
	}
	defer stmt.Close()

	var workId string
	err = stmt.QueryRow(account, workName).Scan(&workId)
	if err != nil {
		if err == sql.ErrNoRows {
			logrus.Errorf("不存在该作品:%v", err)
			return "", err
		}
		logrus.Errorf("作品信息查询错误:%v", err)
		return "", err
	}
	return workId, nil
}
