package processor

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"tracking-updater/config"
	"tracking-updater/internal/api"
	"tracking-updater/internal/model"
)

// CSVProcessor handles processing of CSV files
type CSVProcessor struct {
	config         *config.Config
	logger         *logrus.Logger
	magentoClient  *api.MagentoClient
	workChan       chan string
	wg             sync.WaitGroup
	processedFiles map[string]bool
	mutex          sync.Mutex
}

// NewCSVProcessor creates a new CSV processor
func NewCSVProcessor(cfg *config.Config, logger *logrus.Logger, magentoClient *api.MagentoClient) *CSVProcessor {
	return &CSVProcessor{
		config:         cfg,
		logger:         logger,
		magentoClient:  magentoClient,
		workChan:       make(chan string, 100),
		processedFiles: make(map[string]bool),
	}
}

// Start begins processing files
func (p *CSVProcessor) Start() {
	p.logger.Info("Starting CSV processor")
	
	// Create the processed and failed directories if they don't exist
	if err := os.MkdirAll(p.config.FileWatch.ProcessedDir, 0755); err != nil {
		p.logger.WithError(err).Error("Failed to create processed directory")
	}
	
	if err := os.MkdirAll(p.config.FileWatch.FailedDir, 0755); err != nil {
		p.logger.WithError(err).Error("Failed to create failed directory")
	}

	// Start worker goroutines
	for i := 0; i < p.config.FileWatch.MaxConcurrency; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Stop stops the processor
func (p *CSVProcessor) Stop() {
	p.logger.Info("Stopping CSV processor")
	close(p.workChan)
	p.wg.Wait()
}

// ProcessFile queues a file for processing
func (p *CSVProcessor) ProcessFile(filePath string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// Check if the file has already been processed
	if p.processedFiles[filePath] {
		p.logger.WithField("file", filePath).Info("File already processed, skipping")
		return
	}

	p.processedFiles[filePath] = true
	p.workChan <- filePath
}

// worker processes files from the work channel
func (p *CSVProcessor) worker(id int) {
	defer p.wg.Done()

	log := p.logger.WithField("worker_id", id)
	log.Info("Starting worker")

	for filePath := range p.workChan {
		log := log.WithField("file", filePath)
		log.Info("Processing file")

		success := p.processCSVFile(filePath)
		
		// Move the file to the appropriate directory
		destinationDir := p.config.FileWatch.ProcessedDir
		if !success {
			destinationDir = p.config.FileWatch.FailedDir
		}

		fileName := filepath.Base(filePath)
		destinationPath := filepath.Join(destinationDir, fileName)
		
		if err := os.Rename(filePath, destinationPath); err != nil {
			log.WithError(err).Error("Failed to move file")
		} else {
			log.WithField("destination", destinationPath).Info("Moved file")
		}
	}

	log.Info("Worker stopped")
}

// processCSVFile processes a single CSV file
func (p *CSVProcessor) processCSVFile(filePath string) bool {
	log := p.logger.WithField("file", filePath)
	startTime := time.Now()

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		log.WithError(err).Error("Failed to open file")
		return false
	}
	defer file.Close()

	// Create a new CSV reader
	reader := csv.NewReader(file)
	
	// Read the header
	header, err := reader.Read()
	if err != nil {
		log.WithError(err).Error("Failed to read CSV header")
		return false
	}

	// Check if the CSV has the required columns
	indices := getColumnIndices(header)
	if indices.orderNumber == -1 || indices.trackingNumber == -1 || indices.carrierCode == -1 || indices.title == -1 {
		log.Error("CSV file does not have required columns")
		return false
	}

	// Process each row
	rowCount := 0
	errorCount := 0

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.WithError(err).Error("Failed to read CSV row")
			errorCount++
			continue
		}

		// Process the row
		if err := p.processRow(row, indices); err != nil {
			log.WithError(err).Warn("Failed to process row")
			errorCount++
		}

		rowCount++
	}

	elapsed := time.Since(startTime)
	log.WithFields(logrus.Fields{
		"elapsed":      elapsed,
		"row_count":    rowCount,
		"error_count":  errorCount,
		"success_rate": fmt.Sprintf("%.2f%%", 100*(float64(rowCount-errorCount)/float64(rowCount))),
	}).Info("Completed processing file")

	// Return true if there were no errors or if the error count is acceptable
	return errorCount == 0 || float64(errorCount)/float64(rowCount) < 0.05 // 5% error threshold
}

// columnIndices holds the indices of the required columns
type columnIndices struct {
	orderNumber    int
	trackingNumber int
	carrierCode    int
	title          int
}

// getColumnIndices returns the indices of the required columns
func getColumnIndices(header []string) columnIndices {
	indices := columnIndices{
		orderNumber:    -1,
		trackingNumber: -1,
		carrierCode:    -1,
		title:          -1,
	}

	for i, col := range header {
		switch col {
		case "order_number":
			indices.orderNumber = i
		case "tracking_number":
			indices.trackingNumber = i
		case "carrier_code":
			indices.carrierCode = i
		case "title":
			indices.title = i
		}
	}

	return indices
}

// processRow processes a single row from the CSV file
func (p *CSVProcessor) processRow(row []string, indices columnIndices) error {
	// Extract tracking information from the row
	trackingInfo := &model.TrackingInfo{
		OrderNumber:    row[indices.orderNumber],
		TrackingNumber: row[indices.trackingNumber],
		CarrierCode:    row[indices.carrierCode],
		Title:          row[indices.title],
	}

	// Validate the tracking information
	if err := trackingInfo.Validate(); err != nil {
		return fmt.Errorf("invalid tracking info: %w", err)
	}

	log := p.logger.WithFields(logrus.Fields{
		"order_number":    trackingInfo.OrderNumber,
		"tracking_number": trackingInfo.TrackingNumber,
		"carrier_code":    trackingInfo.CarrierCode,
	})

	log.Info("Processing tracking information")

	// Get the order by increment ID (order number)
	order, err := p.magentoClient.GetOrderByIncrementID(trackingInfo.OrderNumber)
	if err != nil {
		return fmt.Errorf("failed to get order: %w", err)
	}

	// Get shipments for the order
	shipments, err := p.magentoClient.GetShipmentsByOrderID(order.EntityID)
	if err != nil {
		return fmt.Errorf("failed to get shipments: %w", err)
	}

	// Skip if no shipments found
	if len(shipments) == 0 {
		log.Warn("No shipments found for order, skipping tracking update")
		return nil
	}

	// Use the first shipment (as per requirement, each order has only 1 shipment)
	shipment := shipments[0]
	
	// Create tracking information for Magento API
	track := &model.MagentoTrack{
		OrderID:     order.EntityID,
		TrackNumber: trackingInfo.TrackingNumber,
		Title:       trackingInfo.Title,
		CarrierCode: trackingInfo.CarrierCode,
	}

	// Add tracking to the shipment
	if err := p.magentoClient.AddTrackingToShipment(shipment.EntityID, track); err != nil {
		return fmt.Errorf("failed to add tracking: %w", err)
	}

	log.Info("Successfully updated tracking information")
	return nil
}
