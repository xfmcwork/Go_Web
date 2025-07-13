package models

import (
	"log"
	"os"

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
	Dir string `yaml:"dir"`
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
	file, err := os.Open(configPath)
	if err != nil {
		log.Printf("打开配置文件失败: %v", err)
		return nil, err
	}
	defer file.Close()

	var config Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		log.Printf("解析配置文件失败: %v", err)
		return nil, err
	}

	log.Printf("配置文件加载成功: %s\n", configPath)
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
