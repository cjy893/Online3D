package myBase

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const dsn = "root:@1919810ysxB@tcp(localhost:3306)/test?charset=utf8mb4&loc=PRC&parseTime=true"

func initDB(db *sql.DB) (bool, string) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		logrus.Error(err)
		return false, "数据库连接池初始化失败"
	}
	if err := db.Ping(); err != nil {
		logrus.Error(err)
		return false, "数据库连接失败"
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	return true, "数据库连接成功"
}

func Regist(db *sql.DB, name, account, password string) (bool, string) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logrus.Error(err)
		return false, "加密失败"
	}

	stmt, err := db.Prepare("INSERT INTO Accounts (name, account, password) VALUES (?, ?, ?)")
	if err != nil {
		logrus.Error(err)
		return false, "注册失败"
	}
	defer stmt.Close()

	_, err = stmt.Exec(name, account, hashedPassword)
	if err != nil {
		logrus.Error(err)
		return false, "注册失败"
	}
	return true, "注册成功"
}

func Login(db *sql.DB, account, password string) (bool, string) {
	stmt, err := db.Prepare("SELECT password FROM Accounts WHERE account = ?")
	if err != nil {
		logrus.Error(err)
		return false, "查询错误"
	}
	defer stmt.Close()

	var realPassword string
	err = stmt.QueryRow(account).Scan(&realPassword)
	if err != nil {
		logrus.Error(err)
		if err == sql.ErrNoRows {
			return false, "不存在此账号"
		}
		return false, "查询错误"
	}

	err = bcrypt.CompareHashAndPassword([]byte(realPassword), []byte(password))
	if err != nil {
		logrus.Error(err)
		return false, "密码错误"
	}
	return true, "登陆成功"
}
