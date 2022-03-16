package mysql

import (
	"BlueBell2/settings"
	"fmt"

	"go.uber.org/zap"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

var (
	db *sqlx.DB
)

// Init 初始化数据库
func Init(conf *settings.MysqlConfig) (err error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True",
		conf.User,
		conf.Password,
		conf.Host,
		conf.Port,
		conf.DbName,
	)
	db, err = sqlx.Connect("mysql", dsn)
	if err != nil {
		zap.L().Error("connect db failed", zap.Error(err))
	}
	db.SetMaxOpenConns(conf.MaxOpenConns) // 最大连接数
	db.SetMaxIdleConns(conf.MaxIdleConns) // 最大空闲连接数
	return
}

func Close() {
	_ = db.Close()
}
