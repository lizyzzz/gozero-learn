package main

import (
	"context"
	"fmt"
	"grpc-client/greet"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/zrpc"
)

func main() {
	// 创建一个日志文件
	// logFile, err := os.OpenFile("grpc.log", os.O_CREATE|os.O_WRONLY, 0666)
	// if err != nil {
	// 	log.Fatalf("无法打开日志文件: %v", err)
	// }

	// // 创建一个新的日志记录器
	// // logger := log.New(logFile, "", log.LstdFlags)
	// // 设置 gRPC 使用自定义的日志记录器
	// grpclog.SetLoggerV2(grpclog.NewLoggerV2(logFile, logFile, logFile))

	var clientConf zrpc.RpcClientConf
	conf.MustLoad("etc/client.yaml", &clientConf)
	conn := zrpc.MustNewClient(clientConf)

	client := greet.NewGreetClient(conn.Conn())

	resp, err := client.Ping(context.Background(), &greet.Request{Ping: "hello"})
	if err != nil {
		panic(err)
	}

	fmt.Println(resp)
}
