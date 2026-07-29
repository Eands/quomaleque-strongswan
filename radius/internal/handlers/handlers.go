package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"vpn-radius/internal/db"
	"vpn-radius/internal/vici"
)

type Handlers struct {
	db   *db.Database
	vici *vici.ViciManager
}

func NewHandlers(database *db.Database, viciManager *vici.ViciManager) *Handlers {
	return &Handlers{db: database, vici: viciManager}
}

func (h *Handlers) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		userID := session.Get("user_id")
		if userID == nil {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *Handlers) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{})
}

func (h *Handlers) Login(c *gin.Context) {
	var req struct {
		Username string `form:"username" binding:"required"`
		Password string `form:"password" binding:"required"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.HTML(http.StatusBadRequest, "login.html", gin.H{"error": "Invalid request"})
		return
	}

	user, err := h.db.GetUserByUsername(req.Username)
	if err != nil || !user.IsActive {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Invalid credentials"})
		return
	}

	session := sessions.Default(c)
	session.Set("user_id", user.ID)
	session.Set("username", user.Username)
	session.Save()

	c.Redirect(http.StatusFound, "/")
}

func (h *Handlers) Logout(c *gin.Context) {
	session := sessions.Default(c)
	session.Clear()
	session.Save()
	c.Redirect(http.StatusFound, "/login")
}

func (h *Handlers) Dashboard(c *gin.Context) {
	users, _ := h.db.ListUsers()
	sessions, _ := h.db.ListSessions()
	activeCount, _ := h.db.GetActiveSessionCount()
	stats, _ := h.vici.GetStats()

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"total_users":     len(users),
		"active_sessions": activeCount,
		"total_sessions":  len(sessions),
		"vpn_stats":       stats,
	})
}

func (h *Handlers) ListUsers(c *gin.Context) {
	users, err := h.db.ListUsers()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
		return
	}
	c.HTML(http.StatusOK, "users.html", gin.H{"users": users})
}

func (h *Handlers) CreateUserPage(c *gin.Context) {
	c.HTML(http.StatusOK, "user_create.html", gin.H{})
}

func (h *Handlers) CreateUser(c *gin.Context) {
	var req struct {
		Username string `form:"username" binding:"required"`
		Password string `form:"password" binding:"required"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.HTML(http.StatusBadRequest, "user_create.html", gin.H{"error": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "user_create.html", gin.H{"error": err.Error()})
		return
	}

	if err := h.db.CreateUser(req.Username, string(hash)); err != nil {
		c.HTML(http.StatusInternalServerError, "user_create.html", gin.H{"error": err.Error()})
		return
	}

	c.Redirect(http.StatusFound, "/users")
}

func (h *Handlers) EditUserPage(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.db.GetUserByID(id)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "User not found"})
		return
	}

	c.HTML(http.StatusOK, "user_edit.html", gin.H{"user": user})
}

func (h *Handlers) UpdateUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Username string `form:"username" binding:"required"`
		Password string `form:"password"`
	}

	if err := c.ShouldBind(&req); err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": err.Error()})
		return
	}

	user, err := h.db.GetUserByID(id)
	if err != nil {
		c.HTML(http.StatusNotFound, "error.html", gin.H{"error": "User not found"})
		return
	}

	passwordHash := user.PasswordHash
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
			return
		}
		passwordHash = string(hash)
	}

	if err := h.db.UpdateUser(id, req.Username, passwordHash); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
		return
	}

	c.Redirect(http.StatusFound, "/users")
}

func (h *Handlers) DeleteUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid user ID"})
		return
	}

	if err := h.db.DeleteUser(id); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
		return
	}

	c.Redirect(http.StatusFound, "/users")
}

func (h *Handlers) ToggleUser(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.HTML(http.StatusBadRequest, "error.html", gin.H{"error": "Invalid user ID"})
		return
	}

	if err := h.db.ToggleUserActive(id); err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
		return
	}

	c.Redirect(http.StatusFound, "/users")
}

func (h *Handlers) VPNStatus(c *gin.Context) {
	connections, _ := h.vici.GetConnections()
	sas, _ := h.vici.GetSAs()
	version, _ := h.vici.GetVersion()

	c.HTML(http.StatusOK, "vpn.html", gin.H{
		"connections": connections,
		"active_sas":  sas,
		"version":     version,
	})
}

func (h *Handlers) CreateConnection(c *gin.Context) {
	var config map[string]interface{}
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.vici.LoadConnection(config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "connection created"})
}

func (h *Handlers) DeleteConnection(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "connection name required"})
		return
	}

	if err := h.vici.UnloadConnection(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "connection removed"})
}

func (h *Handlers) InitiateConnection(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "connection name required"})
		return
	}

	if err := h.vici.InitiateConnection(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "tunnel initiating"})
}

func (h *Handlers) TerminateConnection(c *gin.Context) {
	name := c.Param("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "connection name required"})
		return
	}

	if err := h.vici.TerminateConnection(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "tunnel terminating"})
}

func (h *Handlers) ListSessions(c *gin.Context) {
	sessions, err := h.db.ListSessions()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
		return
	}
	c.HTML(http.StatusOK, "sessions.html", gin.H{"sessions": sessions})
}

func (h *Handlers) ListLogs(c *gin.Context) {
	logs, err := h.db.ListLogs()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
		return
	}
	c.HTML(http.StatusOK, "logs.html", gin.H{"logs": logs})
}

func (h *Handlers) Stats(c *gin.Context) {
	userCount, _ := h.db.CountUsers()
	activeCount, _ := h.db.GetActiveSessionCount()
	viciStats, _ := h.vici.GetStats()
	version, _ := h.vici.GetVersion()

	c.HTML(http.StatusOK, "stats.html", gin.H{
		"user_count":    userCount,
		"active_count":  activeCount,
		"vici_stats":    viciStats,
		"version":       version,
	})
}

func (h *Handlers) SettingsPage(c *gin.Context) {
	settings, err := h.db.GetAllSettings()
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{"error": err.Error()})
		return
	}
	c.HTML(http.StatusOK, "settings.html", gin.H{"settings": settings})
}

func (h *Handlers) UpdateSettings(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for key, value := range req {
		if err := h.db.SetSetting(key, value); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "settings updated"})
}
