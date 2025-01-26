package myBase

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
)

func addWork(db *sql.DB, account, workName, workId string) (bool, string) {
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

func queryWork(db *sql.DB, account, workName string) (bool, string) {
	stmt, err := db.Prepare("SELECT workid FROM Works WHERE account = ? AND workname = ?")
	if err != nil {
		logrus.Error(err)
		return false, "作品信息查询错误"
	}
	defer stmt.Close()

	var workId string
	err = stmt.QueryRow(account, workName).Scan(&workId)
	if err != nil {
		logrus.Error(err)
		if err == sql.ErrNoRows {
			return false, "不存在此作品"
		}
		return false, "作品信息查询错误"
	}
	return true, workId
}
