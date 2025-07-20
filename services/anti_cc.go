package services

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"log"
)

// ipRequestRecord 存储IP请求记录（用于CC防御）
type ipRequestRecord struct {
	count        int           // 当前窗口请求数
	level        int           // 当前限制级别
	blockedUntil time.Time     // 封锁结束时间
	resetAt      time.Time     // 计数重置时间（窗口结束）
	mu           sync.Mutex    // 并发安全锁
}

var ipRequestMap = sync.Map{} // 全局IP请求记录映射

// AntiCCMiddleware 多级CC防御中间件
// 参数：窗口时间、基础阈值、基础封锁时间
func AntiCCMiddleware(window time.Duration, maxRequests int, baseBlockTime time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()

		// 信任IP直接放行
		if isTrustedIP(clientIP) {
			c.Next()
			return
		}

		// 获取或初始化IP记录
		record, _ := ipRequestMap.LoadOrStore(clientIP, &ipRequestRecord{
			resetAt: time.Now().Add(window),
		})
		ipRecord := record.(*ipRequestRecord)

		ipRecord.mu.Lock()
		defer ipRecord.mu.Unlock()

		// 若处于封锁期，直接拦截
		if time.Now().Before(ipRecord.blockedUntil) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "请求过于频繁，请稍后再试",
			})
			return
		}

		// 窗口过期则重置计数
		if time.Now().After(ipRecord.resetAt) {
			ipRecord.count = 1
			ipRecord.resetAt = time.Now().Add(window)
		} else {
			ipRecord.count++
		}

		// 计算当前级别阈值（基础阈值 * 2^level）
		currentThreshold := maxRequests * (1 << ipRecord.level)
		if ipRecord.count > currentThreshold {
			// 升级级别（最高6级）
			newLevel := ipRecord.level + 1
			if newLevel > 6 {
				newLevel = 6
			}

			// 计算封锁时间（基础时间 * 2^(newLevel-1)）
			blockDuration := baseBlockTime * time.Duration(1<<(newLevel-1))
			blockedUntil := time.Now().Add(blockDuration)

			// 输出日志
			log.Printf(
				"CC防御：IP %s 触发级别%d，阈值%d，封锁%s至%s",
				clientIP,
				newLevel,
				maxRequests*(1<<newLevel),
				blockDuration,
				blockedUntil.Format("2006-01-02 15:04:05"),
			)

			// 更新记录状态
			ipRecord.level = newLevel
			ipRecord.blockedUntil = blockedUntil
			ipRecord.count = 0

			// 拦截请求
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "请求过于频繁，请稍后再试",
			})
			return
		}

		// 正常放行
		c.Next()
	}
}

// isTrustedIP 检查IP是否在信任列表
func isTrustedIP(ip string) bool {
	trustedIPs := map[string]bool{
		"127.0.0.1": true,
		"::1":       true,
	}
	return trustedIPs[ip]
}