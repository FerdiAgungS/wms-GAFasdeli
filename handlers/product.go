package handlers

import (
	"encoding/csv"
	"os"
	"strconv"

	"github.com/fasdeli/wms-GAFasdeli/config"
	"github.com/gofiber/fiber/v2"
)

type Product struct {
	ID       int
	SKU      string
	Name     string
	UOM      string
	MinStock int
	MaxStock int
	Qty      int
	Status   string
}

func ProductPage(c *fiber.Ctx) error {

	rows, err := config.DB.Query(`
		SELECT
			p.id,
			p.sku,
			p.name,
			p.uom,
			p.min_stock,
			p.max_stock,
			IFNULL(SUM(s.qty),0) as total_qty
		FROM products p
		LEFT JOIN stocks s
			ON p.id = s.product_id
		GROUP BY p.id
		ORDER BY p.id DESC
	`)

	if err != nil {
		return err
	}

	defer rows.Close()

	var products []Product

	for rows.Next() {

		var p Product

		rows.Scan(
			&p.ID,
			&p.SKU,
			&p.Name,
			&p.UOM,
			&p.MinStock,
			&p.MaxStock,
			&p.Qty,
		)

		if p.Qty <= p.MinStock {

			p.Status = "LOW"

		} else {

			p.Status = "NORMAL"
		}

		products = append(products, p)
	}

	return c.Render("products", fiber.Map{
		"Title":    "Products",
		"Products": products,
	})
}

// =========================
// ADD PRODUCT
// =========================
func AddProduct(c *fiber.Ctx) error {

	sku := c.FormValue("sku")
	name := c.FormValue("name")
	uom := c.FormValue("uom")

	minStock := c.FormValue("min_stock")
	maxStock := c.FormValue("max_stock")

	_, err := config.DB.Exec(`
		INSERT INTO products
		(
			sku,
			name,
			uom,
			min_stock,
			max_stock
		)
		VALUES (?, ?, ?, ?, ?)
	`,
		sku,
		name,
		uom,
		minStock,
		maxStock,
	)

	if err != nil {
		return err
	}

	return c.Redirect("/products")
}

// =========================
// DELETE PRODUCT
// =========================
func DeleteProduct(c *fiber.Ctx) error {

	id := c.Params("id")

	// delete stock dulu
	config.DB.Exec(`
		DELETE FROM stocks
		WHERE product_id = ?
	`, id)

	// delete product
	_, err := config.DB.Exec(`
		DELETE FROM products
		WHERE id = ?
	`, id)

	if err != nil {
		return err
	}

	return c.Redirect("/products")
}

// =========================
// UPLOAD CSV
// =========================
func UploadProduct(c *fiber.Ctx) error {

	file, err := c.FormFile("file")

	if err != nil {
		return err
	}

	tempFile := "./temp_products.csv"

	if err := c.SaveFile(file, tempFile); err != nil {
		return err
	}

	f, err := os.Open(tempFile)

	if err != nil {
		return err
	}

	defer f.Close()

	reader := csv.NewReader(f)

	rows, err := reader.ReadAll()

	if err != nil {
		return err
	}

	// skip header
	for i, row := range rows {

		if i == 0 {
			continue
		}

		if len(row) < 5 {
			continue
		}

		sku := row[0]
		name := row[1]
		uom := row[2]

		minStock, _ := strconv.Atoi(row[3])
		maxStock, _ := strconv.Atoi(row[4])

		config.DB.Exec(`
			INSERT INTO products
			(
				sku,
				name,
				uom,
				min_stock,
				max_stock
			)
			VALUES (?, ?, ?, ?, ?)
		`,
			sku,
			name,
			uom,
			minStock,
			maxStock,
		)
	}

	os.Remove(tempFile)

	return c.Redirect("/products")
}

// =========================
// DOWNLOAD TEMPLATE CSV
// =========================
func DownloadTemplate(c *fiber.Ctx) error {

	c.Set("Content-Type", "text/csv")
	c.Set(
		"Content-Disposition",
		"attachment; filename=product_template.csv",
	)

	c.WriteString("SKU,Product Name,UOM,Min Stock,Max Stock\n")
	c.WriteString("SKU001,Indomie Box,PCS,10,100\n")
	c.WriteString("SKU002,Aqua 600ML,PCS,20,200\n")

	return nil
}
