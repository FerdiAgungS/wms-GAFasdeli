package handlers

import (
	"encoding/csv"
	"strconv"

	"github.com/fasdeli/wms-GAFasdeli/config"
	"github.com/gofiber/fiber/v2"
)

// =====================================================
// INBOUND STRUCT
// =====================================================
type Inbound struct {
	ID          int
	SKU         string
	ProductName string
	Location    string
	Qty         int
	ReceivedBy  string
	CreatedAt   string
}

// =====================================================
// INBOUND PRODUCT STRUCT
// =====================================================
type InboundProduct struct {
	ID   int
	Name string
}

// =====================================================
// INBOUND LOCATION STRUCT
// =====================================================
type InboundLocation struct {
	ID   int
	Name string
}

// =====================================================
// INBOUND PAGE
// =====================================================
func InboundPage(c *fiber.Ctx) error {

	// =====================================================
	// INBOUND HISTORY
	// =====================================================
	rows, err := config.DB.Query(`
		SELECT
			i.id,
			p.sku,
			p.name,
			l.name,
			i.qty,
			i.received_by,
			DATE_FORMAT(i.created_at,'%d-%m-%Y %H:%i:%s')
		FROM inbound i
		JOIN products p
			ON i.product_id = p.id
		JOIN locations l
			ON i.location_id = l.id
		ORDER BY i.id DESC
	`)

	if err != nil {
		return err
	}

	defer rows.Close()

	var inbounds []Inbound

	for rows.Next() {

		var i Inbound

		rows.Scan(
			&i.ID,
			&i.SKU,
			&i.ProductName,
			&i.Location,
			&i.Qty,
			&i.ReceivedBy,
			&i.CreatedAt,
		)

		inbounds = append(inbounds, i)
	}

	// =====================================================
	// PRODUCTS
	// =====================================================
	productRows, err := config.DB.Query(`
		SELECT
			id,
			name
		FROM products
		ORDER BY name ASC
	`)

	if err != nil {
		return err
	}

	defer productRows.Close()

	var products []InboundProduct

	for productRows.Next() {

		var p InboundProduct

		productRows.Scan(
			&p.ID,
			&p.Name,
		)

		products = append(products, p)
	}

	// =====================================================
	// LOCATIONS
	// =====================================================
	locationRows, err := config.DB.Query(`
		SELECT
			id,
			name
		FROM locations
		ORDER BY name ASC
	`)

	if err != nil {
		return err
	}

	defer locationRows.Close()

	var locations []InboundLocation

	for locationRows.Next() {

		var l InboundLocation

		locationRows.Scan(
			&l.ID,
			&l.Name,
		)

		locations = append(locations, l)
	}

	// =====================================================
	// LOGIN USER
	// =====================================================
	loginUser := c.Cookies("user")
	loginRole := c.Cookies("role")

	// =====================================================
	// SUCCESS POPUP
	// =====================================================
	success := c.Query("success")

	// =====================================================
	// RENDER PAGE
	// =====================================================
	return c.Render("inbound", fiber.Map{
		"Title":     "Inbound",
		"Inbounds":  inbounds,
		"Products":  products,
		"Locations": locations,

		"LoginUser": loginUser,
		"LoginRole": loginRole,

		"Success": success,
	})
}

// =====================================================
// ADD INBOUND MULTI ITEM
// =====================================================
func AddInbound(c *fiber.Ctx) error {

	// =====================================================
	// ARRAY FORM
	// =====================================================
	productIDs := c.Context().PostArgs().PeekMulti("product_id[]")
	locationIDs := c.Context().PostArgs().PeekMulti("location_id[]")
	qtys := c.Context().PostArgs().PeekMulti("qty[]")

	// =====================================================
	// LOGIN USER
	// =====================================================
	receivedBy := c.Cookies("user")

	// =====================================================
	// LOOP ALL ITEM
	// =====================================================
	for i := 0; i < len(productIDs); i++ {

		productID := string(productIDs[i])
		locationID := string(locationIDs[i])
		qty := string(qtys[i])

		// =====================================================
		// VALIDASI
		// =====================================================
		if productID == "" ||
			locationID == "" ||
			qty == "" {

			continue
		}

		// =====================================================
		// INSERT INBOUND
		// =====================================================
		_, err := config.DB.Exec(`
			INSERT INTO inbound
			(
				product_id,
				location_id,
				qty,
				received_by
			)
			VALUES (?, ?, ?, ?)
		`,
			productID,
			locationID,
			qty,
			receivedBy,
		)

		if err != nil {
			return err
		}

		// =====================================================
		// CHECK STOCK
		// =====================================================
		var stockID int

		err = config.DB.QueryRow(`
			SELECT id
			FROM stocks
			WHERE product_id = ?
			AND location_id = ?
		`,
			productID,
			locationID,
		).Scan(&stockID)

		qtyInt, _ := strconv.Atoi(qty)

		// =====================================================
		// UPDATE STOCK
		// =====================================================
		if err == nil {

			_, err = config.DB.Exec(`
				UPDATE stocks
				SET qty = qty + ?
				WHERE id = ?
			`,
				qtyInt,
				stockID,
			)

		} else {

			// =====================================================
			// CREATE STOCK
			// =====================================================
			_, err = config.DB.Exec(`
				INSERT INTO stocks
				(
					product_id,
					location_id,
					qty
				)
				VALUES (?, ?, ?)
			`,
				productID,
				locationID,
				qtyInt,
			)
		}

		if err != nil {
			return err
		}
	}

	// =====================================================
	// SUCCESS REDIRECT
	// =====================================================
	return c.Redirect("/inbound?success=1")
}

// =====================================================
// EXPORT INBOUND HISTORY
// =====================================================
func ExportInboundHistory(c *fiber.Ctx) error {

	rows, err := config.DB.Query(`
		SELECT
			p.sku,
			p.name,
			l.name,
			i.qty,
			i.received_by,
			DATE_FORMAT(i.created_at,'%d-%m-%Y %H:%i:%s')
		FROM inbound i
		JOIN products p
			ON i.product_id = p.id
		JOIN locations l
			ON i.location_id = l.id
		ORDER BY i.id DESC
	`)

	if err != nil {
		return err
	}

	defer rows.Close()

	// =====================================================
	// HEADER DOWNLOAD
	// =====================================================
	c.Set("Content-Type", "text/csv")

	c.Set(
		"Content-Disposition",
		"attachment; filename=inbound_history.csv",
	)

	writer := csv.NewWriter(c.Response().BodyWriter())
	defer writer.Flush()

	// =====================================================
	// CSV HEADER
	// =====================================================
	writer.Write([]string{
		"SKU",
		"Product",
		"Location",
		"Qty",
		"Received By",
		"Created At",
	})

	// =====================================================
	// CSV DATA
	// =====================================================
	for rows.Next() {

		var sku string
		var product string
		var location string
		var qty int
		var receivedBy string
		var createdAt string

		rows.Scan(
			&sku,
			&product,
			&location,
			&qty,
			&receivedBy,
			&createdAt,
		)

		writer.Write([]string{
			sku,
			product,
			location,
			strconv.Itoa(qty),
			receivedBy,
			createdAt,
		})
	}

	return nil
}
