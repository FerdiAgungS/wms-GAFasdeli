package handlers

import (
	"encoding/csv"
	"strconv"
	"strings"

	"github.com/fasdeli/wms-GAFasdeli/config"
	"github.com/gofiber/fiber/v2"
)

// =========================
// INVENTORY STRUCT
// =========================
type Inventory struct {
	ID         int
	ProductID  int
	LocationID int

	SKU      string
	Name     string
	Location string

	Qty    int
	UOM    string
	Status string
}

// =========================
// ADJUSTMENT STRUCT
// =========================
type Adjustment struct {
	ID       int
	Product  string
	Location string

	Qty    int
	Type   string
	Reason string
	Status string

	User       string
	ApprovedBy string
	CreatedAt  string
}

// =========================
// INVENTORY PAGE
// =========================
func InventoryPage(c *fiber.Ctx) error {

	// SEARCH
	searchSKU := strings.TrimSpace(
		c.Query("sku"),
	)

	searchLocation := strings.TrimSpace(
		c.Query("location"),
	)

	// QUERY
	query := `
		SELECT
			MIN(s.id),
			p.id,
			l.id,
			p.sku,
			p.name,
			l.name,
			SUM(s.qty) as total_qty,
			p.uom,
			p.min_stock
		FROM stocks s
		JOIN products p
			ON s.product_id = p.id
		JOIN locations l
			ON s.location_id = l.id
		WHERE 1=1
	`

	var args []interface{}

	// SEARCH SKU
	if searchSKU != "" {

		query += `
			AND p.sku LIKE ?
		`

		args = append(
			args,
			"%"+searchSKU+"%",
		)
	}

	// SEARCH LOCATION
	if searchLocation != "" {

		query += `
			AND l.name LIKE ?
		`

		args = append(
			args,
			"%"+searchLocation+"%",
		)
	}

	query += `
		GROUP BY
			p.id,
			l.id,
			p.sku,
			p.name,
			l.name,
			p.uom,
			p.min_stock
		ORDER BY p.name ASC
	`

	rows, err := config.DB.Query(
		query,
		args...,
	)

	if err != nil {
		return err
	}

	defer rows.Close()

	var inventories []Inventory

	for rows.Next() {

		var i Inventory
		var minStock int

		rows.Scan(
			&i.ID,
			&i.ProductID,
			&i.LocationID,
			&i.SKU,
			&i.Name,
			&i.Location,
			&i.Qty,
			&i.UOM,
			&minStock,
		)

		// STATUS
		if i.Qty <= minStock {

			i.Status = "LOW"

		} else {

			i.Status = "NORMAL"
		}

		inventories = append(
			inventories,
			i,
		)
	}

	// TOTAL PENDING
	var pending int

	config.DB.QueryRow(`
		SELECT COUNT(*)
		FROM adjustments
		WHERE status='PENDING'
	`).Scan(&pending)

	// LOGIN
	loginUser := c.Cookies("user")
	loginRole := c.Cookies("role")

	success := c.Query("success")

	return c.Render("inventory", fiber.Map{
		"Title":       "Inventory",
		"Inventories": inventories,

		"Pending": pending,

		"LoginUser": loginUser,
		"LoginRole": loginRole,

		"Success": success,

		"SearchSKU":      searchSKU,
		"SearchLocation": searchLocation,
	})
}

// =========================
// CREATE ADJUSTMENT
// =========================
func CreateAdjustment(c *fiber.Ctx) error {

	productID := c.FormValue("product_id")
	locationID := c.FormValue("location_id")
	qty := c.FormValue("qty")
	adjType := c.FormValue("type")
	reason := c.FormValue("reason")

	user := c.Cookies("user")

	// VALIDASI
	if productID == "" ||
		locationID == "" ||
		qty == "" ||
		adjType == "" {

		return c.Redirect("/inventory")
	}

	// INSERT
	_, err := config.DB.Exec(`
		INSERT INTO adjustments (
			product_id,
			location_id,
			qty,
			type,
			reason,
			requested_by,
			status
		)
		VALUES (?, ?, ?, ?, ?, ?, 'PENDING')
	`,
		productID,
		locationID,
		qty,
		adjType,
		reason,
		user,
	)

	if err != nil {
		return err
	}

	return c.Redirect("/inventory?success=1")
}

// =========================
// ADJUSTMENT PAGE
// =========================
func AdjustmentPage(c *fiber.Ctx) error {

	rows, err := config.DB.Query(`
		SELECT
			a.id,
			p.name,
			l.name,
			a.qty,
			a.type,
			a.reason,
			a.status,
			a.requested_by,
			IFNULL(a.approved_by,'-'),
			DATE_FORMAT(a.created_at,'%d-%m-%Y %H:%i')
		FROM adjustments a
		JOIN products p
			ON a.product_id = p.id
		JOIN locations l
			ON a.location_id = l.id
		ORDER BY a.created_at DESC
	`)

	if err != nil {
		return err
	}

	defer rows.Close()

	var adjustments []Adjustment

	for rows.Next() {

		var a Adjustment

		rows.Scan(
			&a.ID,
			&a.Product,
			&a.Location,
			&a.Qty,
			&a.Type,
			&a.Reason,
			&a.Status,
			&a.User,
			&a.ApprovedBy,
			&a.CreatedAt,
		)

		adjustments = append(
			adjustments,
			a,
		)
	}

	loginUser := c.Cookies("user")
	loginRole := c.Cookies("role")

	return c.Render("adjustment", fiber.Map{
		"Title":       "Adjustment Approval",
		"Adjustments": adjustments,

		"LoginUser": loginUser,
		"LoginRole": loginRole,
	})
}

// =========================
// APPROVE ADJUSTMENT
// =========================
func ApproveAdjustment(c *fiber.Ctx) error {

	id := c.Params("id")

	var productID int
	var locationID int
	var qty int
	var adjType string

	err := config.DB.QueryRow(`
		SELECT
			product_id,
			location_id,
			qty,
			type
		FROM adjustments
		WHERE id=?
	`,
		id,
	).Scan(
		&productID,
		&locationID,
		&qty,
		&adjType,
	)

	if err != nil {
		return err
	}

	// PLUS
	if adjType == "PLUS" {

		config.DB.Exec(`
			UPDATE stocks
			SET qty = qty + ?
			WHERE product_id=? AND location_id=?
		`,
			qty,
			productID,
			locationID,
		)

	} else {

		// MINUS
		config.DB.Exec(`
			UPDATE stocks
			SET qty = qty - ?
			WHERE product_id=? AND location_id=?
		`,
			qty,
			productID,
			locationID,
		)
	}

	manager := c.Cookies("user")

	// UPDATE STATUS
	config.DB.Exec(`
		UPDATE adjustments
		SET
			status='APPROVED',
			approved_by=?,
			approved_at=NOW()
		WHERE id=?
	`,
		manager,
		id,
	)

	return c.Redirect("/adjustment")
}

// =========================
// REJECT ADJUSTMENT
// =========================
func RejectAdjustment(c *fiber.Ctx) error {

	id := c.Params("id")

	manager := c.Cookies("user")

	config.DB.Exec(`
		UPDATE adjustments
		SET
			status='REJECTED',
			approved_by=?,
			approved_at=NOW()
		WHERE id=?
	`,
		manager,
		id,
	)

	return c.Redirect("/adjustment")
}

// =========================
// EXPORT INVENTORY CSV
// =========================
func ExportInventory(c *fiber.Ctx) error {

	rows, err := config.DB.Query(`
		SELECT
			p.sku,
			p.name,
			l.name,
			SUM(s.qty),
			p.uom
		FROM stocks s
		JOIN products p
			ON s.product_id = p.id
		JOIN locations l
			ON s.location_id = l.id
		GROUP BY
			p.id,
			l.id,
			p.sku,
			p.name,
			l.name,
			p.uom
		ORDER BY p.name ASC
	`)

	if err != nil {
		return err
	}

	defer rows.Close()

	c.Set("Content-Type", "text/csv")

	c.Set(
		"Content-Disposition",
		"attachment; filename=inventory.csv",
	)

	writer := csv.NewWriter(
		c.Response().BodyWriter(),
	)

	defer writer.Flush()

	writer.Write([]string{
		"SKU",
		"Product",
		"Location",
		"Qty",
		"UOM",
	})

	for rows.Next() {

		var sku string
		var name string
		var location string
		var qty int
		var uom string

		rows.Scan(
			&sku,
			&name,
			&location,
			&qty,
			&uom,
		)

		writer.Write([]string{
			sku,
			name,
			location,
			strconv.Itoa(qty),
			uom,
		})
	}

	return nil
}

// =========================
// EXPORT ADJUSTMENT HISTORY
// =========================
func ExportAdjustmentHistory(c *fiber.Ctx) error {

	rows, err := config.DB.Query(`
		SELECT
			p.name,
			l.name,
			a.qty,
			a.type,
			a.reason,
			a.status,
			a.requested_by,
			IFNULL(a.approved_by,'-'),
			DATE_FORMAT(a.created_at,'%d-%m-%Y %H:%i')
		FROM adjustments a
		JOIN products p
			ON a.product_id = p.id
		JOIN locations l
			ON a.location_id = l.id
		ORDER BY a.created_at DESC
	`)

	if err != nil {
		return err
	}

	defer rows.Close()

	c.Set("Content-Type", "text/csv")

	c.Set(
		"Content-Disposition",
		"attachment; filename=adjustment_history.csv",
	)

	writer := csv.NewWriter(
		c.Response().BodyWriter(),
	)

	defer writer.Flush()

	writer.Write([]string{
		"Product",
		"Location",
		"Qty",
		"Type",
		"Reason",
		"Status",
		"Requested By",
		"Approved By",
		"Created At",
	})

	for rows.Next() {

		var product string
		var location string
		var qty int
		var adjType string
		var reason string
		var status string
		var requestBy string
		var approvedBy string
		var createdAt string

		rows.Scan(
			&product,
			&location,
			&qty,
			&adjType,
			&reason,
			&status,
			&requestBy,
			&approvedBy,
			&createdAt,
		)

		writer.Write([]string{
			product,
			location,
			strconv.Itoa(qty),
			adjType,
			reason,
			status,
			requestBy,
			approvedBy,
			createdAt,
		})
	}

	return nil
}
