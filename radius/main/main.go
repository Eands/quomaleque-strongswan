package main

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/bcrypt"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
	"layeh.com/radius/rfc2869"

	"vpn-radius/internal/db"
	"vpn-radius/internal/eap"
	"vpn-radius/internal/handlers"
	"vpn-radius/internal/vici"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
)

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/users.db"
	}

	database, err := db.NewDatabase(dbPath)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	if err := database.RunMigrations(); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	if err := database.Seed(); err != nil {
		log.Printf("Warning: seed failed: %v", err)
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
		log.Fatal("Cannot find env RADIUS_SECRET")
	}
	secret := []byte(radiusSecret)

	authServer := &radius.PacketServer{
		Addr:         ":1812",
		Network:      "udp",
		Handler:      radius.HandlerFunc(makeRadiusHandler(database)),
		SecretSource: radius.StaticSecretSource(secret),
	}

	acctServer := &radius.PacketServer{
		Addr:         ":1813",
		Network:      "udp",
		Handler:      radius.HandlerFunc(makeRadiusHandler(database)),
		SecretSource: radius.StaticSecretSource(secret),
	}

	go func() {
		log.Printf("RADIUS authentication server starting on :1812")
		if err := authServer.ListenAndServe(); err != nil && err != radius.ErrServerShutdown {
			log.Printf("RADIUS auth server error: %v", err)
		}
	}()

	go func() {
		log.Printf("RADIUS accounting server starting on :1813")
		if err := acctServer.ListenAndServe(); err != nil && err != radius.ErrServerShutdown {
			log.Printf("RADIUS acct server error: %v", err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	router := gin.Default()

	store := cookie.NewStore([]byte("zi4BnL1T3I/EXPAWcsQ55tdIH3KnSdCDQf5r+qAmpSnwryoIG61VO/F/JD+7w2OeUFcsYEMAQ6VsOdAFjkBlMQ==")) //todo to env
	store.Options(sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, //7 дней
		HttpOnly: true,
		Secure:   false, //todo move to https
		SameSite: http.SameSiteLaxMode,
	})

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

	go func() {
		log.Println("Web server starting on :8080")
		if err := router.Run(":8080"); err != nil {
			log.Fatal("Failed to start web server:", err)
		}
	}()

	<-ctx.Done()
	log.Println("Shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*60)
	defer cancel()

	authServer.Shutdown(shutdownCtx)
	acctServer.Shutdown(shutdownCtx)
}

func makeRadiusHandler(database *db.Database) func(w radius.ResponseWriter, r *radius.Request) {
	return func(w radius.ResponseWriter, r *radius.Request) {
		switch r.Code {
		case radius.CodeAccessRequest:
			handleAccessRequest(w, r, database)
		case radius.CodeAccountingRequest:
			handleAccountingRequest(w, r, database)
		default:
			log.Printf("RADIUS: unsupported code %d from %s", r.Code, r.RemoteAddr)
		}
	}
}

func clientIP(addr net.Addr) string {
	s := addr.String()
	if host, _, err := net.SplitHostPort(s); err == nil {
		return host
	}
	return s
}

func handleAccessRequest(w radius.ResponseWriter, r *radius.Request, database *db.Database) {
	username := rfc2865.UserName_GetString(r.Packet)
	clientIPStr := clientIP(r.RemoteAddr)

	if username == "" {
		w.Write(r.Response(radius.CodeAccessReject))
		database.InsertRadiusLog("", "Access-Request", "Reject", clientIPStr)
		return
	}

	user, err := database.GetUserByUsername(username)
	if err != nil || !user.IsActive {
		log.Printf("RADIUS: auth failed for %s: %v", username, err)
		w.Write(r.Response(radius.CodeAccessReject))
		database.InsertRadiusLog(username, "Access-Request", "Reject", clientIPStr)
		return
	}

	// Detect EAP vs PAP
	method := eap.DetectMethod(r)
	switch method {
	case eap.AuthMethodMSCHAPv2:
		handleEAP(w, r, database, user, clientIPStr)
	case eap.AuthMethodPAP:
		handlePAP(w, r, database, user, clientIPStr)
	default:
		w.Write(r.Response(radius.CodeAccessReject))
		database.InsertRadiusLog(username, "Access-Request", "Reject", clientIPStr)
	}
}

func handleEAP(w radius.ResponseWriter, r *radius.Request, database *db.Database, user *db.User, clientIPStr string) {
	eapData := rfc2869.EAPMessage_Get(r.Packet)
	stateToken := rfc2865.State_GetString(r.Packet)
	code := byte(0)
	if len(eapData) > 0 {
		code = eapData[0]
	}

	switch {
	case code == eapCodeResponse && len(stateToken) == 0:
		// First round: EAP-Response/Identity → send Challenge
		resp, token, err := eap.HandleIdentity(r, user.Username)
		if err != nil {
			log.Printf("RADIUS: EAP challenge failed for %s: %v", user.Username, err)
			w.Write(r.Response(radius.CodeAccessReject))
			database.InsertRadiusLog(user.Username, "Access-Request", "Reject", clientIPStr)
			return
		}
		rfc2865.State_SetString(resp, token)
		w.Write(resp)
		log.Printf("RADIUS: EAP challenge sent for %s", user.Username)
		database.InsertRadiusLog(user.Username, "Access-Request", "Challenge", clientIPStr)

	case code == eapCodeResponse && len(stateToken) > 0:
		// Second round: EAP-Response/MSCHAPv2 → verify
		resp := eap.HandleResponse(r, stateToken, user.NTHash)
		if resp.Code == radius.CodeAccessAccept {
			if user.MaxConnections > 0 {
				activeCount, _ := database.GetActiveSessionCountByUsername(user.Username)
				if activeCount >= user.MaxConnections {
					log.Printf("RADIUS: limit reached for %s (%d/%d)", user.Username, activeCount, user.MaxConnections)
					w.Write(r.Response(radius.CodeAccessReject))
					database.InsertRadiusLog(user.Username, "Access-Request", "Reject", clientIPStr)
					return
				}
			}
			log.Printf("RADIUS: EAP auth success for %s", user.Username)
			database.InsertRadiusLog(user.Username, "Access-Request", "Accept", clientIPStr)
		} else {
			log.Printf("RADIUS: EAP auth failed for %s", user.Username)
			database.InsertRadiusLog(user.Username, "Access-Request", "Reject", clientIPStr)
		}
		w.Write(resp)

	default:
		w.Write(r.Response(radius.CodeAccessReject))
		database.InsertRadiusLog(user.Username, "Access-Request", "Reject", clientIPStr)
	}
}

func handlePAP(w radius.ResponseWriter, r *radius.Request, database *db.Database, user *db.User, clientIPStr string) {
	password := rfc2865.UserPassword_GetString(r.Packet)

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		log.Printf("RADIUS: bad password for %s", user.Username)
		w.Write(r.Response(radius.CodeAccessReject))
		database.InsertRadiusLog(user.Username, "Access-Request", "Reject", clientIPStr)
		return
	}

	if user.MaxConnections > 0 {
		activeCount, _ := database.GetActiveSessionCountByUsername(user.Username)
		if activeCount >= user.MaxConnections {
			log.Printf("RADIUS: limit reached for %s (%d/%d)", user.Username, activeCount, user.MaxConnections)
			w.Write(r.Response(radius.CodeAccessReject))
			database.InsertRadiusLog(user.Username, "Access-Request", "Reject", clientIPStr)
			return
		}
	}

	log.Printf("RADIUS: PAP auth success for %s", user.Username)
	w.Write(r.Response(radius.CodeAccessAccept))
	database.InsertRadiusLog(user.Username, "Access-Request", "Accept", clientIPStr)
}

// EAP constants for code detection
const (
	eapCodeResponse = 2
)

func handleAccountingRequest(w radius.ResponseWriter, r *radius.Request, database *db.Database) {
	username := rfc2865.UserName_GetString(r.Packet)
	acctStatusType := rfc2866.AcctStatusType_Get(r.Packet)
	sessionID := rfc2866.AcctSessionID_GetString(r.Packet)
	framedIP := rfc2865.FramedIPAddress_Get(r.Packet).String()

	switch acctStatusType {
	case rfc2866.AcctStatusType_Value_Start:
		database.InsertSessionLog(username, framedIP, sessionID)
		log.Printf("RADIUS: accounting start for %s, session %s, IP %s", username, sessionID, framedIP)
	case rfc2866.AcctStatusType_Value_Stop:
		inputOctets := rfc2866.AcctInputOctets_Get(r.Packet)
		outputOctets := rfc2866.AcctOutputOctets_Get(r.Packet)
		database.UpdateSessionStop(sessionID, int64(inputOctets), int64(outputOctets))
		log.Printf("RADIUS: accounting stop for %s, session %s", username, sessionID)
	case rfc2866.AcctStatusType_Value_InterimUpdate:
		inputOctets := rfc2866.AcctInputOctets_Get(r.Packet)
		outputOctets := rfc2866.AcctOutputOctets_Get(r.Packet)
		database.UpdateSessionInterim(sessionID, int64(inputOctets), int64(outputOctets))
		log.Printf("RADIUS: accounting interim for %s, session %s", username, sessionID)
	}

	w.Write(r.Response(radius.CodeAccountingResponse))
	database.InsertRadiusLog(username, "Accounting-Request", "OK", clientIP(r.RemoteAddr))
}
