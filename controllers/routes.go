package controllers

import (
	"fmt"
	"log"
	"net/http"

	"webroot/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

var handlerRegistry = map[string]func(*gin.Context){
	"Blog": BlogHandlers,
}

var hostHandlers map[string]func(*gin.Context)

func init() {
	initHostHandlers()
}

func initHostHandlers() {
	config := models.GetConfig()
	hostHandlers = make(map[string]func(*gin.Context))
	for host, handlerName := range config.Server.HostHandlers {
		handler, exists := handlerRegistry[handlerName]
		if !exists {
			log.Printf("警告: 未找到处理器 %s，主机 %s 将使用默认处理器", handlerName, host)
			continue
		}
		hostHandlers[host] = handler
		log.Printf("注册主机处理器: %s -> %s", host, handlerName)
	}
}

func RegisterRoutes(r *gin.Engine) {
	store := cookie.NewStore([]byte("WoXiHanXiMi250"))
	r.Use(sessions.Sessions("session", store))

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Server", "My-Server")

		configValue, exists := c.Get("config")
		if !exists {
			log.Println("配置未找到")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"msg": "配置未找到"})
			return
		}

		config, ok := configValue.(*models.Config)
		if !ok {
			log.Println("配置类型错误")
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"msg": "配置类型错误"})
			return
		}

		if config.Server.HTTPToHTTPS && c.Request.TLS == nil {
			scheme := "https"
			host := c.Request.Host
			uri := c.Request.URL.String()
			newURL := fmt.Sprintf("%s://%s%s", scheme, host, uri)
			c.Redirect(http.StatusFound, newURL)
			c.Abort()
		}
	})

	r.Use(func(c *gin.Context) {
		host := c.Request.Host
		if i := len(host); i > 0 {
			if host[i-1] == ']' {
			} else if j := len(host) - 1; j >= 0 && host[j] == ':' {
				host = host[:j]
			}
		}

		if handler, exists := hostHandlers[host]; exists {
			handler(c)
		} else {
			DefaultHandlers(c)
		}
		c.Abort()
	})
}
