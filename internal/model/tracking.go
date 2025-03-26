package model

import (
	"fmt"
	"strings"
)

// TrackingInfo represents tracking information from a CSV file
type TrackingInfo struct {
	OrderNumber    string `json:"order_number"`
	TrackingNumber string `json:"tracking_number"`
	CarrierCode    string `json:"carrier_code"`
	Title          string `json:"title"`
}

// Validate checks if all required fields are present
func (t *TrackingInfo) Validate() error {
	if strings.TrimSpace(t.OrderNumber) == "" {
		return fmt.Errorf("order number is required")
	}
	if strings.TrimSpace(t.TrackingNumber) == "" {
		return fmt.Errorf("tracking number is required")
	}
	if strings.TrimSpace(t.CarrierCode) == "" {
		return fmt.Errorf("carrier code is required")
	}
	if strings.TrimSpace(t.Title) == "" {
		return fmt.Errorf("title is required")
	}
	return nil
}

// MagentoOrder represents a simplified Magento order structure
type MagentoOrder struct {
	EntityID            int           `json:"entity_id"`
	IncrementID         string        `json:"increment_id"`
	Status              string        `json:"status"`
	ExtensionAttributes ExtAttributes `json:"extension_attributes"`
}

// ExtAttributes represents Magento order extension attributes
type ExtAttributes struct {
	ShippingAssignments []ShippingAssignment `json:"shipping_assignments"`
}

// ShippingAssignment represents Magento shipping assignment
type ShippingAssignment struct {
	Shipping Shipping `json:"shipping"`
}

// Shipping represents Magento shipping information
type Shipping struct {
	Address Address `json:"address"`
	Method  string  `json:"method"`
}

// Address represents a shipping address
type Address struct {
	FirstName  string   `json:"firstname"`
	LastName   string   `json:"lastname"`
	Street     []string `json:"street"`
	City       string   `json:"city"`
	PostalCode string   `json:"postcode"`
	CountryID  string   `json:"country_id"`
	Telephone  string   `json:"telephone"`
}

// MagentoShipment represents a simplified Magento shipment structure
type MagentoShipment struct {
	EntityID    int    `json:"entity_id"`
	IncrementID string `json:"increment_id"`
	OrderID     int    `json:"order_id"`
}

// MagentoTrack represents a Magento shipment track
type MagentoTrack struct {
	OrderID     int    `json:"order_id"`
	ParentID    int    `json:"parent_id,omitempty"` // Shipment ID
	TrackNumber string `json:"track_number"`
	Title       string `json:"title"`
	CarrierCode string `json:"carrier_code"`
}

// MagentoShipmentResponse represents the response from Magento API for shipment queries
type MagentoShipmentResponse struct {
	Items []MagentoShipment `json:"items"`
	Total int               `json:"total_count"`
}

// MagentoOrderResponse represents the response from Magento API for order queries
type MagentoOrderResponse struct {
	Items []MagentoOrder `json:"items"`
	Total int            `json:"total_count"`
}
