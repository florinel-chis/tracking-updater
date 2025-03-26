package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"tracking-updater/config"
	"tracking-updater/internal/model"

	"github.com/sirupsen/logrus"
)

// MagentoClient handles communication with the Magento 2 API
type MagentoClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
	maxRetries int
	backoff    time.Duration
	logger     *logrus.Logger
}

// NewMagentoClient creates a new Magento API client
func NewMagentoClient(cfg *config.MagentoConfig, logger *logrus.Logger) *MagentoClient {
	return &MagentoClient{
		baseURL: cfg.BaseURL,
		token:   cfg.Token,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		maxRetries: cfg.MaxRetries,
		backoff:    cfg.RetryBackoff,
		logger:     logger,
	}
}

// GetOrderByIncrementID retrieves order details by increment ID (order number)
func (c *MagentoClient) GetOrderByIncrementID(incrementID string) (*model.MagentoOrder, error) {
	log := c.logger.WithFields(logrus.Fields{
		"function":     "GetOrderByIncrementID",
		"increment_id": incrementID,
	})

	log.Info("Retrieving order details")

	// Build the search criteria to find order by increment_id
	endpoint := fmt.Sprintf("%s/orders", c.baseURL)
	params := url.Values{}
	params.Add("searchCriteria[filter_groups][0][filters][0][field]", "increment_id")
	params.Add("searchCriteria[filter_groups][0][filters][0][value]", incrementID)
	params.Add("searchCriteria[filter_groups][0][filters][0][condition_type]", "eq")

	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		log.WithError(err).Error("Failed to create request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	var response model.MagentoOrderResponse
	if err := c.doRequest(req, &response); err != nil {
		log.WithError(err).Error("Failed to get order")
		return nil, fmt.Errorf("failed to get order: %w", err)
	}

	if response.Total == 0 || len(response.Items) == 0 {
		log.Warn("Order not found")
		return nil, fmt.Errorf("order with increment_id %s not found", incrementID)
	}

	log.WithField("order_id", response.Items[0].EntityID).Info("Order found")
	return &response.Items[0], nil
}

// GetShipmentsByOrderID retrieves shipments for a specific order
func (c *MagentoClient) GetShipmentsByOrderID(orderID int) ([]model.MagentoShipment, error) {
	log := c.logger.WithFields(logrus.Fields{
		"function": "GetShipmentsByOrderID",
		"order_id": orderID,
	})

	log.Info("Retrieving shipments for order")

	endpoint := fmt.Sprintf("%s/shipments", c.baseURL)
	params := url.Values{}
	params.Add("searchCriteria[filter_groups][0][filters][0][field]", "order_id")
	params.Add("searchCriteria[filter_groups][0][filters][0][value]", fmt.Sprintf("%d", orderID))
	params.Add("searchCriteria[filter_groups][0][filters][0][condition_type]", "eq")

	fullURL := fmt.Sprintf("%s?%s", endpoint, params.Encode())

	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		log.WithError(err).Error("Failed to create request")
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	var response model.MagentoShipmentResponse
	if err := c.doRequest(req, &response); err != nil {
		log.WithError(err).Error("Failed to get shipments")
		return nil, fmt.Errorf("failed to get shipments: %w", err)
	}

	if response.Total == 0 || len(response.Items) == 0 {
		log.Warn("No shipments found")
		return nil, nil
	}

	log.WithField("shipment_count", len(response.Items)).Info("Shipments found")
	return response.Items, nil
}

// AddTrackingToShipment adds tracking information to a shipment
func (c *MagentoClient) AddTrackingToShipment(shipmentID int, track *model.MagentoTrack) error {
	log := c.logger.WithFields(logrus.Fields{
		"function":    "AddTrackingToShipment",
		"shipment_id": shipmentID,
		"tracking_no": track.TrackNumber,
	})

	log.Info("Adding tracking information to shipment")

	// Set the shipment ID
	track.ParentID = shipmentID

	// Create the proper request structure with "entity" wrapper
	requestBody := map[string]interface{}{
		"entity": track,
	}

	endpoint := fmt.Sprintf("%s/shipment/track", c.baseURL)

	body, err := json.Marshal(requestBody)
	if err != nil {
		log.WithError(err).Error("Failed to marshal tracking data")
		return fmt.Errorf("failed to marshal tracking data: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		log.WithError(err).Error("Failed to create request")
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	var response interface{}
	if err := c.doRequest(req, &response); err != nil {
		log.WithError(err).Error("Failed to add tracking")
		return fmt.Errorf("failed to add tracking: %w", err)
	}

	log.Info("Successfully added tracking information")
	return nil
}

// doRequest performs the HTTP request with retry logic
func (c *MagentoClient) doRequest(req *http.Request, v interface{}) error {
	var resp *http.Response
	var err error
	attempts := 0

	for attempts < c.maxRetries {
		attempts++

		resp, err = c.httpClient.Do(req)
		if err != nil {
			c.logger.WithError(err).WithField("attempt", attempts).
				Warn("Request failed, retrying...")

			if attempts < c.maxRetries {
				time.Sleep(c.backoff * time.Duration(attempts))
				continue
			}
			return fmt.Errorf("request failed after %d attempts: %w", attempts, err)
		}

		defer resp.Body.Close()

		// Check if the response code is not successful
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(resp.Body)
			errMsg := fmt.Sprintf("api error (status: %d): %s", resp.StatusCode, string(body))

			c.logger.WithField("status_code", resp.StatusCode).
				WithField("attempt", attempts).
				WithField("response", string(body)).
				Warn("API returned error, retrying...")

			if attempts < c.maxRetries {
				time.Sleep(c.backoff * time.Duration(attempts))
				// Need to recreate the request body for retries
				if req.Body != nil {
					req.Body = io.NopCloser(bytes.NewBuffer(body))
				}
				continue
			}
			return fmt.Errorf(errMsg)
		}

		// Successful response
		return json.NewDecoder(resp.Body).Decode(v)
	}

	return fmt.Errorf("max retries exceeded")
}
