package main

import (
	"center/server"
	"center/util"
	"os"

	"gopkg.in/ini.v1"
)

func main() {
	// 读取配置文件
	cfg, err := ini.Load("src/static/config.ini")

	if err != nil {
		util.Errorf("读取文件失败: %v", err)
		os.Exit(1)
	}
	// 获得一个SFUServerConfig 结构体
	config := server.DefaultConfig()
	// 给结构体赋值
	config.Host = cfg.Section("general").Key("bind").String()
	config.Port, _ = cfg.Section("general").Key("port").Int()
	config.CertFile = cfg.Section("general").Key("cert").String()
	config.KeyFile = cfg.Section("general").Key("key").String()
	config.HTMLRoot = cfg.Section("general").Key("html_root").String()
	config.NodeHost = cfg.Section("general").Key("nodeHost").String()
	// 获取一个NewSFUServer结构体 结构体中的server即socket服务
	wsServer := server.NewSFUServer()
	wsServer.Bind(config)
}
