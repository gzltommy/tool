package config

import (
	"sync"
	"time"

	"github.com/jinzhu/configor"
)

var (
	Cfg Configuration
	mu  sync.RWMutex
)

type (
	Configuration struct {
		App    AppConfig    `json:"app"`
		Server ServerConfig `json:"server"`
		Mysql  MysqlConfig  `json:"mysql"`
		Logger LoggerConfig `json:"logger"`
		//S3     S3Config     `json:"s3"`
		//Redis           RedisConfig              `json:"redis"`
		//RabbitMqConfig  RabbitMqConfig           `json:"rabbitMq"`
		//Elastic         ElasticConfig            `json:"elastic"`
		//Mongo           MongoConfig              `json:"mongo"`
		//Google          GoogleConfig             `json:"google"`
	}

	ServerConfig struct {
		RunMode         string        `json:"run_mode"`
		ListenAddr      string        `json:"listen_addr"`
		LimitConnection int           `json:"limit_connection"`
		ReadTimeout     time.Duration `json:"read_timeout"`
		WriteTimeout    time.Duration `json:"write_timeout"`
		IdleTimeout     time.Duration `json:"idle_timeout"`
		MaxHeaderBytes  int           `json:"max_header_bytes"`
	}

	LoggerConfig struct {
		Level          string        `json:"level"`
		Formatter      string        `json:"formatter"`
		DisableConsole bool          `json:"disable_console"`
		Write          bool          `json:"write"`
		Path           string        `json:"path"`
		FileName       string        `json:"file_name"`
		MaxAge         time.Duration `json:"max_age"`
		RotationTime   time.Duration `json:"rotation_time"`
		Debug          bool          `json:"debug"`
		ReportCaller   bool          `json:"report_caller"`
	}

	MysqlConfig struct {
		Driver   string `json:"driver"`
		Host     string `json:"host"`
		Port     string `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		DbName   string `json:"db_name"`
	}

	S3Config struct {
		AccessKey string `json:"access_key"`
		SecretKey string `json:"secret_key"`
		Bucket    string `json:"bucket"`
		BaseUrl   string `json:"base_url"`
	}

	AppConfig struct {
		Secret string `json:"secret" default:"secret."`
		Env    string `json:"env" default:""`
	}

	RedisConfig struct {
		Host      string `json:"host" env:"REDIS_HOST"`
		Port      int    `json:"port" env:"REDIS_PORT"`
		Auth      string `json:"auth" env:"REDIS_AUTH"`
		MaxIdle   int    `json:"max_idle" env:"REDIS_MAX_IDLE"`
		MaxActive int    `json:"max_active" env:"REDIS_MAX_ACTIVE"`
		Db        int    `json:"db" env:"REDIS_DB"`
	}

	ElasticConfig struct {
		Url      string `json:"url"`
		Username string `json:"username"`
		Password string `json:"password"`
	}

	MongoConfig struct {
		Host          string `json:"host" env:"MONGO_HOST"`
		Port          int    `json:"port" env:"MONGO_PORT"`
		Username      string `json:"username"`
		Password      string `json:"password"`
		MinPools      int    `json:"min_pools" `
		MaxIdleTimeMS int    `json:"max_idle_time_ms"`
		UseSSl        bool   `json:"use_ssl" default:"false"`
		TlsFile       string `json:"tls_file"`
		Database      string `json:"database"`
	}

	RabbitMqConfig struct {
		Username    string `json:"userName"`
		Password    string `json:"password"`
		Host        string `json:"host"`
		Port        int    `json:"port"`
		VirtualHost string `json:"virtualHost"`
	}

	AWSS3Config struct {
		AccessKey string `json:"access_key"`
		SecretKey string `json:"secret_key"`
		Bucket    string `json:"bucket"`
		BaseUrl   string `json:"base_url"`
	}
)

func Init(file *string) (Configuration, error) {
	mu.Lock()
	defer mu.Unlock()

	err := configor.Load(&Cfg, *file)
	if err != nil {
		return Configuration{}, err
	}
	return Cfg, err
}

func GetConfig() Configuration {
	mu.RLock()
	defer mu.RUnlock()
	return Cfg
}
