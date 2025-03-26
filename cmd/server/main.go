package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"tracking-updater/config"
	"tracking-updater/internal/api"
	"tracking-updater/internal/file"
	"tracking-updater/internal/processor"
	"tracking-updater/pkg/logger"

	"github.com/sirupsen/logrus"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "config.yaml", "Path to config file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		fmt.Printf("Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Set up logger
	log := logger.Setup(&cfg.Log)
	log.Info("Starting tracking-updater service")

	// Create Magento API client
	magentoClient := api.NewMagentoClient(&cfg.Magento, log)

	// Create CSV processor
	csvProcessor := processor.NewCSVProcessor(cfg, log, magentoClient)
	csvProcessor.Start()
	defer csvProcessor.Stop()

	// Create file watcher
	fileWatcher, err := file.NewWatcher(&cfg.FileWatch, log, csvProcessor)
	if err != nil {
		log.WithError(err).Fatal("Failed to create file watcher")
	}

	// Start the file watcher
	if err := fileWatcher.Start(); err != nil {
		log.WithError(err).Fatal("Failed to start file watcher")
	}
	defer fileWatcher.Stop()

	log.WithFields(logrus.Fields{
		"watch_dir":     cfg.FileWatch.Directory,
		"file_pattern":  cfg.FileWatch.FilePattern,
		"processed_dir": cfg.FileWatch.ProcessedDir,
		"failed_dir":    cfg.FileWatch.FailedDir,
	}).Info("Service started successfully")

	// Wait for a signal to shut down
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Info("Shutting down service")
}
