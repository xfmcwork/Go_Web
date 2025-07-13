package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func DefaultHandlers(c *gin.Context) {
	path := c.Request.URL.Path
	method := c.Request.Method
	switch {
	case path == "/":
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "域名没有绑定"})
	case path == "/api":
		switch method {
		case http.MethodPost:
		case http.MethodGet:
		}
	default:
		c.JSON(http.StatusNotFound, gin.H{"msg": "目录不存在"})
	}
}
