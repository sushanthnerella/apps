package db

import (
	"errors"
	"fmt"
	"math"

	"github.com/lib/pq"
	"github.com/purohit-platform/backend/internal/models"
)

var (
	ErrStoreItemUnknown         = errors.New("store_item_unknown")
	ErrStoreItemVendorMismatch   = errors.New("store_item_vendor_mismatch")
	ErrStoreItemOutOfStock      = errors.New("store_item_out_of_stock")
	ErrStoreItemInvalidQuantity = errors.New("store_item_invalid_quantity")
)

// GetVendorInventory retrieves products stocked by a vendor in minor units without floating-point conversion.
func (s *Storage) GetVendorInventory(vendorID string) ([]models.VendorInventoryItem, error) {
	query := `
		SELECT id, vendor_id, item_name, category, COALESCE(price_minor, 0), stock_quantity, unit, COALESCE(image_url, ''), is_available 
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
			&item.PriceMinor, &item.StockQuantity, &item.Unit, &item.ImageURL, &item.IsAvailable,
		); err != nil {
			continue
		}
		item.Currency = "INR"
		items = append(items, item)
	}

	return items, rows.Err()
}

// CreateStoreOrder validates vendor inventory server-side and writes order records in pure minor units.
// Returns order ID and total amount in minor units.
func (s *Storage) CreateStoreOrder(req models.CreateStoreOrderRequest) (string, int64, error) {
	if req.VendorID == "" {
		return "", 0, errors.New("vendor_id is required")
	}
	if len(req.Items) == 0 {
		return "", 0, errors.New("order must contain at least one item")
	}

	// Validate quantities and collect item IDs
	itemIDs := make([]string, 0, len(req.Items))
	qtyMap := make(map[string]int, len(req.Items))
	for _, it := range req.Items {
		id := it.InventoryItemID
		if id == "" {
			id = it.ItemID
		}
		if id == "" {
			return "", 0, ErrStoreItemUnknown
		}
		if it.Quantity < 1 || it.Quantity > 100 {
			return "", 0, ErrStoreItemInvalidQuantity
		}
		qtyMap[id] += it.Quantity
		itemIDs = append(itemIDs, id)
	}

	tx, err := s.DB.Begin()
	if err != nil {
		return "", 0, fmt.Errorf("failed to begin order tx: %w", err)
	}
	defer tx.Rollback()

	// Query server-authoritative inventory prices and stock using row-level locking
	inventoryQuery := `
		SELECT id, vendor_id, item_name, COALESCE(price_minor, 0), stock_quantity, is_available
		FROM vendor_inventory
		WHERE id = ANY($1)
		FOR UPDATE
	`
	rows, err := tx.Query(inventoryQuery, pq.Array(itemIDs))
	if err != nil {
		return "", 0, fmt.Errorf("failed to query vendor inventory: %w", err)
	}
	defer rows.Close()

	type invRecord struct {
		id            string
		vendorID      string
		itemName      string
		priceMinor    int64
		stockQuantity int
		isAvailable   bool
	}

	foundItems := make(map[string]invRecord)
	for rows.Next() {
		var rec invRecord
		if err := rows.Scan(&rec.id, &rec.vendorID, &rec.itemName, &rec.priceMinor, &rec.stockQuantity, &rec.isAvailable); err != nil {
			return "", 0, fmt.Errorf("failed to scan inventory item: %w", err)
		}
		foundItems[rec.id] = rec
	}
	if err := rows.Err(); err != nil {
		return "", 0, fmt.Errorf("inventory query iteration error: %w", err)
	}

	// Server-side validation: verify all items exist, match the vendor, and have sufficient stock
	var totalAmountMinor int64
	type validatedItem struct {
		invID         string
		name          string
		qty           int
		unitPrice     int64
		subtotalMinor int64
	}
	validated := make([]validatedItem, 0, len(req.Items))

	for id, qty := range qtyMap {
		rec, exists := foundItems[id]
		if !exists {
			return "", 0, ErrStoreItemUnknown
		}
		if rec.vendorID != req.VendorID {
			return "", 0, ErrStoreItemVendorMismatch
		}
		if !rec.isAvailable || rec.stockQuantity < qty {
			return "", 0, ErrStoreItemOutOfStock
		}

		// Compute subtotal and check integer overflow
		lineTotal := rec.priceMinor * int64(qty)
		if qty > 0 && lineTotal/int64(qty) != rec.priceMinor {
			return "", 0, errors.New("integer overflow during order calculation")
		}
		if math.MaxInt64-totalAmountMinor < lineTotal {
			return "", 0, errors.New("integer overflow during order sum")
		}
		totalAmountMinor += lineTotal

		validated = append(validated, validatedItem{
			invID:         rec.id,
			name:          rec.itemName,
			qty:           qty,
			unitPrice:     rec.priceMinor,
			subtotalMinor: lineTotal,
		})
	}

	// Insert master store order with minor units only
	queryOrder := `
		INSERT INTO store_orders (user_id, vendor_id, booking_id, status, total_amount_minor, delivery_address, delivery_slot) 
		VALUES ($1, $2, NULLIF($3, '')::uuid, 'PLACED', $4, $5, $6) 
		RETURNING id
	`
	var orderID string
	err = tx.QueryRow(queryOrder, req.UserID, req.VendorID, req.BookingID, totalAmountMinor, req.DeliveryAddress, req.DeliverySlot).Scan(&orderID)
	if err != nil {
		return "", 0, fmt.Errorf("failed to insert store order: %w", err)
	}

	// Insert line items in minor units only
	queryItem := `
		INSERT INTO store_order_items (order_id, inventory_item_id, item_name, quantity, unit_price_minor, subtotal_minor) 
		VALUES ($1, $2::uuid, $3, $4, $5, $6)
	`
	for _, it := range validated {
		if _, err := tx.Exec(queryItem, orderID, it.invID, it.name, it.qty, it.unitPrice, it.subtotalMinor); err != nil {
			return "", 0, fmt.Errorf("failed to insert store order item: %w", err)
		}
		// Rows are locked FOR UPDATE above, so the validated stock check still holds.
		if _, err := tx.Exec(`UPDATE vendor_inventory SET stock_quantity = stock_quantity - $2 WHERE id = $1`, it.invID, it.qty); err != nil {
			return "", 0, fmt.Errorf("failed to decrement stock: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return "", 0, fmt.Errorf("failed to commit store order transaction: %w", err)
	}

	return orderID, totalAmountMinor, nil
}

// UpdateStoreOrderStatus updates order progression (PLACED -> PACKED -> DISPATCHED -> DELIVERED).
func (s *Storage) UpdateStoreOrderStatus(orderID, status string) error {
	query := `UPDATE store_orders SET status = $2 WHERE id = $1`
	_, err := s.DB.Exec(query, orderID, status)
	return err
}
