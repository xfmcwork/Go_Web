package services

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"webroot/controllers"
	"webroot/models"

	"github.com/gin-gonic/gin"
)

type ServerService struct {
	Config    *models.Config
	server    *http.Server
	tlsServer *http.Server
}

func (s *ServerService) GetConfig() any {
	return s.Config
}

func NewServerService() *ServerService {
	config := models.GetConfig()
	if config.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}
	return &ServerService{Config: config}
}

func CustomLogger() gin.HandlerFunc {
	logger := log.New(os.Stdout, "", 0)

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		end := time.Now()
		latency := end.Sub(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		statusCode := c.Writer.Status()

		logger.Printf("%s | %3d | %13v | %s | %s %s\n",
			end.Format("2006/01/02 - 15:04:05"),
			statusCode,
			latency,
			clientIP,
			method,
			path,
		)
	}
}

func (s *ServerService) StartServers() {
	r := gin.New()
	r.Use(CustomLogger())
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Set("config", s.Config)
		c.Next()
	})

	controllers.RegisterRoutes(r)

	if s.Config.Server.HTTPEncabled {
		addr := s.Config.Server.HTTPPort
		log.Printf("HTTP服务器已启动 监听地址：%s\n", addr)
		s.server = &http.Server{
			Addr:    addr,
			Handler: r,
		}
		go func() {
			if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("监听错误: %s\n", err)
			}
		}()
	}

	if s.Config.Server.HTTPSEnabled {
		addr := s.Config.Server.HTTPSPort
		certFile := s.Config.Server.SSL.CertFile
		keyFile := s.Config.Server.SSL.KeyFile
		log.Printf("HTTPS服务器已启动 监听地址：%s\n", addr)
		s.tlsServer = &http.Server{
			Addr:    addr,
			Handler: r,
		}
		go func() {
			if err := s.tlsServer.ListenAndServeTLS(certFile, keyFile); err != nil && err != http.ErrServerClosed {
				log.Printf("监听错误: %s\n", err)
			}
		}()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务器...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			log.Println("强制关闭服务器:", err)
		}
	}

	if s.tlsServer != nil {
		if err := s.tlsServer.Shutdown(ctx); err != nil {
			log.Println("强制关闭服务器:", err)
		}
	}

	log.Println("服务器已退出")
}
