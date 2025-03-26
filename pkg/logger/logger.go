package logger

import (
	"io"
	"os"
	"strings"

	"tracking-updater/config"

	"github.com/sirupsen/logrus"
)

// Setup initializes the logger with the given configuration
func Setup(cfg *config.LogConfig) *logrus.Logger {
	log := logrus.New()

	// Set log level
	level, err := logrus.ParseLevel(strings.ToLower(cfg.Level))
	if err != nil {
		level = logrus.InfoLevel
	}
	log.SetLevel(level)

	// Set log format
	switch strings.ToLower(cfg.Format) {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	default:
		log.SetFormatter(&logrus.TextFormatter{
			FullTimestamp:   true,
			TimestampFormat: "2006-01-02T15:04:05.000Z07:00",
		})
	}

	// Set log output
	if cfg.EnableFile && cfg.File != "" {
		file, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			log.SetOutput(io.MultiWriter(os.Stdout, file))
		} else {
			log.Warn("Failed to log to file, using default stderr")
		}
	} else {
		log.SetOutput(os.Stdout)
	}

	return log
}

// NewWithFields creates a new logrus entry with pre-defined fields
func NewWithFields(log *logrus.Logger, fields map[string]interface{}) *logrus.Entry {
	return log.WithFields(fields)
}
