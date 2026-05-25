package handlers

import (
	"time"

	"github.com/fasdeli/wms-GAFasdeli/config"
	"github.com/gofiber/fiber/v2"
	"golang.org/x/crypto/bcrypt"
)

// =========================
// USER MODEL
// =========================
type UserApproval struct {
	ID        int
	FullName  string
	Email     string
	Role      string
	Status    string
	CreatedAt string
}

// =========================
// REGISTER
// =========================
func Register(c *fiber.Ctx) error {

	fullname := c.FormValue("full_name")
	email := c.FormValue("email")
	password := c.FormValue("password")
	role := c.FormValue("role")

	// =========================
	// VALIDATION
	// =========================

	if fullname == "" ||
		email == "" ||
		password == "" {

		return c.Render("register", fiber.Map{
			"Error": "All fields required",
		})
	}

	// DEFAULT ROLE
	if role == "" {
		role = "STAFF"
	}

	// =========================
	// HASH PASSWORD
	// =========================

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		14,
	)

	if err != nil {
		return err
	}

	// =========================
	// AUTO APPROVAL
	// =========================

	isApproved := 0
	status := "PENDING"

	// manager auto approve
	if role == "MANAGER" {

		isApproved = 1
		status = "ACTIVE"
	}

	// =========================
	// INSERT USER
	// =========================

	_, err = config.DB.Exec(`
		INSERT INTO users
		(
			full_name,
			username,
			email,
			password,
			role,
			email_verified,
			is_approved,
			status,
			created_at
		)
		VALUES (?, ?, ?, ?, ?, 1, ?, ?, NOW())
	`,
		fullname,
		email,
		email,
		string(hash),
		role,
		isApproved,
		status,
	)

	if err != nil {

		return c.Render("register", fiber.Map{
			"Error": "Email already registered",
		})
	}

	// =========================
	// SUCCESS MESSAGE
	// =========================

	if role == "MANAGER" {

		return c.Render("login", fiber.Map{
			"Success": "Manager account created successfully",
		})
	}

	return c.Render("login", fiber.Map{
		"Success": "Register success, waiting manager approval",
	})
}

// =========================
// LOGIN
// =========================
func Login(c *fiber.Ctx) error {

	username := c.FormValue("username")
	password := c.FormValue("password")

	type User struct {
		ID         int
		Password   string
		Role       string
		IsApproved int
		Status     string
	}

	var user User

	err := config.DB.QueryRow(`
		SELECT
			id,
			password,
			role,
			is_approved,
			status
		FROM users
		WHERE username = ?
	`,
		username,
	).Scan(
		&user.ID,
		&user.Password,
		&user.Role,
		&user.IsApproved,
		&user.Status,
	)

	if err != nil {

		return c.Render("login", fiber.Map{
			"Error": "User not found",
		})
	}

	// =========================
	// PASSWORD CHECK
	// =========================

	err = bcrypt.CompareHashAndPassword(
		[]byte(user.Password),
		[]byte(password),
	)

	if err != nil {

		return c.Render("login", fiber.Map{
			"Error": "Wrong password",
		})
	}

	// =========================
	// APPROVAL CHECK
	// =========================

	if user.IsApproved == 0 {

		return c.Render("login", fiber.Map{
			"Error": "Account waiting approval",
		})
	}

	// =========================
	// STATUS CHECK
	// =========================

	if user.Status != "ACTIVE" {

		return c.Render("login", fiber.Map{
			"Error": "Account inactive",
		})
	}

	// =========================
	// UPDATE LAST LOGIN
	// =========================

	config.DB.Exec(`
		UPDATE users
		SET last_login = NOW()
		WHERE id = ?
	`, user.ID)

	// =========================
	// SAVE COOKIE
	// =========================

	c.Cookie(&fiber.Cookie{
		Name:     "user",
		Value:    username,
		HTTPOnly: true,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	c.Cookie(&fiber.Cookie{
		Name:     "role",
		Value:    user.Role,
		HTTPOnly: true,
		Expires:  time.Now().Add(24 * time.Hour),
	})

	return c.Redirect("/")
}

// =========================
// LOGOUT
// =========================
func Logout(c *fiber.Ctx) error {

	c.ClearCookie("user")
	c.ClearCookie("role")

	return c.Redirect("/login")
}

// =========================
// AUTH MIDDLEWARE
// =========================
func AuthMiddleware(c *fiber.Ctx) error {

	user := c.Cookies("user")

	if user == "" {

		return c.Redirect("/login")
	}

	return c.Next()
}

// =========================
// MANAGER ONLY
// =========================
func ManagerOnly(c *fiber.Ctx) error {

	role := c.Cookies("role")

	if role != "MANAGER" &&
		role != "SUPERADMIN" {

		return c.Status(403).SendString("403 Forbidden")
	}

	return c.Next()
}
