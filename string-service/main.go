package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/go-kit/kit/log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	var (
		consulHost  = flag.String("consul.host", "127.0.0.1", "consul ip address")
		consulPort  = flag.String("consul.port", "8500", "consul port")
		serviceHost = flag.String("service.host", "localhost", "service host")
		servicePort = flag.String("service.port", "8080", "service port")
	)
	flag.Parse()

	ctx := context.Background()
	errChan := make(chan error)

	var logger log.Logger
	logger = log.NewLogfmtLogger(os.Stderr)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "caller", log.DefaultCaller)

	var svc Service
	svc = StringService{}

	//增加日志中间件
	svc = LoggingMiddleware(logger)(svc)

	//string endpoint
	endpoint := MakeStringEndpoint(svc)

	//创建健康检查的endpoint
	healthEndpoint := MakeHealthCheckEndpoint(svc)

	//封装
	endpts := StringEndpoints{
		StringEndpoint:      endpoint,
		HealthCheckEndpoint: healthEndpoint,
	}

	r := MakeHttpHandler(ctx, endpts, logger)

	//创建注册对象
	//TODO 用consul pkg替换
	register := Register(*consulHost, *consulPort, *serviceHost, *servicePort, logger)

	go func() {
		fmt.Println("http server start at port:" + *servicePort)
		//启动注册
		register.Register()
		handler := r
		errChan <- http.ListenAndServe(":"+*servicePort, handler)
	}()

	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errChan <- fmt.Errorf("%s", <-c)
	}()

	error := <-errChan

	//服务退出取消注册
	register.Deregister()
	fmt.Println(error)

}
