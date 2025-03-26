package file

import (
	"os"
	"path/filepath"
	"regexp"
	"time"

	"tracking-updater/config"
	"tracking-updater/internal/processor"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

// Watcher monitors a directory for new CSV files
type Watcher struct {
	config      *config.FileWatchConfig
	logger      *logrus.Logger
	processor   *processor.CSVProcessor
	watcher     *fsnotify.Watcher
	stopChan    chan struct{}
	filePattern *regexp.Regexp
	isRunning   bool
}

// NewWatcher creates a new file watcher
func NewWatcher(cfg *config.FileWatchConfig, logger *logrus.Logger, processor *processor.CSVProcessor) (*Watcher, error) {
	// Compile the file pattern regex
	pattern, err := regexp.Compile(cfg.FilePattern)
	if err != nil {
		return nil, err
	}

	// Create the fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		config:      cfg,
		logger:      logger,
		processor:   processor,
		watcher:     watcher,
		stopChan:    make(chan struct{}),
		filePattern: pattern,
		isRunning:   false,
	}, nil
}

// Start begins watching the directory for new files
func (w *Watcher) Start() error {
	if w.isRunning {
		return nil
	}

	w.logger.WithField("directory", w.config.Directory).Info("Starting file watcher")

	// Ensure the directory exists
	if err := os.MkdirAll(w.config.Directory, 0755); err != nil {
		return err
	}

	// Add the directory to the watcher
	if err := w.watcher.Add(w.config.Directory); err != nil {
		return err
	}

	w.isRunning = true

	// Start the file watcher goroutine
	go w.watchLoop()

	// Process any existing files on startup
	go w.processExistingFiles()

	return nil
}

// Stop stops the file watcher
func (w *Watcher) Stop() {
	if !w.isRunning {
		return
	}

	w.logger.Info("Stopping file watcher")
	close(w.stopChan)
	w.watcher.Close()
	w.isRunning = false
}

// watchLoop monitors the directory for new files
func (w *Watcher) watchLoop() {
	for {
		select {
		case <-w.stopChan:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			w.handleEvent(event)
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logger.WithError(err).Error("Error watching files")
		}
	}
}

// handleEvent processes a file system event
func (w *Watcher) handleEvent(event fsnotify.Event) {
	// Only process create and write events
	if event.Op&(fsnotify.Create|fsnotify.Write) == 0 {
		return
	}

	// Check if it's a CSV file matching our pattern
	if !w.isTargetFile(event.Name) {
		return
	}

	w.logger.WithField("file", event.Name).Info("New CSV file detected")

	// Wait a moment to ensure the file is fully written
	time.Sleep(500 * time.Millisecond)

	// Check if the file is accessible and not being written
	if !w.isFileReady(event.Name) {
		return
	}

	// Process the file
	w.processor.ProcessFile(event.Name)
}

// processExistingFiles processes any existing files in the directory
func (w *Watcher) processExistingFiles() {
	w.logger.Info("Processing existing files")

	files, err := filepath.Glob(filepath.Join(w.config.Directory, "*.csv"))
	if err != nil {
		w.logger.WithError(err).Error("Failed to list existing files")
		return
	}

	for _, file := range files {
		if w.isTargetFile(file) && w.isFileReady(file) {
			w.logger.WithField("file", file).Info("Processing existing file")
			w.processor.ProcessFile(file)
		}
	}
}

// isTargetFile checks if a file matches our target pattern
func (w *Watcher) isTargetFile(path string) bool {
	// Check if it's a regular file
	fileInfo, err := os.Stat(path)
	if err != nil || fileInfo.IsDir() {
		return false
	}

	// Check if it matches our file pattern
	fileName := filepath.Base(path)
	return w.filePattern.MatchString(fileName)
}

// isFileReady checks if a file is fully written and not being modified
func (w *Watcher) isFileReady(path string) bool {
	// Get initial file info
	initialInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Wait a moment and check again
	time.Sleep(1 * time.Second)

	// Get updated file info
	currentInfo, err := os.Stat(path)
	if err != nil {
		return false
	}

	// If size or modification time has changed, file is still being written
	return initialInfo.Size() == currentInfo.Size() &&
		initialInfo.ModTime() == currentInfo.ModTime()
}
