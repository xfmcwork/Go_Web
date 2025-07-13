package models

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

var (
	defaultConfigPath = "./config/config.yaml"
	globalConfig      *Config
)

type ServerConfig struct {
	HTTPEncabled bool       `yaml:"http_enabled"`
	HTTPSEnabled bool       `yaml:"https_enabled"`
	HTTPPort     string     `yaml:"http_port"`
	HTTPSPort    string     `yaml:"https_port"`
	HTTPToHTTPS  bool       `yaml:"http_to_https"`
	SSL          SSLConfig  `yaml:"ssl"`
	DB           DBConfig   `yaml:"db"`
	SMTP         SMTPConfig `yaml:"smtp"`
}

type SSLConfig struct {
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type SMTPConfig struct {
	Server   string `yaml:"server"`
	Port     int    `yaml:"port"`
	SSL      bool   `yaml:"ssl"`
	Email    string `yaml:"email"`
	Password string `yaml:"password"`
}

type Config struct {
	Debug  bool         `yaml:"debug"`
	Server ServerConfig `yaml:"server"`
}

type DBConfig struct {
	File string `yaml:"file"`
}

type LoggingConfig struct {
	Level          string `yaml:"level"`
	ConsoleEnabled bool   `yaml:"console_enabled"`
	FileEnabled    bool   `yaml:"file_enabled"`
}

func SetDefaultConfigPath(path string) {
	defaultConfigPath = path
}

func ParseConfig(configPath string) (*Config, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("配置文件不存在，创建默认配置: %s", configPath)
		dir := filepath.Dir(configPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("创建配置目录失败: %v", err)
		}
		defaultConfig := &Config{
			Debug: false,
			Server: ServerConfig{
				HTTPEncabled: true,
				HTTPSEnabled: false,
				HTTPPort:     ":80",
				HTTPSPort:    ":443",
				HTTPToHTTPS:  false,
				SSL: SSLConfig{
					CertFile: "./config/cert.pem",
					KeyFile:  "/config/key.pem",
				},
				DB: DBConfig{
					File: "./config/sqlite.db",
				},
				SMTP: SMTPConfig{
					Server:   "smtp.example.com",
					Port:     587,
					SSL:      true,
					Email:    "your_email@example.com",
					Password: "your_password",
				},
			},
		}
		yamlData, err := yaml.Marshal(defaultConfig)
		if err != nil {
			return nil, fmt.Errorf("生成默认配置失败: %v", err)
		}
		if err := os.WriteFile(configPath, yamlData, 0644); err != nil {
			return nil, fmt.Errorf("写入配置文件失败: %v", err)
		}

		log.Println("已创建默认配置文件，请根据需要修改")
		return defaultConfig, nil
	}

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


func GetConfig() *Config {
	return globalConfig
}

func init() {
	config, err := ParseConfig(defaultConfigPath)
	if err != nil {
		log.Fatalf("初始化配置失败: %v\n", err)
	}
	globalConfig = config
}
