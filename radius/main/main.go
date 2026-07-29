package main

import (
	"log"
	"os"

	"vpn-radius/internal/db"
	"vpn-radius/internal/handlers"
	"vpn-radius/internal/radius"
	"vpn-radius/internal/vici"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func main() {
	database, err := db.NewDatabase("./data/users.db")
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	if err := database.RunMigrations(); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	viciManager, err := vici.NewViciManager()
	if err != nil {
		log.Printf("Warning: Failed to connect to VICI: %v", err)
	}
	if viciManager != nil {
		defer viciManager.Close()
	}

	radiusSecret := os.Getenv("RADIUS_SECRET")
	if radiusSecret == "" {
		radiusSecret = "HpE98gAFA4OaJaHYU46M"
	}

	radiusServer := radius.NewServer(database, radiusSecret)
	go func() {
		if err := radiusServer.StartAuth(":1812"); err != nil {
			log.Printf("Warning: RADIUS auth server: %v", err)
		}
	}()
	go func() {
		if err := radiusServer.StartAcct(":1813"); err != nil {
			log.Printf("Warning: RADIUS acct server: %v", err)
		}
	}()
	defer radiusServer.Close()

	router := gin.Default()

	store := cookie.NewStore([]byte("vpn-radius-session-secret-key"))
	router.Use(sessions.Sessions("vpn_session", store))

	router.Static("/static", "./web/static")
	router.LoadHTMLGlob("web/templates/*")

	h := handlers.NewHandlers(database, viciManager)

	router.GET("/login", h.LoginPage)
	router.POST("/login", h.Login)
	router.GET("/logout", h.Logout)

	auth := router.Group("/")
	auth.Use(h.AuthMiddleware())
	{
		auth.GET("/", h.Dashboard)

		auth.GET("/users", h.ListUsers)
		auth.GET("/users/create", h.CreateUserPage)
		auth.POST("/users", h.CreateUser)
		auth.GET("/users/:id/edit", h.EditUserPage)
		auth.PUT("/users/:id", h.UpdateUser)
		auth.DELETE("/users/:id", h.DeleteUser)
		auth.POST("/users/:id/toggle", h.ToggleUser)

		auth.GET("/vpn", h.VPNStatus)
		auth.POST("/vpn/connections", h.CreateConnection)
		auth.DELETE("/vpn/connections/:name", h.DeleteConnection)
		auth.POST("/vpn/connections/:name/initiate", h.InitiateConnection)
		auth.POST("/vpn/connections/:name/terminate", h.TerminateConnection)

		auth.GET("/sessions", h.ListSessions)
		auth.GET("/logs", h.ListLogs)
		auth.GET("/stats", h.Stats)

		auth.GET("/settings", h.SettingsPage)
		auth.PUT("/settings", h.UpdateSettings)
	}

	log.Println("Web server starting on :8080")
	if err := router.Run(":8080"); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}
