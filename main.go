package main

import (
	"log"
	"webroot/models"
	"webroot/services"
)

func main() {

	config := models.GetConfig()

	if err := models.Init(config.Server.DB.Dir); err != nil {
		log.Printf("数据库初始化失败: %v", err)
		return
	}

	if err := models.Init(config.Server.DB.Dir); err != nil {
		log.Printf("数据库初始化失败: %v", err)
	}
	serverService := services.NewServerService()

	log.Println("准备服务...")
	serverService.StartServers()
}
