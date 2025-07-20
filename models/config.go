package models

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v2"
)

var (
	defaultConfigPath = "./config/config.yaml"
	globalConfig      *Config
)

// ServerConfig 服务器核心配置
type ServerConfig struct {
	HTTPEnabled  bool        `yaml:"http_enabled"`   // 是否启用HTTP服务器
	HTTPSEnabled bool        `yaml:"https_enabled"`  // 是否启用HTTPS服务器
	HTTP3Enabled bool        `yaml:"http3_enabled"`  // 是否启用HTTP/3服务器
	HTTPPort     string      `yaml:"http_port"`      // HTTP监听端口（如:80）
	HTTPSPort    string      `yaml:"https_port"`     // HTTPS监听端口（如:443）
	HTTPToHTTPS  bool        `yaml:"http_to_https"`  // 是否自动从HTTP跳转至HTTPS
	HostHandlers HostHandler `yaml:"hostHandlers"`  // 域名与处理器的映射关系
	DB           DBConfig    `yaml:"db"`             // 数据库配置
	SMTP         SMTPConfig  `yaml:"smtp"`           // 邮件服务配置
	SSL          SSLConfig   `yaml:"ssl"`            // ACME证书配置
	AntiCC       AntiCCConfig `yaml:"anti_cc"`        // CC攻击防御配置
}

// AntiCCConfig CC攻击防御配置
type AntiCCConfig struct {
	Enabled       bool          `yaml:"enabled"`        // 是否启用CC防御
	Window        time.Duration `yaml:"window"`         // 计数时间窗口（如1m）
	MaxRequests   int           `yaml:"max_requests"`   // 基础阈值（窗口内允许的最大请求数）
	BaseBlockTime time.Duration `yaml:"base_block_time"`// 基础封锁时间（如5m）
}

// HostHandler 域名与处理器的映射（key:域名, value:处理器名称）
type HostHandler map[string]string

// SMTPConfig 邮件发送配置
type SMTPConfig struct {
	Server   string `yaml:"server"`   // SMTP服务器地址（如smtp.qq.com）
	Port     int    `yaml:"port"`     // SMTP服务器端口（如465）
	SSL      bool   `yaml:"ssl"`      // 是否启用SSL加密
	Email    string `yaml:"email"`    // 发送邮件的账号
	Password string `yaml:"password"` // 发送邮件的密码/授权码
}

// Config 全局配置根结构
type Config struct {
	Debug  bool         `yaml:"debug"`  // 是否启用调试模式
	Server ServerConfig `yaml:"server"` // 服务器配置
}

// DBConfig 数据库配置（当前为SQLite配置）
type DBConfig struct {
	File string `yaml:"file"` // 数据库文件路径
}

// SSLConfig ACME证书申请配置
type SSLConfig struct {
	DirectoryURL string `yaml:"directory_url"` // ACME服务器地址（如Let's Encrypt）
	Mail         string `yaml:"mail"`          // 用于证书申请的邮箱
	KeyID        string `yaml:"keyid"`         // EAB认证的Key ID
	HMACKey      string `yaml:"hmackey"`       // EAB认证的HMAC密钥
}

// SetDefaultConfigPath 设置默认配置文件路径
func SetDefaultConfigPath(path string) {
	defaultConfigPath = path
}

// ParseConfig 解析配置文件，若不存在则创建默认配置
func ParseConfig(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("配置文件不存在，创建默认配置: %s", configPath)
		dir := filepath.Dir(configPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("创建配置目录失败: %v", err)
		}
		// 生成默认配置
		defaultConfig := &Config{
			Debug: false,
			Server: ServerConfig{
				HTTPEnabled:  true,
				HTTPSEnabled: false,
				HTTP3Enabled: false,
				HTTPPort:     ":80",
				HTTPSPort:    ":443",
				HTTPToHTTPS:  false,
				HostHandlers: HostHandler{
					"127.0.0.1":    "default",
					"localhost":    "default",
					"www.example.com": "default",
				},
				DB: DBConfig{
					File: "./config/data.db",
				},
				SMTP: SMTPConfig{
					Server:   "smtp.qq.com",
					Port:     465,
					SSL:      true,
					Email:    "",
					Password: "",
				},
				SSL: SSLConfig{
					DirectoryURL: "https://acme-staging-v02.api.letsencrypt.org/directory", // 测试环境
					// DirectoryURL: "https://acme-v02.api.letsencrypt.org/directory",      // 生产环境
					Mail:    "",
					KeyID:   "",
					HMACKey: "",
				},
				AntiCC: AntiCCConfig{
					Enabled:       true,
					Window:        1 * time.Minute,
					MaxRequests:   100,
					BaseBlockTime: 5 * time.Minute,
				},
			},
		}
		// 写入默认配置文件
		yamlData, err := yaml.Marshal(defaultConfig)
		if err != nil {
			return nil, fmt.Errorf("生成默认配置失败: %v", err)
		}
		if err := os.WriteFile(configPath, yamlData, 0644); err != nil {
			return nil, fmt.Errorf("写入默认配置文件失败: %v", err)
		}
		log.Println("默认配置文件创建成功，请根据需求修改")
		return defaultConfig, nil
	}

	// 读取并解析现有配置文件
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("打开配置文件失败: %v", err)
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	log.Printf("配置文件加载成功: %s", configPath)
	return &config, nil
}

// GetConfig 获取全局配置实例
func GetConfig() *Config {
	return globalConfig
}

// UpdateConfig 更新全局配置并写入文件
func UpdateConfig(updateFunc func(*Config)) error {
	// 深拷贝配置避免并发问题
	configCopy, err := copyConfig(globalConfig)
	if err != nil {
		return fmt.Errorf("拷贝配置失败: %v", err)
	}
	// 应用更新
	updateFunc(configCopy)
	// 序列化并写入文件
	yamlData, err := yaml.Marshal(configCopy)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}
	if err := os.WriteFile(defaultConfigPath, yamlData, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}
	// 更新全局配置引用
	globalConfig = configCopy
	log.Printf("配置已更新: %s", defaultConfigPath)
	return nil
}

// UpdateSSLConfig 仅更新SSL配置并写入文件
func UpdateSSLConfig(updateFunc func(*SSLConfig)) error {
	return UpdateConfig(func(c *Config) {
		updateFunc(&c.Server.SSL)
	})
}

// copyConfig 深拷贝配置（通过序列化-反序列化实现）
func copyConfig(src *Config) (*Config, error) {
	data, err := yaml.Marshal(src)
	if err != nil {
		return nil, err
	}
	var dest Config
	if err := yaml.Unmarshal(data, &dest); err != nil {
		return nil, err
	}
	return &dest, nil
}

// 初始化全局配置
func init() {
	config, err := ParseConfig(defaultConfigPath)
	if err != nil {
		log.Fatalf("初始化配置失败: %v", err)
	}
	globalConfig = config
}