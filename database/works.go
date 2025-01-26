package myBase

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
)

func addWork(account, workName, workId string) (bool, string) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		logrus.Error(err)
		return false, "MySQL连接失败"
	}
	defer db.Close()

	stmt, err := db.Prepare("INSERT INTO Works (account, workname, workid) VALUE (?, ?, ?)")
	if err != nil {
		logrus.Error(err)
		return false, "作品信息记录失败"
	}
	defer stmt.Close()

	_, err = stmt.Exec(account, workName, workId)
	if err != nil {
		logrus.Error(err)
		return false, "作品信息记录失败"
	}
	return true, "作品信息记录成功"
}
