package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fasdeli/wms-GAFasdeli/config"
	"github.com/fasdeli/wms-GAFasdeli/handlers"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor" // Adaptor wajib untuk Vercel
	"github.com/gofiber/template/html/v2"
)

// Fungsi Handler ini yang akan dipanggil secara otomatis oleh Vercel
func Handler(w http.ResponseWriter, r *http.Request) {
	// 1. Inisialisasi Database tiap kali fungsi dipanggil
	config.ConnectDB()

	// 2. Seting HTML Engine secara absolut agar Vercel tidak bingung mencari folder
	templateDir := filepath.Join(".", "templates")
	engine := html.New(templateDir, ".html")

	// 3. Buat App Fiber Baru
	app := fiber.New(fiber.Config{
		Views: engine,
	})

	// 4. Folder Static
	app.Static("/static", "./static")
	app.Static("/uploads", "./uploads")

	// 5. Rute Aplikasi (Sama persis seperti kodingan lu sebelumnya)
	app.Get("/", func(c *fiber.Ctx) error {
		user := c.Cookies("user")
		if user == "" {
			return c.Redirect("/login")
		}
		return handlers.DashboardPage(c)
	})

	app.Get("/login", func(c *fiber.Ctx) error {
		if c.Cookies("user") != "" {
			return c.Redirect("/")
		}
		return c.Render("login", fiber.Map{"Title": "Login"})
	})

	app.Get("/register", func(c *fiber.Ctx) error {
		return c.Render("register", fiber.Map{"Title": "Register"})
	})

	app.Post("/login", handlers.Login)
	app.Post("/register", handlers.Register)
	app.Get("/logout", handlers.Logout)

	auth := app.Group("/", handlers.AuthMiddleware)
	auth.Get("/dashboard", handlers.DashboardPage)
	auth.Get("/products", handlers.ProductPage)
	auth.Post("/products/add", handlers.AddProduct)
	auth.Post("/products/upload", handlers.UploadProduct)
	auth.Get("/products/template", handlers.DownloadTemplate)
	auth.Get("/products/delete/:id", handlers.ManagerOnly, handlers.DeleteProduct)

	auth.Get("/inventory", handlers.InventoryPage)
	auth.Get("/inventory/export", handlers.ExportInventory)

	auth.Post("/adjustment/create", handlers.CreateAdjustment)
	auth.Get("/adjustment", handlers.ManagerOnly, handlers.AdjustmentPage)
	auth.Get("/adjustment/approve/:id", handlers.ManagerOnly, handlers.ApproveAdjustment)
	auth.Get("/adjustment/reject/:id", handlers.ManagerOnly, handlers.RejectAdjustment)
	auth.Get("/adjustment/export", handlers.ManagerOnly, handlers.ExportAdjustmentHistory)

	auth.Get("/inbound", handlers.InboundPage)
	auth.Post("/inbound/add", handlers.AddInbound)
	auth.Get("/inbound/export", handlers.ExportInboundHistory)

	auth.Get("/outbound", handlers.OutboundPage)
	app.Post("/outbound/create", handlers.CreateOutbound)
	auth.Get("/outbound/sj", handlers.DownloadSJ)
	auth.Get("/outbound/history", handlers.OutboundHistory)

	auth.Get("/return", handlers.ReturnPage)
	auth.Post("/return/create", handlers.CreateReturn)
	auth.Get("/return/approval", handlers.ManagerOnly, handlers.ReturnApprovalPage)
	auth.Get("/return/approve/:id", handlers.ManagerOnly, handlers.ApproveReturn)
	auth.Get("/return/reject/:id", handlers.ManagerOnly, handlers.RejectReturn)
	auth.Get("/return/history", handlers.ReturnHistory)

	auth.Get("/users/approval", handlers.ManagerOnly, handlers.UserApprovalPage)
	auth.Get("/users/approve/:id", handlers.ManagerOnly, handlers.ApproveUser)
	auth.Get("/users/reject/:id", handlers.ManagerOnly, handlers.RejectUser)
	auth.Get("/users/delete/:id", handlers.ManagerOnly, handlers.DeleteUser)

	app.Use(func(c *fiber.Ctx) error {
		return c.Status(404).SendString("404 Page Not Found")
	})

	// 6. Alihkan request dari Vercel ke dalam aplikasi Fiber lu
	adaptor.FiberApp(app).ServeHTTP(w, r)
}

// Tetap sediakan func main biasa agar aplikasi lu tetap bisa dijalankan di lokal laptop lu
func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}
	log.Printf("Menjalankan server lokal di port %s", port)

	// Mode lokal pake http standar biar aman
	http.HandleFunc("/", Handler)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
