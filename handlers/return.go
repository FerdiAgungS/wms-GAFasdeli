package handlers

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/fasdeli/wms-GAFasdeli/config"
	"github.com/gofiber/fiber/v2"
)

// =====================================================
// RETURN PAGE
// =====================================================
func ReturnPage(c *fiber.Ctx) error {

	// =====================================================
	// PRODUCTS
	// =====================================================
	rows, err := config.DB.Query(`
		SELECT
			id,
			sku,
			name,
			uom
		FROM products
		ORDER BY name ASC
	`)

	if err != nil {
		return err
	}

	defer rows.Close()

	type Product struct {
		ID   int
		SKU  string
		Name string
		UOM  string
	}

	var products []Product

	for rows.Next() {

		var p Product

		rows.Scan(
			&p.ID,
			&p.SKU,
			&p.Name,
			&p.UOM,
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

	type Location struct {
		ID   int
		Name string
	}

	var locations []Location

	for locationRows.Next() {

		var l Location

		locationRows.Scan(
			&l.ID,
			&l.Name,
		)

		locations = append(locations, l)
	}

	// =====================================================
	// LOGIN
	// =====================================================
	loginUser := c.Cookies("user")
	loginRole := c.Cookies("role")

	// =====================================================
	// RENDER
	// =====================================================
	return c.Render("return", fiber.Map{
		"Title":     "Return",
		"Products":  products,
		"Locations": locations,

		"Msg": c.Query("msg"),

		"LoginUser": loginUser,
		"LoginRole": loginRole,
	})
}

// =====================================================
// CREATE RETURN
// =====================================================
func CreateReturn(c *fiber.Ctx) error {

	productID := c.FormValue("product_id")
	locationID := c.FormValue("location_id")
	qtyStr := c.FormValue("qty")
	reason := c.FormValue("reason")

	qty, _ := strconv.Atoi(qtyStr)

	// =====================================================
	// VALIDASI
	// =====================================================
	if qty <= 0 {

		return c.Redirect("/return?msg=error")
	}

	// =====================================================
	// PHOTO
	// =====================================================
	file, err := c.FormFile("photo")

	if err != nil {

		return c.Redirect("/return?msg=nophoto")
	}

	os.MkdirAll("./uploads/returns", os.ModePerm)

	filename := strconv.FormatInt(
		time.Now().UnixNano(),
		10,
	) + filepath.Ext(file.Filename)

	savePath := "./uploads/returns/" + filename

	if err := c.SaveFile(file, savePath); err != nil {

		return c.Redirect("/return?msg=error")
	}

	// =====================================================
	// LOGIN USER
	// =====================================================
	requestBy := c.Cookies("user")

	// =====================================================
	// INSERT RETURN
	// =====================================================
	_, err = config.DB.Exec(`
		INSERT INTO returns
		(
			product_id,
			location_id,
			qty,
			reason,
			photo,
			requested_by,
			status
		)
		VALUES (?, ?, ?, ?, ?, ?, 'PENDING')
	`,
		productID,
		locationID,
		qty,
		reason,
		filename,
		requestBy,
	)

	if err != nil {

		return err
	}

	// =====================================================
	// SUCCESS
	// =====================================================
	return c.Redirect("/return?msg=success")
}

// =====================================================
// RETURN APPROVAL PAGE
// =====================================================
func ReturnApprovalPage(c *fiber.Ctx) error {

	rows, err := config.DB.Query(`
		SELECT
			r.id,
			p.sku,
			p.name,
			p.uom,
			r.qty,
			l.name,
			r.reason,
			r.photo,
			r.requested_by,
			r.status,
			IFNULL(r.approved_by,'-'),
			DATE_FORMAT(r.created_at,'%d-%m-%Y %H:%i:%s')
		FROM returns r
		JOIN products p
			ON r.product_id = p.id
		JOIN locations l
			ON r.location_id = l.id
		ORDER BY r.created_at DESC
	`)

	if err != nil {

		return err
	}

	defer rows.Close()

	// =====================================================
	// STRUCT
	// =====================================================
	type Data struct {
		ID         int
		SKU        string
		Name       string
		UOM        string
		Qty        int
		Location   string
		Reason     string
		Photo      string
		RequestBy  string
		Status     string
		ApprovedBy string
		Date       string
	}

	var data []Data

	for rows.Next() {

		var d Data

		rows.Scan(
			&d.ID,
			&d.SKU,
			&d.Name,
			&d.UOM,
			&d.Qty,
			&d.Location,
			&d.Reason,
			&d.Photo,
			&d.RequestBy,
			&d.Status,
			&d.ApprovedBy,
			&d.Date,
		)

		data = append(data, d)
	}

	// =====================================================
	// LOGIN
	// =====================================================
	loginUser := c.Cookies("user")
	loginRole := c.Cookies("role")

	// =====================================================
	// MESSAGE
	// =====================================================
	msg := c.Query("msg")

	// =====================================================
	// RENDER
	// =====================================================
	return c.Render("return_approval", fiber.Map{
		"Title": "Return Approval",
		"Data":  data,
		"Msg":   msg,

		"LoginUser": loginUser,
		"LoginRole": loginRole,
	})
}

// =====================================================
// APPROVE RETURN
// =====================================================
func ApproveReturn(c *fiber.Ctx) error {

	id := c.Params("id")

	var productID int
	var locationID int
	var qty int
	var status string

	err := config.DB.QueryRow(`
		SELECT
			product_id,
			location_id,
			qty,
			status
		FROM returns
		WHERE id = ?
	`,
		id,
	).Scan(
		&productID,
		&locationID,
		&qty,
		&status,
	)

	if err != nil {

		return err
	}

	// =====================================================
	// CEGAH DOUBLE APPROVE
	// =====================================================
	if status != "PENDING" {

		return c.Redirect("/return/approval")
	}

	// =====================================================
	// UPDATE STOCK
	// =====================================================
	config.DB.Exec(`
		INSERT INTO stocks
		(product_id, location_id, qty)
		VALUES (?, ?, ?)
		ON DUPLICATE KEY UPDATE
		qty = qty + VALUES(qty)
	`,
		productID,
		locationID,
		qty,
	)

	// =====================================================
	// LOGIN USER
	// =====================================================
	manager := c.Cookies("user")

	// =====================================================
	// APPROVE
	// =====================================================
	config.DB.Exec(`
		UPDATE returns
		SET
			status='APPROVED',
			approved_by=?,
			approved_at=NOW()
		WHERE id=?
	`,
		manager,
		id,
	)

	// =====================================================
	// SUCCESS
	// =====================================================
	return c.Redirect("/return/approval?msg=approved")
}

// =====================================================
// REJECT RETURN
// =====================================================
func RejectReturn(c *fiber.Ctx) error {

	id := c.Params("id")

	var status string

	config.DB.QueryRow(`
		SELECT status
		FROM returns
		WHERE id=?
	`,
		id,
	).Scan(&status)

	// =====================================================
	// CEGAH DOUBLE REJECT
	// =====================================================
	if status != "PENDING" {

		return c.Redirect("/return/approval")
	}

	// =====================================================
	// LOGIN USER
	// =====================================================
	manager := c.Cookies("user")

	// =====================================================
	// REJECT
	// =====================================================
	config.DB.Exec(`
		UPDATE returns
		SET
			status='REJECTED',
			approved_by=?,
			approved_at=NOW()
		WHERE id=?
	`,
		manager,
		id,
	)

	// =====================================================
	// SUCCESS
	// =====================================================
	return c.Redirect("/return/approval?msg=rejected")
}

// =====================================================
// RETURN HISTORY
// =====================================================
func ReturnHistory(c *fiber.Ctx) error {

	rows, err := config.DB.Query(`
		SELECT
			p.sku,
			p.name,
			p.uom,
			r.qty,
			l.name,
			r.reason,
			r.requested_by,
			r.status,
			IFNULL(r.approved_by,'-'),
			DATE_FORMAT(r.created_at,'%d-%m-%Y %H:%i:%s')
		FROM returns r
		JOIN products p
			ON r.product_id = p.id
		JOIN locations l
			ON r.location_id = l.id
		ORDER BY r.created_at DESC
	`)

	if err != nil {

		return err
	}

	defer rows.Close()

	// =====================================================
	// STRUCT
	// =====================================================
	type Log struct {
		SKU        string
		Name       string
		UOM        string
		Qty        int
		Location   string
		Reason     string
		User       string
		Status     string
		ApprovedBy string
		Date       string
	}

	var logs []Log

	for rows.Next() {

		var l Log

		rows.Scan(
			&l.SKU,
			&l.Name,
			&l.UOM,
			&l.Qty,
			&l.Location,
			&l.Reason,
			&l.User,
			&l.Status,
			&l.ApprovedBy,
			&l.Date,
		)

		logs = append(logs, l)
	}

	// =====================================================
	// LOGIN
	// =====================================================
	loginUser := c.Cookies("user")
	loginRole := c.Cookies("role")

	// =====================================================
	// RENDER
	// =====================================================
	return c.Render("return_history", fiber.Map{
		"Title": "Return History",
		"Logs":  logs,

		"LoginUser": loginUser,
		"LoginRole": loginRole,
	})
}
