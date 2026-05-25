package handlers

import (
	"fmt"
	"strconv"
	"time"

	"github.com/fasdeli/wms-GAFasdeli/config"
	"github.com/gofiber/fiber/v2"
	"github.com/xuri/excelize/v2"
)

// =====================================================
// GENERATE SJ
// =====================================================
func GenerateSJ() string {

	var count int

	config.DB.QueryRow(`
		SELECT COUNT(*)
		FROM outbounds
		WHERE DATE(created_at)=CURDATE()
	`).Scan(&count)

	return fmt.Sprintf(
		"SJ-%s-%03d",
		time.Now().Format("20060102"),
		count+1,
	)
}

// =====================================================
// OUTBOUND PAGE
// =====================================================
func OutboundPage(c *fiber.Ctx) error {

	rows, err := config.DB.Query(`
		SELECT
			p.id,
			p.sku,
			p.name,
			p.uom,
			l.id,
			l.name,
			s.qty
		FROM stocks s
		JOIN products p
			ON s.product_id = p.id
		JOIN locations l
			ON s.location_id = l.id
		WHERE s.qty > 0
		ORDER BY p.name ASC
	`)

	if err != nil {
		return err
	}

	defer rows.Close()

	type Product struct {
		ID         int
		SKU        string
		Name       string
		UOM        string
		LocationID int
		Location   string
		Qty        int
	}

	var products []Product

	for rows.Next() {

		var p Product

		rows.Scan(
			&p.ID,
			&p.SKU,
			&p.Name,
			&p.UOM,
			&p.LocationID,
			&p.Location,
			&p.Qty,
		)

		products = append(products, p)
	}

	// =====================================================
	// LOGIN USER
	// =====================================================
	loginUser := c.Cookies("user")
	loginRole := c.Cookies("role")

	return c.Render("outbound", fiber.Map{
		"Title":     "Outbound",
		"Products":  products,
		"Success":   c.Query("success"),
		"SJ":        c.Query("sj"),
		"LoginUser": loginUser,
		"LoginRole": loginRole,
	})
}

// =====================================================
// CREATE OUTBOUND
// =====================================================
func CreateOutbound(c *fiber.Ctx) error {

	// =====================================================
	// AUTO LOGIN USER
	// =====================================================
	user := c.Cookies("user")

	if user == "" {

		user = "UNKNOWN"
	}

	sj := GenerateSJ()

	productIDs := c.Context().PostArgs().PeekMulti("product_id")
	locationIDs := c.Context().PostArgs().PeekMulti("location_id")
	qtys := c.Context().PostArgs().PeekMulti("qty")

	for i := range productIDs {

		productID := string(productIDs[i])
		locationID := string(locationIDs[i])
		qtyStr := string(qtys[i])

		qty, _ := strconv.Atoi(qtyStr)

		if qty <= 0 {
			continue
		}

		// =====================================================
		// CHECK STOCK
		// =====================================================
		var stock int

		err := config.DB.QueryRow(`
			SELECT qty
			FROM stocks
			WHERE product_id = ?
			AND location_id = ?
		`,
			productID,
			locationID,
		).Scan(&stock)

		if err != nil {
			continue
		}

		// =====================================================
		// VALIDASI STOCK
		// =====================================================
		if qty > stock {
			continue
		}

		// =====================================================
		// UPDATE STOCK
		// =====================================================
		config.DB.Exec(`
			UPDATE stocks
			SET qty = qty - ?
			WHERE product_id = ?
			AND location_id = ?
		`,
			qty,
			productID,
			locationID,
		)

		// =====================================================
		// INSERT OUTBOUND
		// =====================================================
		config.DB.Exec(`
			INSERT INTO outbounds
			(
				product_id,
				location_id,
				qty,
				user,
				sj_number
			)
			VALUES (?, ?, ?, ?, ?)
		`,
			productID,
			locationID,
			qty,
			user,
			sj,
		)
	}

	return c.Redirect("/outbound?success=1&sj=" + sj)
}

// =====================================================
// DOWNLOAD SURAT JALAN
// =====================================================
func DownloadSJ(c *fiber.Ctx) error {

	sj := c.Query("sj")

	if sj == "" {

		return c.SendString("SJ kosong")
	}

	rows, err := config.DB.Query(`
		SELECT
			o.sj_number,
			p.sku,
			p.name,
			p.uom,
			o.qty,
			l.name
		FROM outbounds o
		JOIN products p
			ON o.product_id = p.id
		JOIN locations l
			ON o.location_id = l.id
		WHERE o.sj_number = ?
	`,
		sj,
	)

	if err != nil {

		return err
	}

	defer rows.Close()

	type Row struct {
		SJ       string
		SKU      string
		Name     string
		UOM      string
		Qty      int
		Location string
	}

	var data []Row

	for rows.Next() {

		var r Row

		rows.Scan(
			&r.SJ,
			&r.SKU,
			&r.Name,
			&r.UOM,
			&r.Qty,
			&r.Location,
		)

		data = append(data, r)
	}

	if len(data) == 0 {

		return c.SendString("Data SJ tidak ditemukan")
	}

	// =====================================================
	// CREATE EXCEL
	// =====================================================
	file := excelize.NewFile()

	sheet := "SURAT JALAN"

	file.SetSheetName("Sheet1", sheet)

	// TITLE
	file.SetCellValue(sheet, "A1", "SURAT JALAN")
	file.MergeCell(sheet, "A1", "F1")

	// INFO
	file.SetCellValue(sheet, "A3", "Nomor SJ")
	file.SetCellValue(sheet, "B3", sj)

	file.SetCellValue(sheet, "A4", "Tanggal")
	file.SetCellValue(sheet, "B4", time.Now().Format("02 Jan 2006"))

	// HEADER
	headers := []string{
		"No",
		"SKU",
		"Product Name",
		"Qty",
		"UOM",
		"Location",
	}

	startRow := 6

	for i, h := range headers {

		cell := string(rune('A'+i)) + strconv.Itoa(startRow)

		file.SetCellValue(sheet, cell, h)
	}

	// DATA
	for i, d := range data {

		row := startRow + 1 + i

		file.SetCellValue(sheet, "A"+strconv.Itoa(row), i+1)
		file.SetCellValue(sheet, "B"+strconv.Itoa(row), d.SKU)
		file.SetCellValue(sheet, "C"+strconv.Itoa(row), d.Name)
		file.SetCellValue(sheet, "D"+strconv.Itoa(row), d.Qty)
		file.SetCellValue(sheet, "E"+strconv.Itoa(row), d.UOM)
		file.SetCellValue(sheet, "F"+strconv.Itoa(row), d.Location)
	}

	endRow := startRow + len(data)

	// STYLE
	headerStyle, _ := file.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
		},
		Alignment: &excelize.Alignment{
			Horizontal: "center",
		},
	})

	file.SetCellStyle(
		sheet,
		"A6",
		"F6",
		headerStyle,
	)

	// WIDTH
	file.SetColWidth(sheet, "A", "F", 20)

	// SIGNATURE
	signRow := endRow + 4

	file.SetCellValue(sheet, "B"+strconv.Itoa(signRow), "Dikeluarkan Oleh")
	file.SetCellValue(sheet, "E"+strconv.Itoa(signRow), "Diterima Oleh")

	// =====================================================
	// DOWNLOAD
	// =====================================================
	c.Set(
		"Content-Type",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	)

	c.Set(
		"Content-Disposition",
		"attachment; filename=SJ_"+sj+".xlsx",
	)

	return file.Write(c.Response().BodyWriter())
}

// =====================================================
// OUTBOUND HISTORY
// =====================================================
func OutboundHistory(c *fiber.Ctx) error {

	rows, err := config.DB.Query(`
		SELECT
			o.sj_number,
			p.sku,
			p.name,
			p.uom,
			o.qty,
			l.name,
			o.user,
			DATE_FORMAT(o.created_at,'%d-%m-%Y %H:%i:%s')
		FROM outbounds o
		JOIN products p
			ON o.product_id = p.id
		JOIN locations l
			ON o.location_id = l.id
		ORDER BY o.created_at DESC
	`)

	if err != nil {

		return err
	}

	defer rows.Close()

	type Data struct {
		SJ       string
		SKU      string
		Name     string
		UOM      string
		Qty      int
		Location string
		User     string
		Date     string
	}

	var list []Data

	for rows.Next() {

		var d Data

		rows.Scan(
			&d.SJ,
			&d.SKU,
			&d.Name,
			&d.UOM,
			&d.Qty,
			&d.Location,
			&d.User,
			&d.Date,
		)

		list = append(list, d)
	}

	// =====================================================
	// LOGIN USER
	// =====================================================
	loginUser := c.Cookies("user")
	loginRole := c.Cookies("role")

	return c.Render("outbound_history", fiber.Map{
		"Title":     "Outbound History",
		"Data":      list,
		"LoginUser": loginUser,
		"LoginRole": loginRole,
	})
}
