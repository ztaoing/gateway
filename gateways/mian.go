package main

import (
	"flag"
	"fmt"
	"github.com/go-kit/kit/log"
	"github.com/hashicorp/consul/api"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
)

func main() {
	//创建环境变量
	var (
		consulHost = flag.String("consul.host", "127.0.0.0.1", "consul server host")
		consulPort = flag.Int("consul.port", 8500, "consul server port")
	)
	flag.Parse()

	//创建日志组件
	var logger log.Logger

	logger = log.NewLogfmtLogger(os.Stderr)
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "caller", log.DefaultCaller)

	//创建consul api客户端
	consulConfig := api.DefaultConfig()
	consulConfig.Address = "http://" + *consulHost + ":" + strconv.Itoa(*consulPort)
	consulClient, err := api.NewClient(consulConfig)
	if err != nil {
		logger.Log("err", err)
		os.Exit(-1)
	}

	//创建反向代理
	proxy := NewReverseProxy(consulClient, logger)

	//结束信号
	errChan := make(chan error)
	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errChan <- fmt.Errorf("%s", <-c)
	}()

	//监听
	go func() {
		logger.Log("transport", "http", "addr", "9090")
		errChan <- http.ListenAndServe(":9090", proxy)
	}()

	//等待结束
	logger.Log("exit", <-errChan)
}

//创建反向代理
/**
 in consul客户端对象，日志记录工具
out 反向代理对象
*/

func NewReverseProxy(client *api.Client, logger log.Logger) *httputil.ReverseProxy {
	//Director必须是一个可以修改request为一个新的可以通过传输发送的方法
	//他会将返回response给原始的未经修改的client
	//Director不能访问正在返回中的request
	director := func(req *http.Request) {
		//查询原始请求路径 如：/arithmetic/calculate/10/5
		reqPath := req.URL.Path
		if reqPath == "" {
			return
		}
		//获取服务名称
		//对路径进行分解
		pathArray := strings.Split(reqPath, "/")
		serviceName := pathArray[1]

		//根据服务名称获取服务列表
		Catalog_result, _, err := client.Catalog().Service(serviceName, "", nil)
		if err != nil {
			logger.Log("ReveseProxy failed", "query service by serviceName error")
			return
		}
		//去掉服务名称后的请求路径
		destPath := strings.Join(pathArray[2:], "/")

		//随机选择服务列表中的一个
		target := Catalog_result[rand.Int()%len(Catalog_result)]
		logger.Log("service id", target.ServiceID)

		//设置代理服务地址信息
		req.URL.Scheme = "http"
		req.URL.Host = fmt.Sprintf("%s:%d", target.ServiceAddress, target.ServicePort)
		req.URL.Path = "/" + destPath
	}
	return &httputil.ReverseProxy{
		Director: director,
	}
}
