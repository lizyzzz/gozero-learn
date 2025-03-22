package svc

import (
	"user-api/internal/config"
	"user-api/internal/db"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ServiceContext struct {
	Config    config.Config
	Conn      sqlx.SqlConn
	RedisConn *redis.Redis
}

func NewServiceContext(c config.Config) *ServiceContext {
	return &ServiceContext{
		Config:    c,
		Conn:      db.NewMysql(c.MysqlConfig),
		RedisConn: db.NewRedis(c.RedisConfig),
	}
}
