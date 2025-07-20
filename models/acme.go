package models

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"log"
	"math/big"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"golang.org/x/crypto/acme"
	"golang.org/x/crypto/acme/autocert"
)

type ACMEManager struct {
	Config  *Config
	Manager *autocert.Manager
}

func NewACMEManager(config *Config) *ACMEManager {
	return &ACMEManager{Config: config}
}

func (a *ACMEManager) Setup() error {
	if !a.Config.Server.HTTPSEnabled {
		log.Print("未启用HTTPS ACME服务未启用")
		return nil
	}
	domains := GetValidDomains(a.Config.Server.HostHandlers)
	if len(domains) == 0 {
		return fmt.Errorf("未找到有效的域名配置")
	}

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &loggingTransport{
			base:  http.DefaultTransport,
			debug: a.Config.Debug,
		},
	}

	caURL := a.Config.Server.SSL.DirectoryURL
	if caURL == "" {
		caURL = "https://acme-staging-v02.api.letsencrypt.org/directory"
		log.Printf("未配置CA服务器 默认lets测试服务器: %s", caURL)
	} else {
		log.Printf("已配置自定义CA服务器: %s", caURL)
	}
	Mail := a.Config.Server.SSL.Mail
	if Mail == "" {
		Mail, err := GenerateRandomGmail()
		if err != nil {
			fmt.Printf("生成随机Gmail失败: %v\n", err)
		}
		if Mail == "" {
			Mail, err := GenerateRandomGmail()
			if err != nil {
				fmt.Printf("生成随机Gmail失败: %v\n", err)
			}
			UpdateSSLConfig(func(ssl *SSLConfig) { ssl.Mail = Mail })
			log.Printf("未配置CA邮箱 使用随机邮箱申请证书(已写入配置文件): %s", Mail)
		} else {

			log.Printf("已配置CA邮箱 使用邮箱申请证书: %s", Mail)
		}
		log.Printf("未配置CA邮箱 使用随机邮箱申请证书: %s", Mail)
	} else {
		log.Printf("已配置CA邮箱 使用邮箱申请证书: %s", Mail)
	}
	eabKeyID := a.Config.Server.SSL.KeyID
	eabHMACKey := a.Config.Server.SSL.HMACKey
	hmacKeyBytes, err := base64.RawURLEncoding.DecodeString(eabHMACKey)
	if err != nil {
		return fmt.Errorf("解码 HMAC 密钥失败: %v", err)
	}

	if eabKeyID == "" && eabHMACKey == "" {
		log.Print("未配置EAB的KID或Key 默认邮箱申请")
		a.Manager = &autocert.Manager{
			Cache:       autocert.DirCache("./config/ssl"),
			Prompt:      autocert.AcceptTOS,
			HostPolicy:  autocert.HostWhitelist(domains...),
			Email:       Mail,
			RenewBefore: 30 * 24 * time.Hour,
			Client: &acme.Client{
				DirectoryURL: caURL,
				HTTPClient:   client,
			},
		}
	} else {
		log.Print("已配置EAB的KID与Key 使用外置账号申请")
		a.Manager = &autocert.Manager{
			Cache:       autocert.DirCache("./config/ssl"),
			Prompt:      autocert.AcceptTOS,
			HostPolicy:  autocert.HostWhitelist(domains...),
			Email:       a.Config.Server.SSL.Mail,
			RenewBefore: 30 * 24 * time.Hour,
			Client: &acme.Client{
				DirectoryURL: caURL,
				HTTPClient:   client,
			},
			ExternalAccountBinding: &acme.ExternalAccountBinding{
				KID: eabKeyID,
				Key: hmacKeyBytes,
			},
		}
	}

	a.Manager.TLSConfig().GetCertificate = func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
		log.Printf("收到TLS握手请求: ServerName=%s", info.ServerName)

		cert, err := a.Manager.GetCertificate(info)
		if err != nil {
			log.Printf("获取证书失败: ServerName=%s, 错误=%v", info.ServerName, err)
			return nil, err
		}

		if len(cert.Certificate) > 0 {
			x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
			if err == nil {
				log.Printf("成功获取证书: ServerName=%s, 通用名=%s, SAN=%v, 有效期至=%s",
					info.ServerName,
					x509Cert.Subject.CommonName,
					x509Cert.DNSNames,
					x509Cert.NotAfter.Format(time.RFC3339))
			} else {
				log.Printf("解析证书失败: ServerName=%s, 错误=%v", info.ServerName, err)
			}
		} else {
			log.Printf("证书内容为空: ServerName=%s", info.ServerName)
		}

		return cert, nil
	}

	go a.checkCertificates(domains)

	return nil
}

func (a *ACMEManager) GetHTTPHandler() http.Handler {
	return a.Manager.HTTPHandler(nil)
}

func (a *ACMEManager) GetTLSConfig() *tls.Config {
	return a.Manager.TLSConfig()
}

func (a *ACMEManager) checkCertificates(domains []string) {
	time.Sleep(5 * time.Second)
	log.Println("检查证书缓存状态...")

	for _, domain := range domains {
		cert, err := a.Manager.GetCertificate(&tls.ClientHelloInfo{ServerName: domain})
		if err != nil {
			log.Printf("检查证书 [%s] 失败: %v", domain, err)
		} else if cert != nil {
			log.Printf("证书 [%s] 已缓存，包含 %d 个证书", domain, len(cert.Certificate))
		} else {
			log.Printf("证书 [%s] 未找到，将在首次请求时申请", domain)
		}
	}
}

type loggingTransport struct {
	base  http.RoundTripper
	debug bool
}

func (t *loggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.debug {
		reqDump, err := httputil.DumpRequestOut(req, true)
		if err == nil {
			log.Printf("[ACME] 请求:\n%s\n", string(reqDump))
		} else {
			log.Printf("[ACME] 记录请求失败: %v", err)
		}
	}

	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return nil, err
	}

	if t.debug {
		respDump, err := httputil.DumpResponse(resp, true)
		if err == nil {
			log.Printf("[ACME] 响应:\n%s\n", string(respDump))
		} else {
			log.Printf("[ACME] 记录响应失败: %v", err)
		}
	}

	return resp, nil
}

func GetValidDomains(hostHandlers HostHandler) []string {
	var domains []string
	for host := range hostHandlers {
		if !strings.Contains(host, ".") {
			continue
		}
		if strings.HasPrefix(host, "127.") || strings.HasPrefix(host, "192.168.") {
			continue
		}
		domains = append(domains, host)
	}
	return domains
}

func GenerateRandomGmail() (string, error) {
	letterBytes := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	n := 6
	result := make([]byte, n)
	for i := range result {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			return "", err
		}
		result[i] = letterBytes[num.Int64()]
	}
	return string(result) + "@gmail.com", nil
}
