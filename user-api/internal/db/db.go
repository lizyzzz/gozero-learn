package db

import (
	"context"
	"time"
	"user-api/internal/config"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func NewMysql(mysqlConfig config.MysqlConfig) sqlx.SqlConn {
	// 获得一个 mysql 连接
	mysql := sqlx.NewMysql(mysqlConfig.DataSource)

	db, err := mysql.RawDB()

	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(mysqlConfig.ConnectTimeout))
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(100)
	db.SetMaxIdleConns(10)

	return mysql
}
