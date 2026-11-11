package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/go-viper/mapstructure/v2"
	"github.com/joho/godotenv"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/v2"
)

// root application configs
type Config struct {
	Primary       PrimaryConfig       `koanf:"primary"`
	AppServer     AppServerConfig     `koanf:"app_server"`
	SystemServer  SystemServerConfig  `koanf:"system_server"`
	Database      DatabaseConfig      `koanf:"database"`
	RedisCache    RedisCacheConfig    `koanf:"redis_cache"`
	RedisAsynq    RedisAsynqConfig    `koanf:"redis_asynq"`
	Worker        WorkerConfig        `koanf:"worker"`
	Auth          AuthConfig          `koanf:"auth"`
	Observability ObservabilityConfig `koanf:"observability"`
	Integration   IntegrationConfig   `koanf:"integration"`
}

type PrimaryConfig struct {
	Env         string `koanf:"env" validate:"required,oneof=local development staging production"`
	ServiceName string `koanf:"service_name" validate:"required"`
}

type AppServerConfig struct {
	Port               int           `koanf:"port" validate:"required,min=1,max=65535"`
	ReadTimeout        time.Duration `koanf:"read_timeout" validate:"required,gt=0"`
	WriteTimeout       time.Duration `koanf:"write_timeout" validate:"required,gt=0"`
	IdleTimeout        time.Duration `koanf:"idle_timeout" validate:"required,gt=0"`
	CORSAllowedOrigins []string      `koanf:"cors_allowed_origins" validate:"required"`
}

type SystemServerConfig struct {
	Port         int           `koanf:"port" validate:"required,min=1,max=65535"`
	ReadTimeout  time.Duration `koanf:"read_timeout" validate:"required,gt=0"`
	WriteTimeout time.Duration `koanf:"write_timeout" validate:"required,gt=0"`
	IdleTimeout  time.Duration `koanf:"idle_timeout" validate:"required,gt=0"`
	// System Server doesn't need CORS
}

type DatabaseConfig struct {
	Host            string        `koanf:"host" validate:"required"`
	Port            int           `koanf:"port" validate:"required,min=1,max=65535"`
	User            string        `koanf:"user" validate:"required"`
	Password        string        `koanf:"password"`
	Name            string        `koanf:"name" validate:"required"`
	SSLMode         string        `koanf:"ssl_mode" validate:"required,oneof=disable require verify-ca verify-full prefer allow"`
	MaxOpenConns    int           `koanf:"max_open_conns" validate:"required,gt=0"`
	MaxIdleConns    int           `koanf:"max_idle_conns" validate:"required,gte=0"`
	ConnMaxLifetime time.Duration `koanf:"conn_max_lifetime" validate:"required,gt=0"`
	ConnMaxIdleTime time.Duration `koanf:"conn_max_idle_time" validate:"required,gt=0"`
}

type RedisCacheConfig struct {
	Address      string `koanf:"address" validate:"required"`
	Password     string `koanf:"password" validate:"required"`
	Database     int    `koanf:"database" validate:"gte=0,lte=15"`
	PoolSize     int    `koanf:"pool_size" validate:"required,gt=0"`
	MinIdleConns int    `koanf:"min_idle_conns" validate:"gte=0"`
}

// Asynq manages its own pool internally, so we just need the connection details
type RedisAsynqConfig struct {
	Address  string `koanf:"address" validate:"required"`
	Password string `koanf:"password" validate:"required"`
	Database int    `koanf:"database" validate:"gte=0,lte=15"`
}

type WorkerConfig struct {
	Concurrency int `koanf:"concurrency" validate:"required,gt=0"`
}

type AuthConfig struct {
	SecretKey string `koanf:"secret_key" validate:"required,min=32"`
}

// observability configs
type ObservabilityConfig struct {
	Logging LoggingConfig `koanf:"logging"`
	OTel    OTelConfig    `koanf:"otel"`
}

type LoggingConfig struct {
	Level              string        `koanf:"level" validate:"required,oneof=debug info warn error"`
	Format             string        `koanf:"format" validate:"required,oneof=json console"`
	SlowQueryThreshold time.Duration `koanf:"slow_query_threshold" validate:"required,gte=0"`
}

type OTelConfig struct {
	Enabled             bool          `koanf:"enabled"`
	TracesEnabled       bool          `koanf:"traces_enabled" validate:"required_if=Enabled true"`
	MetricsEnabled      bool          `koanf:"metrics_enabled" validate:"required_if=Enabled true"`
	LogsEnabled         bool          `koanf:"logs_enabled" validate:"required_if=Enabled true"`
	ServiceName         string        `koanf:"service_name" validate:"required_if=Enabled true"`
	Environment         string        `koanf:"environment" validate:"required_if=Enabled true,oneof=local development staging production"`
	Exporter            string        `koanf:"exporter" validate:"required_if=Enabled true"`
	Endpoint            string        `koanf:"endpoint" validate:"required_if=Enabled true"`
	Protocol            string        `koanf:"protocol" validate:"required_if=Enabled true,omitempty,oneof=http/protobuf grpc"`
	Headers             string        `koanf:"headers"`
	ExportTimeout       time.Duration `koanf:"export_timeout" validate:"required_if=Enabled true,omitempty,gt=0"`
	BatchTimeout        time.Duration `koanf:"batch_timeout" validate:"required_if=Enabled true,omitempty,gt=0"`
	MaxExportBatchSize  int           `koanf:"max_export_batch_size" validate:"required_if=Enabled true,omitempty,gt=0"`
	MaxQueueSize        int           `koanf:"max_queue_size" validate:"required_if=Enabled true,omitempty,gt=0"`
	TracesSamplingRatio float64       `koanf:"traces_sampling_ratio" validate:"required_if=Enabled true,gte=0,lte=1"`
}

// external integration configs
type IntegrationConfig struct {
	Email EmailIntegrationConfig `koanf:"email"`
}

type EmailIntegrationConfig struct {
	EmailAPIKey string `koanf:"api_key" validate:"required"`
}

// loads the config from environment variables and an optional .env file
func LoadConfig() (*Config, error) {
	k := koanf.New(".")

	// 1. if a local .env file exists, load its variables directly into the OS process environment
	if _, err := os.Stat(".env"); err == nil {
		if err := godotenv.Load(".env"); err != nil {
			return nil, fmt.Errorf("failed to load .env file into environment: %w", err)
		}
	}

	// 2. load live OS environment variables
	err := k.Load(env.Provider(".", env.Opt{
		Prefix: "MYAPP_",
		TransformFunc: func(k, v string) (string, any) {
			s := strings.TrimPrefix(k, "MYAPP_")
			s = strings.ReplaceAll(s, "__", ".")
			return strings.ToLower(s), v
		},
	}), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	var cfg Config

	// 3. define decode hooks using the maintained v2 mapstructure package
	// environment variables are ALWAYS strings ("15s", "30")
	// Go structs often need specific types like time.Duration
	// this hook intercepts string values during unmarshaling and parses them into time.Duration.
	decodeHook := mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
	)

	// 4. unmarshal configuration with koanf.UnmarshalConf and custom DecoderConfig
	if err := k.UnmarshalWithConf("", &cfg, koanf.UnmarshalConf{
		Tag: "koanf",
		DecoderConfig: &mapstructure.DecoderConfig{
			DecodeHook:       decodeHook,
			TagName:          "koanf",
			WeaklyTypedInput: true,
		},
	}); err != nil {
		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)
	}

	// 5. validate configuration using go-playground/validator
	// this checks all rules like validate:"required" or validate:"oneof=..."
	// ensuring the app fails fast on startup if anything is missing or misconfigured.
	validate := validator.New()
	if err := validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &cfg, nil
}

// config Utils functions

func (cfg *ObservabilityConfig) IsProduction() bool {
	return cfg.OTel.Environment == "production"
}

func (cfg *ObservabilityConfig) GetLogLevel() string {
	if cfg.Logging.Level == "" {
		return "info"
	}
	return cfg.Logging.Level
}
