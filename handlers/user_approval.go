package handlers

import (
	"time"

	"github.com/fasdeli/wms-GAFasdeli/config"
	"github.com/gofiber/fiber/v2"
)

// =========================
// USER MODEL
// =========================
type ApprovalUser struct {
	ID        int
	FullName  string
	Email     string
	Role      string
	Status    string
	CreatedAt string
}

// =========================
// USER APPROVAL PAGE
// =========================
func UserApprovalPage(c *fiber.Ctx) error {

	rows, err := config.DB.Query(`
		SELECT
			id,
			full_name,
			email,
			role,
			status,
			DATE_FORMAT(created_at,'%d-%m-%Y %H:%i')
		FROM users
		ORDER BY created_at DESC
	`)

	if err != nil {
		return err
	}

	defer rows.Close()

	var users []ApprovalUser

	for rows.Next() {

		var u ApprovalUser

		err := rows.Scan(
			&u.ID,
			&u.FullName,
			&u.Email,
			&u.Role,
			&u.Status,
			&u.CreatedAt,
		)

		if err != nil {
			continue
		}

		users = append(users, u)
	}

	loginUser := c.Cookies("user")
	loginRole := c.Cookies("role")

	return c.Render("user_approval", fiber.Map{
		"Title": "User Approval",

		"Users": users,

		"LoginUser": loginUser,
		"LoginRole": loginRole,
	})
}

// =========================
// APPROVE USER
// =========================
func ApproveUser(c *fiber.Ctx) error {

	id := c.Params("id")

	approvedBy := c.Cookies("user")

	_, err := config.DB.Exec(`
		UPDATE users
		SET
			is_approved = 1,
			status = 'ACTIVE',
			last_login = NULL
		WHERE id = ?
	`, id)

	if err != nil {
		return err
	}

	// LOG
	config.DB.Exec(`
		INSERT INTO activity_logs
		(
			username,
			activity,
			created_at
		)
		VALUES (?, ?, ?)
	`,
		approvedBy,
		"Approved user ID "+id,
		time.Now(),
	)

	return c.Redirect("/users/approval")
}

// =========================
// REJECT USER
// =========================
func RejectUser(c *fiber.Ctx) error {

	id := c.Params("id")

	rejectedBy := c.Cookies("user")

	_, err := config.DB.Exec(`
		UPDATE users
		SET
			status = 'REJECTED',
			is_approved = 0
		WHERE id = ?
	`, id)

	if err != nil {
		return err
	}

	// LOG
	config.DB.Exec(`
		INSERT INTO activity_logs
		(
			username,
			activity,
			created_at
		)
		VALUES (?, ?, ?)
	`,
		rejectedBy,
		"Rejected user ID "+id,
		time.Now(),
	)

	return c.Redirect("/users/approval")
}

// =========================
// DELETE USER
// =========================
func DeleteUser(c *fiber.Ctx) error {

	id := c.Params("id")

	deletedBy := c.Cookies("user")

	// DELETE USER
	_, err := config.DB.Exec(`
		DELETE FROM users
		WHERE id = ?
	`, id)

	if err != nil {
		return err
	}

	// LOG
	config.DB.Exec(`
		INSERT INTO activity_logs
		(
			username,
			activity,
			created_at
		)
		VALUES (?, ?, ?)
	`,
		deletedBy,
		"Deleted user ID "+id,
		time.Now(),
	)

	return c.Redirect("/users/approval")
}
