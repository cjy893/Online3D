package myBase

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

const dsn = "root:@1919810ysxB@tcp(localhost:3306)/test?charset=utf8mb4&loc=PRC&parseTime=true"

func Regist(name, account, password string) (bool, string) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logrus.Error(err)
		return false, "加密错误"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		logrus.Error(err)
		return false, "MySQL连接错误"
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logrus.Error(err)
		return false, "数据库连接错误"
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

func Login(account, password string) (bool, string) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		logrus.Error(err)
		return false, "哈希生成错误"
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		logrus.Error(err)
		return false, "MySQL连接错误"
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logrus.Error(err)
		return false, "数据库连接错误"
	}

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

	if realPassword != string(hashedPassword) {
		logrus.Error(err)
		return false, "密码错误"
	}
	return true, "登陆成功"
}
