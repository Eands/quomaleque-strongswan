package handlers

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"webapp/internal/models"
)

type UsersHandler struct {
	DB *sql.DB
}

func NewUsersHandler(db *sql.DB) *UsersHandler {
	return &UsersHandler{DB: db}
}

func (h *UsersHandler) ListUsers(c *gin.Context) {
	rows, err := h.DB.Query(`
		SELECT rc.id, rc.username,
			COALESCE((SELECT value FROM radcheck rc2
				WHERE rc2.username = rc.username AND rc2.attribute = 'Cleartext-Password'), '') as password,
			COALESCE((SELECT value::int FROM radcheck rc3
				WHERE rc3.username = rc.username AND rc3.attribute = 'Simultaneous-Use'), 0) as simultaneous_use,
			COALESCE((SELECT groupname FROM radusergroup rug
				WHERE rug.username = rc.username ORDER BY priority LIMIT 1), '') as groupname
		FROM radcheck rc
		WHERE rc.attribute = 'Cleartext-Password'
		ORDER BY rc.username
	`)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", H("users", gin.H{"error": "Failed to load users: " + err.Error()}))
		return
	}
	defer rows.Close()

	var users []models.RadiusUser
	for rows.Next() {
		var u models.RadiusUser
		if err := rows.Scan(&u.ID, &u.Username, &u.Password, &u.SimultaneousUse, &u.Group); err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", H("users", gin.H{"error": "Failed to scan user: " + err.Error()}))
			return
		}
		users = append(users, u)
	}

	c.HTML(http.StatusOK, "users_list.html", H("users", gin.H{"users": users}))
}

func (h *UsersHandler) CreateUserPage(c *gin.Context) {
	c.HTML(http.StatusOK, "users_create.html", H("users", gin.H{}))
}

func (h *UsersHandler) CreateUser(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	group := c.PostForm("group")
	maxConn := c.PostForm("max_connections")

	if username == "" || password == "" {
		c.HTML(http.StatusBadRequest, "users_create.html", H("users", gin.H{"error": "Username and password are required"}))
		return
	}

	maxConnInt := 0
	if maxConn != "" {
		maxConnInt, _ = strconv.Atoi(maxConn)
	}

	tx, err := h.DB.Begin()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "users_create.html", H("users", gin.H{"error": "Transaction error: " + err.Error()}))
		return
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT INTO radcheck (username, attribute, op, value) VALUES ($1, 'Cleartext-Password', ':=', $2)`, username, password)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "users_create.html", H("users", gin.H{"error": "Failed to create password entry: " + err.Error()}))
		return
	}

	if maxConnInt > 0 {
		_, err = tx.Exec(`INSERT INTO radcheck (username, attribute, op, value) VALUES ($1, 'Simultaneous-Use', ':=', $2)`, username, strconv.Itoa(maxConnInt))
		if err != nil {
			c.HTML(http.StatusInternalServerError, "users_create.html", H("users", gin.H{"error": "Failed to create Simultaneous-Use entry: " + err.Error()}))
			return
		}
	}

	if group != "" {
		_, err = tx.Exec(`INSERT INTO radusergroup (username, groupname, priority) VALUES ($1, $2, 1)`, username, group)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "users_create.html", H("users", gin.H{"error": "Failed to create group entry: " + err.Error()}))
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.HTML(http.StatusInternalServerError, "users_create.html", H("users", gin.H{"error": "Commit error: " + err.Error()}))
		return
	}

	c.Redirect(http.StatusFound, "/users")
}

func (h *UsersHandler) EditUserPage(c *gin.Context) {
	username := c.Param("username")

	var u models.RadiusUser
	u.Username = username

	err := h.DB.QueryRow(`SELECT value FROM radcheck WHERE username = $1 AND attribute = 'Cleartext-Password'`, username).Scan(&u.Password)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", H("users", gin.H{"error": "User not found"}))
		return
	}

	h.DB.QueryRow(`SELECT COALESCE(value::int, 0) FROM radcheck WHERE username = $1 AND attribute = 'Simultaneous-Use'`, username).Scan(&u.SimultaneousUse)
	h.DB.QueryRow(`SELECT COALESCE(groupname, '') FROM radusergroup WHERE username = $1 ORDER BY priority LIMIT 1`, username).Scan(&u.Group)

	c.HTML(http.StatusOK, "users_edit.html", H("users", gin.H{"user": u}))
}

func (h *UsersHandler) EditUser(c *gin.Context) {
	username := c.Param("username")
	password := c.PostForm("password")
	group := c.PostForm("group")
	maxConn := c.PostForm("max_connections")

	tx, err := h.DB.Begin()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "users_edit.html", H("users", gin.H{"user": models.RadiusUser{Username: username}, "error": "Transaction error: " + err.Error()}))
		return
	}
	defer tx.Rollback()

	if password != "" {
		_, err = tx.Exec(`UPDATE radcheck SET value = $1 WHERE username = $2 AND attribute = 'Cleartext-Password'`, password, username)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "users_edit.html", H("users", gin.H{"user": models.RadiusUser{Username: username}, "error": "Failed to update password: " + err.Error()}))
			return
		}
	}

	maxConnInt := 0
	if maxConn != "" {
		maxConnInt, _ = strconv.Atoi(maxConn)
	}

	_, err = tx.Exec(`DELETE FROM radcheck WHERE username = $1 AND attribute = 'Simultaneous-Use'`, username)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "users_edit.html", H("users", gin.H{"user": models.RadiusUser{Username: username}, "error": "Failed to update Simultaneous-Use: " + err.Error()}))
		return
	}

	if maxConnInt > 0 {
		_, err = tx.Exec(`INSERT INTO radcheck (username, attribute, op, value) VALUES ($1, 'Simultaneous-Use', ':=', $2)`, username, strconv.Itoa(maxConnInt))
		if err != nil {
			c.HTML(http.StatusInternalServerError, "users_edit.html", H("users", gin.H{"user": models.RadiusUser{Username: username}, "error": "Failed to insert Simultaneous-Use: " + err.Error()}))
			return
		}
	}

	_, err = tx.Exec(`DELETE FROM radusergroup WHERE username = $1`, username)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "users_edit.html", H("users", gin.H{"user": models.RadiusUser{Username: username}, "error": "Failed to update group: " + err.Error()}))
		return
	}

	if group != "" {
		_, err = tx.Exec(`INSERT INTO radusergroup (username, groupname, priority) VALUES ($1, $2, 1)`, username, group)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "users_edit.html", H("users", gin.H{"user": models.RadiusUser{Username: username}, "error": "Failed to insert group: " + err.Error()}))
			return
		}
	}

	if err := tx.Commit(); err != nil {
		c.HTML(http.StatusInternalServerError, "users_edit.html", H("users", gin.H{"user": models.RadiusUser{Username: username}, "error": "Commit error: " + err.Error()}))
		return
	}

	c.Redirect(http.StatusFound, "/users")
}

func (h *UsersHandler) DeleteUser(c *gin.Context) {
	username := c.Param("username")

	tx, err := h.DB.Begin()
	if err != nil {
		c.Redirect(http.StatusFound, "/users")
		return
	}
	defer tx.Rollback()

	tx.Exec(`DELETE FROM radcheck WHERE username = $1`, username)
	tx.Exec(`DELETE FROM radreply WHERE username = $1`, username)
	tx.Exec(`DELETE FROM radusergroup WHERE username = $1`, username)
	tx.Commit()

	c.Redirect(http.StatusFound, "/users")
}
