package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log"
	"math/big"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"webapp/internal/config"
	"webapp/internal/cron"
	"webapp/internal/db"
	"webapp/internal/handlers"
	"webapp/internal/vici"
)

func main() {
	cfg := config.Load()

	database, err := db.Connect(cfg)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	defer database.Close()

	viciClient := vici.NewClient(cfg.ViciAddr, cfg.ViciTimeout, cfg.ViciReconnect)
	go viciClient.Run()

	cron.Start(database, cfg.LogRetentionDays)

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.LoadHTMLGlob("/templates/*")

	auth := handlers.NewAuthHandler(cfg.AppUser, cfg.AppPassword)
	usersH := handlers.NewUsersHandler(database)
	monitorH := handlers.NewMonitorHandler(viciClient)
	settingsH := handlers.NewSettingsHandler(database)

	r.GET("/login", auth.LoginPage)
	r.POST("/login", auth.Login)
	r.GET("/logout", auth.Logout)

	protected := r.Group("/")
	protected.Use(auth.AuthMiddleware())
	{
		protected.GET("/", func(c *gin.Context) {
			c.HTML(http.StatusOK, "dashboard.html", handlers.H("dashboard", gin.H{}))
		})
		protected.GET("/users", func(c *gin.Context) {
			usersH.ListUsers(c)
		})
		protected.GET("/users/create", func(c *gin.Context) {
			usersH.CreateUserPage(c)
		})
		protected.POST("/users/create", func(c *gin.Context) {
			usersH.CreateUser(c)
		})
		protected.GET("/users/:username/edit", func(c *gin.Context) {
			usersH.EditUserPage(c)
		})
		protected.POST("/users/:username/edit", func(c *gin.Context) {
			usersH.EditUser(c)
		})
		protected.GET("/users/:username/delete", func(c *gin.Context) {
			usersH.DeleteUser(c)
		})
		protected.GET("/monitor", func(c *gin.Context) {
			monitorH.ActiveSessions(c)
		})
		protected.GET("/settings", func(c *gin.Context) {
			settingsH.SettingsPage(c)
		})
		protected.POST("/settings", func(c *gin.Context) {
			settingsH.UpdateSettings(c)
		})
	}

	go func() {
		redirect := gin.New()
		redirect.Any("/*path", func(c *gin.Context) {
			host := c.Request.Host
			c.Redirect(http.StatusMovedPermanently, "https://"+host+c.Request.URL.String())
		})
		log.Fatal(redirect.Run(":80"))
	}()

	tlsConfig, err := loadOrGenerateTLS(cfg)
	if err != nil {
		log.Fatalf("TLS configuration failed: %v", err)
	}

	server := &http.Server{
		Addr:      ":443",
		Handler:   r,
		TLSConfig: tlsConfig,
	}

	log.Println("Starting HTTPS server on :443")
	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func loadOrGenerateTLS(cfg *config.Config) (*tls.Config, error) {
	certPath := cfg.SSLCertPath
	keyPath := cfg.SSLKeyPath

	if certPath != "" && keyPath != "" {
		cert, err := tls.LoadX509KeyPair(certPath, keyPath)
		if err != nil {
			return nil, err
		}
		log.Println("Loaded external TLS certificate")
		return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
	}

	certDir := "/certs"
	certFile := certDir + "/server.crt"
	keyFile := certDir + "/server.key"

	if _, err := os.Stat(certFile); err == nil {
		if _, err := os.Stat(keyFile); err == nil {
			cert, err := tls.LoadX509KeyPair(certFile, keyFile)
			if err == nil {
				log.Println("Loaded existing certificate from /certs")
				return &tls.Config{Certificates: []tls.Certificate{cert}}, nil
			}
		}
	}

	log.Println("Generating self-signed certificate...")
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: "VPN Management",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})

	os.MkdirAll(certDir, 0700)
	if err := os.WriteFile(certFile, certPEM, 0644); err != nil {
		log.Printf("Warning: could not save cert: %v", err)
	}
	if err := os.WriteFile(keyFile, keyPEM, 0600); err != nil {
		log.Printf("Warning: could not save key: %v", err)
	}

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &tls.Config{Certificates: []tls.Certificate{tlsCert}}, nil
}
