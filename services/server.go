package services

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"webroot/controllers"
	"webroot/models"

	"github.com/gin-gonic/gin"
)

// ServerService 服务器核心服务
type ServerService struct {
	Config       *models.Config
	server       *http.Server
	tlsServer    *http.Server
	acme         *models.ACMEManager
	validDomains []string
}

type ip429LogRecord struct {
	count    int        // 429次数
	lastTime time.Time  // 最后一次429时间（用于自动重置）
	mu       sync.Mutex // 并发安全锁
}

var ip429LogMap = sync.Map{} // 存储IP的429日志计数

func (s *ServerService) GetConfig() any {
	return s.Config
}

func NewServerService() (*ServerService, error) {
	config := models.GetConfig()
	if config.Debug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	acmeManager := models.NewACMEManager(config)
	if err := acmeManager.Setup(); err != nil {
		return nil, err
	}
	validDomains := models.GetValidDomains(config.Server.HostHandlers)

	return &ServerService{
		Config:       config,
		acme:         acmeManager,
		validDomains: validDomains,
	}, nil
}

// CustomLogger 自定义日志中间件
func CustomLogger() gin.HandlerFunc {
	logger := log.New(os.Stdout, "", 0)

	// 配置：超过5次后屏蔽，5分钟后自动重置计数
	logThreshold := 5
	resetWindow := 5 * time.Minute

	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		end := time.Now()

		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		path := c.Request.URL.Path
		host := c.Request.Host
		latency := end.Sub(start)

		// 非429状态，正常输出日志
		if statusCode != http.StatusTooManyRequests {
			logger.Printf("%s | %3d | %13v | %s | %s | %s %s\n",
				end.Format("2006/01/02 - 15:04:05"),
				statusCode,
				latency,
				clientIP,
				host,
				method,
				path,
			)
			return
		}

		// 处理429状态：同一IP超过5次后屏蔽日志
		// 获取或初始化IP的429计数记录
		record, _ := ip429LogMap.LoadOrStore(clientIP, &ip429LogRecord{})
		logRecord := record.(*ip429LogRecord)

		logRecord.mu.Lock()
		defer logRecord.mu.Unlock()

		// 超过重置窗口（5分钟），重置计数
		if time.Since(logRecord.lastTime) > resetWindow {
			logRecord.count = 1
			logRecord.lastTime = time.Now()
		} else {
			logRecord.count++
			logRecord.lastTime = time.Now()
		}

		// 计数≤5时输出日志，超过则屏蔽
		if logRecord.count <= logThreshold {
			logger.Printf("%s | %3d | %13v | %s | %s | %s %s\n",
				end.Format("2006/01/02 - 15:04:05"),
				statusCode,
				latency,
				clientIP,
				host,
				method,
				path,
			)
		}
	}
}

// StartServers 启动HTTP/HTTPS服务器
func (s *ServerService) StartServers() {
	r := gin.New()
	r.Use(CustomLogger())
	r.Use(gin.Recovery())
	r.Use(func(c *gin.Context) {
		c.Set("config", s.Config)
		c.Next()
	})

	// 启用CC防御
	if s.Config.Server.AntiCC.Enabled {
		r.Use(AntiCCMiddleware(
			s.Config.Server.AntiCC.Window,
			s.Config.Server.AntiCC.MaxRequests,
			s.Config.Server.AntiCC.BaseBlockTime,
		))
		log.Printf("已启用CC防御：基础阈值 %d次/%v，基础封锁时间 %v",
			s.Config.Server.AntiCC.MaxRequests,
			s.Config.Server.AntiCC.Window,
			s.Config.Server.AntiCC.BaseBlockTime)
	}

	controllers.RegisterRoutes(r)

	// HTTP跳转引擎（用于80端口跳转HTTPS）
	redirectEngine := gin.New()
	redirectEngine.Use(CustomLogger())
	redirectEngine.Use(gin.Recovery())
	if s.Config.Server.AntiCC.Enabled {
		redirectEngine.Use(AntiCCMiddleware(
			s.Config.Server.AntiCC.Window,
			s.Config.Server.AntiCC.MaxRequests,
			s.Config.Server.AntiCC.BaseBlockTime,
		))
	}

	// ACME HTTP-01验证路由
	redirectEngine.GET("/.well-known/acme-challenge/*any", func(c *gin.Context) {
		s.acme.GetHTTPHandler().ServeHTTP(c.Writer, c.Request)
	})

	// HTTP跳转HTTPS
	redirectEngine.NoRoute(func(c *gin.Context) {
		host := c.Request.Host
		domain := strings.Split(host, ":")[0]
		isValid := false
		for _, d := range s.validDomains {
			if d == domain {
				isValid = true
				break
			}
		}
		if isValid {
			target := "https://" + host + c.Request.URL.RequestURI()
			c.Redirect(http.StatusFound, target)
		} else {
			c.AbortWithStatus(http.StatusNotFound)
		}
	})

	// 启动HTTP服务器
	if s.Config.Server.HTTPEnabled {
		addr := s.Config.Server.HTTPPort
		log.Printf("HTTP服务器已启动 监听地址：%s\n", addr)
		s.server = &http.Server{
			Addr:    addr,
			Handler: redirectEngine,
		}
		go func() {
			if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Printf("HTTP服务器监听失败: %v\n", err)
			}
		}()
	}

	// 启动HTTPS服务器
	if s.Config.Server.HTTPSEnabled {
		addr := s.Config.Server.HTTPSPort
		log.Printf("HTTPS服务器已启动 监听地址：%s\n", addr)
		s.tlsServer = &http.Server{
			Addr:      addr,
			Handler:   r,
			TLSConfig: s.acme.GetTLSConfig(),
		}
		go func() {
			if err := s.tlsServer.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				log.Printf("HTTPS服务器监听失败: %v\n", err)
			}
		}()
	}

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务器...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			log.Println("HTTP服务器强制关闭:", err)
		}
	}

	if s.tlsServer != nil {
		if err := s.tlsServer.Shutdown(ctx); err != nil {
			log.Println("HTTPS服务器强制关闭:", err)
		}
	}

	log.Println("服务器已退出")
}
