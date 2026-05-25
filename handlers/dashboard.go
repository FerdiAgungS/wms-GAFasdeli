package handlers

import (
	"github.com/fasdeli/wms-GAFasdeli/config"
	"github.com/gofiber/fiber/v2"
)

// =====================================================
// INVENTORY DASHBOARD
// =====================================================
type InventoryDashboard struct {
	SKU      string
	Name     string
	Location string
	Qty      int
	UOM      string
}

// =====================================================
// ANALYTICS
// =====================================================
type Analytics struct {
	Name string
	Qty  int
}

// =====================================================
// DASHBOARD PAGE
// =====================================================
func DashboardPage(c *fiber.Ctx) error {

	// =====================================================
	// LOGIN INFO
	// =====================================================
	loginUser := c.Cookies("user")
	loginRole := c.Cookies("role")

	// =====================================================
	// INVENTORY
	// =====================================================
	rows, err := config.DB.Query(`
		SELECT
			p.sku,
			p.name,
			l.name,
			SUM(s.qty) as total_qty,
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
		HAVING SUM(s.qty) > 0
		ORDER BY total_qty DESC
		LIMIT 10
	`)

	if err != nil {
		return err
	}

	defer rows.Close()

	var products []InventoryDashboard

	var totalQty int

	for rows.Next() {

		var p InventoryDashboard

		rows.Scan(
			&p.SKU,
			&p.Name,
			&p.Location,
			&p.Qty,
			&p.UOM,
		)

		totalQty += p.Qty

		products = append(products, p)
	}

	// =====================================================
	// TOTAL SKU (FIXED)
	// =====================================================
	var totalSKU int

	config.DB.QueryRow(`
		SELECT COUNT(*)
		FROM products
	`).Scan(&totalSKU)

	// =====================================================
	// TOTAL LOCATION
	// =====================================================
	var totalLocation int

	config.DB.QueryRow(`
		SELECT COUNT(*)
		FROM locations
	`).Scan(&totalLocation)

	// =====================================================
	// INBOUND TODAY
	// =====================================================
	var inbound int

	config.DB.QueryRow(`
		SELECT IFNULL(SUM(qty),0)
		FROM inbound
		WHERE DATE(created_at)=CURDATE()
	`).Scan(&inbound)

	// =====================================================
	// OUTBOUND TODAY
	// =====================================================
	var outbound int

	config.DB.QueryRow(`
		SELECT IFNULL(SUM(qty),0)
		FROM outbounds
		WHERE DATE(created_at)=CURDATE()
	`).Scan(&outbound)

	// =====================================================
	// RETURN PENDING
	// =====================================================
	var returnPending int

	config.DB.QueryRow(`
		SELECT COUNT(*)
		FROM returns
		WHERE status='PENDING'
	`).Scan(&returnPending)

	// =====================================================
	// LOW STOCK COUNT
	// =====================================================
	var lowStockCount int

	config.DB.QueryRow(`
		SELECT COUNT(*)
		FROM (
			SELECT
				p.id,
				IFNULL(SUM(s.qty),0) total,
				p.min_stock
			FROM products p
			LEFT JOIN stocks s
				ON p.id = s.product_id
			GROUP BY p.id, p.min_stock
			HAVING total <= p.min_stock
		) x
	`).Scan(&lowStockCount)

	// =====================================================
	// TOP OUTBOUND
	// =====================================================
	var topOutbound []Analytics

	topRows, err := config.DB.Query(`
		SELECT
			p.name,
			IFNULL(SUM(o.qty),0) total
		FROM outbounds o
		JOIN products p
			ON o.product_id = p.id
		GROUP BY p.id, p.name
		ORDER BY total DESC
		LIMIT 5
	`)

	if err == nil {

		defer topRows.Close()

		for topRows.Next() {

			var a Analytics

			topRows.Scan(
				&a.Name,
				&a.Qty,
			)

			topOutbound = append(topOutbound, a)
		}
	}

	// =====================================================
	// SLOW MOVING
	// =====================================================
	var slowMoving []Analytics

	slowRows, err := config.DB.Query(`
		SELECT
			p.name,
			IFNULL(SUM(o.qty),0) total
		FROM products p
		LEFT JOIN outbounds o
			ON p.id = o.product_id
		GROUP BY p.id, p.name
		ORDER BY total ASC
		LIMIT 5
	`)

	if err == nil {

		defer slowRows.Close()

		for slowRows.Next() {

			var a Analytics

			slowRows.Scan(
				&a.Name,
				&a.Qty,
			)

			slowMoving = append(slowMoving, a)
		}
	}

	// =====================================================
	// LOW STOCK LIST
	// =====================================================
	var lowStock []Analytics

	lowRows, err := config.DB.Query(`
		SELECT
			p.name,
			IFNULL(SUM(s.qty),0) total
		FROM products p
		LEFT JOIN stocks s
			ON p.id = s.product_id
		GROUP BY
			p.id,
			p.name,
			p.min_stock
		HAVING total <= p.min_stock
		ORDER BY total ASC
		LIMIT 5
	`)

	if err == nil {

		defer lowRows.Close()

		for lowRows.Next() {

			var a Analytics

			lowRows.Scan(
				&a.Name,
				&a.Qty,
			)

			lowStock = append(lowStock, a)
		}
	}

	// =====================================================
	// TOP LOCATION STOCK
	// =====================================================
	var topLocation []Analytics

	locationRows, err := config.DB.Query(`
		SELECT
			l.name,
			IFNULL(SUM(s.qty),0) total
		FROM locations l
		LEFT JOIN stocks s
			ON l.id = s.location_id
		GROUP BY l.id, l.name
		ORDER BY total DESC
		LIMIT 5
	`)

	if err == nil {

		defer locationRows.Close()

		for locationRows.Next() {

			var a Analytics

			locationRows.Scan(
				&a.Name,
				&a.Qty,
			)

			topLocation = append(topLocation, a)
		}
	}

	// =====================================================
	// RENDER
	// =====================================================
	return c.Render("dashboard", fiber.Map{
		"Title": "Dashboard",

		"Products": products,

		"Qty":      totalQty,
		"TotalSKU": totalSKU,

		"Inbound":  inbound,
		"Outbound": outbound,

		"TopOutbound":   topOutbound,
		"SlowMoving":    slowMoving,
		"LowStock":      lowStock,
		"LowStockCount": lowStockCount,

		"ReturnPending": returnPending,
		"TotalLocation": totalLocation,
		"TopLocation":   topLocation,

		// LOGIN
		"LoginUser": loginUser,
		"LoginRole": loginRole,
	})
}
