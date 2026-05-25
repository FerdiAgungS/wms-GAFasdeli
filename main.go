package main

import (
	"log"
	"os" // Ditambahkan untuk membaca PORT dari Render

	"github.com/fasdeli/wms-GAFasdeli/config"
	"github.com/fasdeli/wms-GAFasdeli/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/template/html/v2"
)

func main() {

	// =====================================================
	// CONNECT DATABASE
	// =====================================================

	config.ConnectDB()

	// =====================================================
	// HTML ENGINE
	// =====================================================

	engine := html.New("./templates", ".html")

	// =====================================================
	// FIBER APP
	// =====================================================

	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// =====================================================
	// STATIC FILES
	// =====================================================

	app.Static("/static", "./static")

	// UPLOAD IMAGE
	app.Static("/uploads", "./uploads")

	// =====================================================
	// ROOT
	// =====================================================

	app.Get("/", func(c *fiber.Ctx) error {

		user := c.Cookies("user")

		if user == "" {
			return c.Redirect("/login")
		}

		return handlers.DashboardPage(c)
	})

	// =====================================================
	// LOGIN
	// =====================================================

	app.Get("/login", func(c *fiber.Ctx) error {

		if c.Cookies("user") != "" {
			return c.Redirect("/")
		}

		return c.Render("login", fiber.Map{
			"Title": "Login",
		})
	})

	// =====================================================
	// REGISTER
	// =====================================================

	app.Get("/register", func(c *fiber.Ctx) error {

		return c.Render("register", fiber.Map{
			"Title": "Register",
		})
	})

	// =====================================================
	// AUTH ACTION
	// =====================================================

	app.Post("/login", handlers.Login)

	app.Post("/register", handlers.Register)

	app.Get("/logout", handlers.Logout)

	// =====================================================
	// PROTECTED ROUTES
	// =====================================================

	auth := app.Group("/", handlers.AuthMiddleware)

	// =====================================================
	// DASHBOARD
	// =====================================================

	auth.Get("/dashboard", handlers.DashboardPage)

	// =====================================================
	// PRODUCTS
	// =====================================================

	auth.Get("/products", handlers.ProductPage)

	auth.Post("/products/add", handlers.AddProduct)

	auth.Post("/products/upload", handlers.UploadProduct)

	auth.Get("/products/template", handlers.DownloadTemplate)

	auth.Get(
		"/products/delete/:id",
		handlers.ManagerOnly,
		handlers.DeleteProduct,
	)

	// =====================================================
	// INVENTORY
	// =====================================================

	auth.Get("/inventory", handlers.InventoryPage)

	// EXPORT INVENTORY CSV
	auth.Get(
		"/inventory/export",
		handlers.ExportInventory,
	)

	// =====================================================
	// STOCK ADJUSTMENT
	// =====================================================

	auth.Post(
		"/adjustment/create",
		handlers.CreateAdjustment,
	)

	auth.Get(
		"/adjustment",
		handlers.ManagerOnly,
		handlers.AdjustmentPage,
	)

	auth.Get(
		"/adjustment/approve/:id",
		handlers.ManagerOnly,
		handlers.ApproveAdjustment,
	)

	auth.Get(
		"/adjustment/reject/:id",
		handlers.ManagerOnly,
		handlers.RejectAdjustment,
	)

	// EXPORT ADJUSTMENT HISTORY
	auth.Get(
		"/adjustment/export",
		handlers.ManagerOnly,
		handlers.ExportAdjustmentHistory,
	)

	// =====================================================
	// INBOUND
	// =====================================================

	auth.Get("/inbound", handlers.InboundPage)

	auth.Post("/inbound/add", handlers.AddInbound)

	// EXPORT INBOUND HISTORY
	auth.Get(
		"/inbound/export",
		handlers.ExportInboundHistory,
	)

	// =====================================================
	// OUTBOUND
	// =====================================================

	auth.Get("/outbound", handlers.OutboundPage)

	auth.Post("/outbound/create", handlers.CreateOutbound)

	auth.Get("/outbound/sj", handlers.DownloadSJ)

	auth.Get("/outbound/history", handlers.OutboundHistory)

	// =====================================================
	// RETURN
	// =====================================================

	auth.Get("/return", handlers.ReturnPage)

	auth.Post("/return/create", handlers.CreateReturn)

	auth.Get(
		"/return/approval",
		handlers.ManagerOnly,
		handlers.ReturnApprovalPage,
	)

	auth.Get(
		"/return/approve/:id",
		handlers.ManagerOnly,
		handlers.ApproveReturn,
	)

	auth.Get(
		"/return/reject/:id",
		handlers.ManagerOnly,
		handlers.RejectReturn,
	)

	auth.Get(
		"/return/history",
		handlers.ReturnHistory,
	)

	// =====================================================
	// USER APPROVAL
	// =====================================================

	auth.Get(
		"/users/approval",
		handlers.ManagerOnly,
		handlers.UserApprovalPage,
	)

	auth.Get(
		"/users/approve/:id",
		handlers.ManagerOnly,
		handlers.ApproveUser,
	)

	auth.Get(
		"/users/reject/:id",
		handlers.ManagerOnly,
		handlers.RejectUser,
	)

	auth.Get(
		"/users/delete/:id",
		handlers.ManagerOnly,
		handlers.DeleteUser,
	)

	// =====================================================
	// 404
	// =====================================================

	app.Use(func(c *fiber.Ctx) error {

		return c.Status(404).SendString("404 Page Not Found")
	})

	// =====================================================
	// RUN SERVER (SUDAH DISESUAIKAN UNTUK RENDER)
	// =====================================================

	// Render otomatis mengirim port lewat environment variable bernama "PORT"
	port := os.Getenv("PORT")
	if port == "" {
		port = "4004" // Cadangan jika dijalankan di lokal laptop kamu
	}

	log.Printf("Aplikasi WMS berjalan di port %s", port)
	log.Fatal(app.Listen(":" + port))
}
