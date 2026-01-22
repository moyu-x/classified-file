package config

import (
	"github.com/spf13/viper"

	"github.com/moyu-x/classified-file/internal"
)

type Config struct {
	Database struct {
		Path string
	}
	Scanner struct {
		FollowSymlinks bool
	}
	Performance struct {
		Workers int
	}
	Logging struct {
		Level string
		File  string
	}
}

var cfg Config

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	viper.AddConfigPath("$HOME/.classified-file")
	viper.AddConfigPath(".")
	viper.AddConfigPath("/etc/classified-file")

	viper.SetDefault("database.path", internal.DefaultDatabasePath)
	viper.SetDefault("scanner.follow_symlinks", false)
	viper.SetDefault("performance.workers", internal.DefaultWorkers)
	viper.SetDefault("logging.level", "info")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func Get() *Config {
	return &cfg
}
