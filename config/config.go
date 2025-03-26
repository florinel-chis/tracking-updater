package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Magento   MagentoConfig   `mapstructure:"magento"`
	FileWatch FileWatchConfig `mapstructure:"file_watch"`
	Log       LogConfig       `mapstructure:"log"`
}

// MagentoConfig holds Magento API configuration
type MagentoConfig struct {
	BaseURL      string        `mapstructure:"base_url"`
	Token        string        `mapstructure:"token"`
	Timeout      time.Duration `mapstructure:"timeout"`
	MaxRetries   int           `mapstructure:"max_retries"`
	RetryBackoff time.Duration `mapstructure:"retry_backoff"`
}

// FileWatchConfig holds file watching configuration
type FileWatchConfig struct {
	Directory       string        `mapstructure:"directory"`
	FilePattern     string        `mapstructure:"file_pattern"`
	ProcessedDir    string        `mapstructure:"processed_dir"`
	FailedDir       string        `mapstructure:"failed_dir"`
	PollInterval    time.Duration `mapstructure:"poll_interval"`
	MaxConcurrency  int           `mapstructure:"max_concurrency"`
	BatchSize       int           `mapstructure:"batch_size"`
	FileProcessTime time.Duration `mapstructure:"file_process_time"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level      string `mapstructure:"level"`
	Format     string `mapstructure:"format"`
	File       string `mapstructure:"file"`
	EnableFile bool   `mapstructure:"enable_file"`
}

// LoadConfig loads application configuration
func LoadConfig(filePath string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(filePath)

	// Set default values
	setDefaults(v)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Watch for config changes
	v.WatchConfig()

	// Load environment variables
	v.AutomaticEnv()

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	// Magento defaults
	v.SetDefault("magento.timeout", 30*time.Second)
	v.SetDefault("magento.max_retries", 3)
	v.SetDefault("magento.retry_backoff", 1*time.Second)

	// File watching defaults
	v.SetDefault("file_watch.file_pattern", "^\\d{8}_\\d{6}\\.csv$")
	v.SetDefault("file_watch.poll_interval", 5*time.Second)
	v.SetDefault("file_watch.max_concurrency", 5)
	v.SetDefault("file_watch.batch_size", 50)
	v.SetDefault("file_watch.file_process_time", 10*time.Minute)

	// Logging defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")
	v.SetDefault("log.enable_file", false)
}
