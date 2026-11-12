package db

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/purohit-platform/backend/internal/models"
)

// GetPoojaSamagriTx returns the itemized materials required for a given pooja within a transaction
func (s *Storage) GetPoojaSamagriTx(ctx context.Context, tx *sql.Tx, poojaID string) ([]models.PoojaSamagriItem, error) {
	if poojaID == "" { // custom-service bookings have no pooja catalog samagri
		return nil, nil
	}
	query := `
		SELECT id, pooja_id, item_name, category, quantity, is_mandatory, estimated_price, COALESCE(estimated_price_minor, ROUND(estimated_price * 100))
		FROM pooja_samagri_items 
		WHERE pooja_id = $1 
		ORDER BY category, item_name
	`
	rows, err := tx.QueryContext(ctx, query, poojaID)
	if err != nil {
		return nil, fmt.Errorf("failed to query samagri items: %w", err)
	}
	defer rows.Close()

	var items []models.PoojaSamagriItem
	for rows.Next() {
		var item models.PoojaSamagriItem
		if err := rows.Scan(
			&item.ID, &item.PoojaID, &item.ItemName, &item.Category,
			&item.Quantity, &item.IsMandatory, &item.EstimatedPrice, &item.EstimatedPriceMinor,
		); err != nil {
			return nil, err
		}
		item.Currency = "INR"
		items = append(items, item)
	}

	if len(items) == 0 {
		items = []models.PoojaSamagriItem{
			{PoojaID: poojaID, ItemName: "Brass / Copper Puja Plates (Thalis)", Category: "Devotee Household Items", Quantity: "3 nos", IsMandatory: true, EstimatedPrice: 0.0, EstimatedPriceMinor: 0, Currency: "INR"},
			{PoojaID: poojaID, ItemName: "Brass Panchapatra & Udharani", Category: "Devotee Household Items", Quantity: "1 set", IsMandatory: true, EstimatedPrice: 0.0, EstimatedPriceMinor: 0, Currency: "INR"},
			{PoojaID: poojaID, ItemName: "Brass Deepam / Diyas", Category: "Devotee Household Items", Quantity: "2 nos", IsMandatory: true, EstimatedPrice: 0.0, EstimatedPriceMinor: 0, Currency: "INR"},
			{PoojaID: poojaID, ItemName: "Pure Cow Ghee & Matchbox", Category: "Devotee Household Items", Quantity: "500g", IsMandatory: true, EstimatedPrice: 350.0, EstimatedPriceMinor: 35000, Currency: "INR"},
			{PoojaID: poojaID, ItemName: "Sitting Mats (Asanam)", Category: "Devotee Household Items", Quantity: "3 nos", IsMandatory: true, EstimatedPrice: 0.0, EstimatedPriceMinor: 0, Currency: "INR"},
			{PoojaID: poojaID, ItemName: "Haldi (Pasupu) & Kumkuma", Category: "Sacred Ritual Samagri", Quantity: "100g each", IsMandatory: true, EstimatedPrice: 80.0, EstimatedPriceMinor: 8000, Currency: "INR"},
			{PoojaID: poojaID, ItemName: "Akshatulu & Gandham Paste", Category: "Sacred Ritual Samagri", Quantity: "1 box", IsMandatory: true, EstimatedPrice: 60.0, EstimatedPriceMinor: 6000, Currency: "INR"},
			{PoojaID: poojaID, ItemName: "Dhoop Cones & Karpooram (Camphor)", Category: "Sacred Ritual Samagri", Quantity: "1 pack", IsMandatory: true, EstimatedPrice: 120.0, EstimatedPriceMinor: 12000, Currency: "INR"},
			{PoojaID: poojaID, ItemName: "Navadhanya Kit & Sacred Janeu", Category: "Sacred Ritual Samagri", Quantity: "1 kit", IsMandatory: true, EstimatedPrice: 150.0, EstimatedPriceMinor: 15000, Currency: "INR"},
		}
	}

	return items, nil
}

// GetPoojaSamagri returns the itemized materials required for a given pooja
func (s *Storage) GetPoojaSamagri(poojaID string) ([]models.PoojaSamagriItem, error) {
	query := `
		SELECT id, pooja_id, item_name, category, quantity, is_mandatory, estimated_price, COALESCE(estimated_price_minor, ROUND(estimated_price * 100))
		FROM pooja_samagri_items 
		WHERE pooja_id = $1 
		ORDER BY category, item_name
	`
	rows, err := s.DB.Query(query, poojaID)
	if err != nil {
		return nil, fmt.Errorf("failed to query samagri items: %w", err)
	}
	defer rows.Close()

	var items []models.PoojaSamagriItem
	for rows.Next() {
		var item models.PoojaSamagriItem
		if err := rows.Scan(
			&item.ID, &item.PoojaID, &item.ItemName, &item.Category,
			&item.Quantity, &item.IsMandatory, &item.EstimatedPrice, &item.EstimatedPriceMinor,
		); err != nil {
			continue
		}
		item.Currency = "INR"
		items = append(items, item)
	}

	// Fallback seed items if table is empty for this pooja
	if len(items) == 0 {
		items = []models.PoojaSamagriItem{
			{PoojaID: poojaID, ItemName: "Brass / Copper Puja Plates (Thalis)", Category: "Devotee Household Items", Quantity: "3 nos", IsMandatory: true, EstimatedPrice: 0.0, EstimatedPriceMinor: 0, Currency: "INR"},
			{PoojaID: poojaID, ItemName: "Brass Panchapatra & Udharani", Category: "Devotee Household Items", Quantity: "1 set", IsMandatory: true, EstimatedPrice: 0.0, EstimatedPriceMinor: 0, Currency: "INR"},
			{PoojaID: poojaID, ItemName: "Brass Deepam / Diyas", Category: "Devotee Household Items", Quantity: "2 nos", IsMandatory: true, EstimatedPrice: 0.0, EstimatedPriceMinor: 0, Currency: "INR"},
			{PoojaID: poojaID, ItemName: "Pure Cow Ghee & Matchbox", Category: "Devotee Household Items", Quantity: "500g", IsMandatory: true, EstimatedPrice: 350.0, EstimatedPriceMinor: 35000, Currency: "INR"},
			{PoojaID: poojaID, ItemName: "Sitting Mats (Asanam)", Category: "Devotee Household Items", Quantity: "3 nos", IsMandatory: true, EstimatedPrice: 0.0, EstimatedPriceMinor: 0, Currency: "INR"},
			{PoojaID: poojaID, ItemName: "Haldi (Pasupu) & Kumkuma", Category: "Sacred Ritual Samagri", Quantity: "100g each", IsMandatory: true, EstimatedPrice: 80.0, EstimatedPriceMinor: 8000, Currency: "INR"},
			{PoojaID: poojaID, ItemName: "Akshatulu & Gandham Paste", Category: "Sacred Ritual Samagri", Quantity: "1 box", IsMandatory: true, EstimatedPrice: 60.0, EstimatedPriceMinor: 6000, Currency: "INR"},
			{PoojaID: poojaID, ItemName: "Dhoop Cones & Karpooram (Camphor)", Category: "Sacred Ritual Samagri", Quantity: "1 pack", IsMandatory: true, EstimatedPrice: 120.0, EstimatedPriceMinor: 12000, Currency: "INR"},
			{PoojaID: poojaID, ItemName: "Navadhanya Kit & Sacred Janeu", Category: "Sacred Ritual Samagri", Quantity: "1 kit", IsMandatory: true, EstimatedPrice: 150.0, EstimatedPriceMinor: 15000, Currency: "INR"},
		}
	}

	return items, nil
}

// AddPoojaSamagriItem adds a new standard item to the pooja checklist (CMS)
func (s *Storage) AddPoojaSamagriItem(item models.PoojaSamagriItem) (string, error) {
	query := `
		INSERT INTO pooja_samagri_items (pooja_id, item_name, category, quantity, is_mandatory, estimated_price)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id;
	`
	var id string
	err := s.DB.QueryRow(
		query, item.PoojaID, item.ItemName, item.Category,
		item.Quantity, item.IsMandatory, item.EstimatedPrice,
	).Scan(&id)
	return id, err
}

// DeletePoojaSamagriItem deletes an item from the standard checklist (CMS)
func (s *Storage) DeletePoojaSamagriItem(itemID string) error {
	query := `DELETE FROM pooja_samagri_items WHERE id = $1;`
	_, err := s.DB.Exec(query, itemID)
	return err
}

// GetBookingMaterials retrieves the active materials checklist for a specific booking
func (s *Storage) GetBookingMaterials(bookingID string) ([]models.BookingMaterial, error) {
	query := `
		SELECT id, booking_id, item_name, category, quantity, is_included_by_priest, is_fulfilled_by_vendor, COALESCE(vendor_order_id::text, '')
		FROM booking_materials
		WHERE booking_id = $1
		ORDER BY category, item_name;
	`
	rows, err := s.DB.Query(query, bookingID)
	if err != nil {
		return nil, fmt.Errorf("failed to query booking materials: %w", err)
	}
	defer rows.Close()

	var materials []models.BookingMaterial
	for rows.Next() {
		var m models.BookingMaterial
		if err := rows.Scan(
			&m.ID, &m.BookingID, &m.ItemName, &m.Category,
			&m.Quantity, &m.IsIncludedByPriest, &m.IsFulfilledByVendor, &m.VendorOrderID,
		); err != nil {
			continue
		}
		materials = append(materials, m)
	}
	return materials, nil
}

// UpdateBookingMaterials updates priest bringing status or checklist items for a booking
func (s *Storage) UpdateBookingMaterials(bookingID string, materials []models.BookingMaterial) error {
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, m := range materials {
		if m.ID != "" {
			_, err := tx.Exec(`
				UPDATE booking_materials 
				SET is_included_by_priest = $1, quantity = $2
				WHERE id = $3 AND booking_id = $4;
			`, m.IsIncludedByPriest, m.Quantity, m.ID, bookingID)
			if err != nil {
				return err
			}
		} else {
			_, err := tx.Exec(`
				INSERT INTO booking_materials (booking_id, item_name, category, quantity, is_included_by_priest)
				VALUES ($1, $2, $3, $4, $5);
			`, bookingID, m.ItemName, m.Category, m.Quantity, m.IsIncludedByPriest)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
