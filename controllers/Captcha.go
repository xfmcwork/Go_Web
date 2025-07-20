package controllers

import (
	"errors"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/dchest/captcha"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// 记录IP的验证码生成记录
type ipCaptchaRecord struct {
	count    int        // 1秒内的请求次数
	lastTime time.Time  // 最后一次请求时间
	mu       sync.Mutex // 并发安全锁
}

var ipCaptchaMap = sync.Map{} // 全局IP验证码计数器
func GetCaptcha(c *gin.Context) {
	clientIP := c.ClientIP()

	// 1. 获取或初始化IP记录
	record, _ := ipCaptchaMap.LoadOrStore(clientIP, &ipCaptchaRecord{})
	ipRecord := record.(*ipCaptchaRecord)

	ipRecord.mu.Lock()
	defer ipRecord.mu.Unlock()

	// 2. 检查时间窗口（1秒内）
	now := time.Now()
	if now.Sub(ipRecord.lastTime) < time.Second {
		// 1秒内，累加计数
		ipRecord.count++
		// 超过2次，返回429
		if ipRecord.count > 2 {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "验证码请求过于频繁，请稍后再试",
			})
			return
		}
	} else {
		// 超过1秒，重置计数
		ipRecord.count = 1
	}
	ipRecord.lastTime = now // 更新最后请求时间

	// 3. 原有验证码生成逻辑
	session := sessions.Default(c)
	if oldID, ok := session.Get("captcha_id").(string); ok && oldID != "" {
		captcha.Reload(oldID)
	}
	captchaID := captcha.NewLen(4)
	session.Set("captcha_id", captchaID)
	log.Printf("生成新验证码: %s (IP: %s, 计数: %d/2)", captchaID, clientIP, ipRecord.count)
	session.Save()

	c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
	c.Header("Pragma", "no-cache")
	c.Header("Expires", "0")
	c.Status(200)
	captcha.WriteImage(c.Writer, captchaID, 240, 80)
}

func VerifyForm(c *gin.Context) (bool, error) {
	session := sessions.Default(c)
	captchaID := session.Get("captcha_id")
	if captchaID == nil {
		return false, errors.New("验证码已过期或无效")
	}
	userCaptcha := c.PostForm("captcha")
	if userCaptcha == "" {
		log.Printf("验证码为空 %v", captchaID)
		return false, errors.New("请提供验证码")
	}
	if !captcha.VerifyString(captchaID.(string), userCaptcha) {
		log.Printf("验证码错误 %v 用户输入: %s", captchaID, userCaptcha)
		return false, errors.New("验证码错误")
	}
	log.Printf("验证码验证成功 %v 已删除", captchaID)
	return true, nil
}
