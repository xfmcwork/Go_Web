package main

import (
	"embed"
	"log"
	"webroot/controllers"
	"webroot/models"
	"webroot/services"
)

//go:embed public/*
var staticFS embed.FS

func main() {

	config := models.GetConfig()
	if err := models.Init(config.Server.DB.File); err != nil {
		log.Printf("数据库初始化失败: %v", err)
		return
	}
	controllers.InitStaticFS(staticFS)
	serverService, err := services.NewServerService()
	if err != nil {
		log.Fatalf("初始化服务器服务失败: %v", err)
	}

	log.Println("准备服务...")
	serverService.StartServers()
}
