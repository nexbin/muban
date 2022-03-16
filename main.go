package main

import (
	"BlueBell2/dao/mysql"
	"BlueBell2/dao/redis"
	"BlueBell2/logger"
	"BlueBell2/routes"
	"BlueBell2/settings"
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

func main() {
	// 1.加载配置
	configPath := flag.String("c", "", "config path")
	flag.Parse()
	if err := settings.Init(*configPath); err != nil {
		fmt.Printf("init settings failed, err:%v\n", err)
		return
	}
	// 2.初始化日志
	if err := logger.Init(settings.Conf.LogConfig); err != nil {
		fmt.Printf("init logger failed, err:%v\n", err)
		return
	}
	// 同步缓冲区日志
	defer zap.L().Sync()
	zap.L().Debug("logger init success..")
	// 3.初始化mysql
	if err := mysql.Init(settings.Conf.MysqlConfig); err != nil {
		fmt.Printf("init mysql failed, err:%v\n", err)
	}
	defer mysql.Close()
	// 4.初始化redis
	if err := redis.Init(settings.Conf.RedisConfig); err != nil {
		fmt.Printf("init redis failed, err:%v\n", err)
	}
	defer redis.Close()
	// 初始化路由
	r := routes.SetupRouter()

	server := &http.Server{
		Addr:      fmt.Sprintf(":%d", settings.Conf.Port),
		Handler:   r,
		TLSConfig: nil,
	}
	// 启动服务
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen %s\n", err)
		}
	}()
	// 优雅关机
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shut down server..")
	// 创建一个5s超时的context
	withTimeOutCtx, cancelFunc := context.WithTimeout(context.Background(), time.Second*5)
	defer cancelFunc()
	if err := server.Shutdown(withTimeOutCtx); err != nil {
		zap.L().Fatal("server shutdown: ", zap.Error(err))
	}
	zap.L().Info("server exiting..")
}
