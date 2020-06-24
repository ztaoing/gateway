package main

import (
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/sd"
	"github.com/go-kit/kit/sd/consul"
	"github.com/google/uuid"
	"github.com/hashicorp/consul/api"
	"strconv"
)

func Register(consulHost, consulPort, svcHost, svcPort string, logger log.Logger) (register sd.Registrar) {
	//创建consul客户端连接
	var client consul.Client
	consulCfg := api.DefaultConfig()
	consulCfg.Address = consulHost + ":" + consulPort
	consulClient, err := api.NewClient(consulCfg)
	if err != nil {
		logger.Log("create consul client err:", err)
	}

	client = consul.NewClient(consulClient)

	//设置consul健康检查的参数
	check := api.AgentServiceCheck{
		HTTP:     "http://" + svcHost + ":" + svcPort + "/health",
		Interval: "10s",
		Timeout:  "1s",
		Notes:    "Consul check service health status",
	}
	port, _ := strconv.Atoi(svcPort)

	//设置微服务consul的注册信息
	reg := api.AgentServiceRegistration{
		ID:      "string-service" + uuid.New().String(),
		Name:    "string-service",
		Address: svcHost,
		Port:    port,
		Tags:    []string{"string-service", "aoho"},
		Check:   &check,
	}
	register = consul.NewRegistrar(client, &reg, logger)
	return register

}
