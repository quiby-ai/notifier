package config

import (
	"log"

	"github.com/spf13/viper"
)

type HTTPCfg struct {
	Addr           string   `mapstructure:"addr"`
	AllowedOrigins []string `mapstructure:"allowed_origins"`
}
type KCfg struct {
	Brokers []string `mapstructure:"brokers"`
	GroupID string   `mapstructure:"group_id"`
	Topic   string   `mapstructure:"topic"`
}
type WSCfg struct {
	PingIntervalSec int `mapstructure:"ping_interval_sec"`
	WriteTimeoutSec int `mapstructure:"write_timeout_sec"`
	MaxMessageBytes int `mapstructure:"max_message_bytes"`
}
type SecCfg struct {
	RequireAuth bool `mapstructure:"require_auth"`
}

type Config struct {
	HTTP     HTTPCfg `mapstructure:"http"`
	Kafka    KCfg    `mapstructure:"kafka"`
	WS       WSCfg   `mapstructure:"ws"`
	Security SecCfg  `mapstructure:"security"`
}

func Load() Config {
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath("/")

	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("failed to read config.toml: %v", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("failed to unmarshal config: %v", err)
	}
	return cfg
}
