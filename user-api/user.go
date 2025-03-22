package main

import (
	"flag"
	"fmt"
	"net/http"

	"user-api/internal/biz"
	"user-api/internal/config"
	"user-api/internal/handler"
	"user-api/internal/middlewares"
	"user-api/internal/svc"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/user-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf)
	// 添加中间件
	server.Use(middlewares.PrintLog)
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// 统一的错误处理方式, httpx.ErrorCtx(r.Context(), w, err) 处理错误时会调用到设置的错误处理函数
	httpx.SetErrorHandler(func(err error) (int, any) {
		switch e := err.(type) {
		case *biz.Error:
			return http.StatusOK, biz.Fail(e)
		default:
			return http.StatusInternalServerError, nil
		}
	})

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
