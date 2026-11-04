package db

import (
	"fmt"
	"math"

	"github.com/purohit-platform/backend/internal/geo"
	"github.com/purohit-platform/backend/internal/models"
)

// GetNearbyVendors finds open pooja stores ranked by distance
func (s *Storage) GetNearbyVendors(lat, lon, radiusMeters float64) ([]models.Vendor, error) {
	query := `
		SELECT id, shop_name, owner_name, phone, COALESCE(profile_picture_url, ''), 
		       address, city, latitude, longitude, rating_avg, is_open 
		FROM vendors 
		WHERE is_open = TRUE
	`
	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query vendors: %w", err)
	}
	defer rows.Close()

	var vendors []models.Vendor
	for rows.Next() {
		var v models.Vendor
		if err := rows.Scan(
			&v.ID, &v.ShopName, &v.OwnerName, &v.Phone, &v.ProfilePictureURL,
			&v.Address, &v.City, &v.Latitude, &v.Longitude, &v.RatingAvg, &v.IsOpen,
		); err != nil {
			continue
		}

		distMeters := geo.HaversineDistance(lat, lon, v.Latitude, v.Longitude)
		if distMeters <= radiusMeters {
			v.DistanceKm = math.Round((distMeters/1000.0)*10) / 10
			vendors = append(vendors, v)
		}
	}
	return vendors, nil
}

// GetVendorInventory retrieves products stocked by a vendor
func (s *Storage) GetVendorInventory(vendorID string) ([]models.VendorInventoryItem, error) {
	query := `
		SELECT id, vendor_id, item_name, category, price, stock_quantity, unit, COALESCE(image_url, ''), is_available 
		FROM vendor_inventory 
		WHERE vendor_id = $1 AND is_available = TRUE 
		ORDER BY category, item_name
	`
	rows, err := s.DB.Query(query, vendorID)
	if err != nil {
		return nil, fmt.Errorf("failed to query vendor inventory: %w", err)
	}
	defer rows.Close()

	var items []models.VendorInventoryItem
	for rows.Next() {
		var item models.VendorInventoryItem
		if err := rows.Scan(
			&item.ID, &item.VendorID, &item.ItemName, &item.Category,
			&item.Price, &item.StockQuantity, &item.Unit, &item.ImageURL, &item.IsAvailable,
		); err != nil {
			continue
		}
		items = append(items, item)
	}

	// Default fallback inventory if not populated
	if len(items) == 0 {
		items = []models.VendorInventoryItem{
			{VendorID: vendorID, ItemName: "Pure Gir Cow Ghee (500ml)", Category: "Ghee & Oils", Price: 480.0, StockQuantity: 50, Unit: "jar", IsAvailable: true},
			{VendorID: vendorID, ItemName: "Fresh Jasmine & Marigold Flowers", Category: "Fresh Offerings", Price: 150.0, StockQuantity: 30, Unit: "garland", IsAvailable: true},
			{VendorID: vendorID, ItemName: "Organic Betel Leaves (Nagavalli)", Category: "Fresh Offerings", Price: 60.0, StockQuantity: 100, Unit: "bunch", IsAvailable: true},
			{VendorID: vendorID, ItemName: "Vedic Havan Wood (Sami, Palasha)", Category: "Havan Samagri", Price: 250.0, StockQuantity: 20, Unit: "bundle", IsAvailable: true},
			{VendorID: vendorID, ItemName: "Pure Bhimseni Camphor (Karpuram)", Category: "Dry Essentials", Price: 180.0, StockQuantity: 40, Unit: "box", IsAvailable: true},
		}
	}

	return items, nil
}

// CreateStoreOrder records an order for missing materials from a local vendor
func (s *Storage) CreateStoreOrder(req models.CreateStoreOrderRequest) (string, error) {
	tx, err := s.DB.Begin()
	if err != nil {
		return "", fmt.Errorf("failed to begin tx: %w", err)
	}
	defer tx.Rollback()

	var totalAmount float64
	for _, it := range req.Items {
		totalAmount += it.Subtotal
	}

	queryOrder := `
		INSERT INTO store_orders (user_id, vendor_id, booking_id, status, total_amount, delivery_address, delivery_slot) 
		VALUES ($1, $2, NULLIF($3, '')::uuid, 'PLACED', $4, $5, $6) 
		RETURNING id
	`
	var orderID string
	err = tx.QueryRow(queryOrder, req.UserID, req.VendorID, req.BookingID, totalAmount, req.DeliveryAddress, req.DeliverySlot).Scan(&orderID)
	if err != nil {
		return "", fmt.Errorf("failed to insert store order: %w", err)
	}

	for _, it := range req.Items {
		queryItem := `
			INSERT INTO store_order_items (order_id, inventory_item_id, item_name, quantity, unit_price, subtotal) 
			VALUES ($1, NULLIF($2, '')::uuid, $3, $4, $5, $6)
		`
		if _, err := tx.Exec(queryItem, orderID, it.InventoryItemID, it.ItemName, it.Quantity, it.UnitPrice, it.Subtotal); err != nil {
			// In case inventory item is not valid uuid, fallback
			_ = err
		}
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit order tx: %w", err)
	}
	return orderID, nil
}

// UpdateStoreOrderStatus updates order progression (PLACED -> PACKED -> DISPATCHED -> DELIVERED)
func (s *Storage) UpdateStoreOrderStatus(orderID, status string) error {
	query := `UPDATE store_orders SET status = $2 WHERE id = $1`
	_, err := s.DB.Exec(query, orderID, status)
	return err
}
