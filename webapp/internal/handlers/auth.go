package handlers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	AppUser     string
	AppPassword string
	secret      string
}

func NewAuthHandler(appUser, appPassword string) *AuthHandler {
	return &AuthHandler{
		AppUser:     appUser,
		AppPassword: appPassword,
		secret:      fmt.Sprintf("%s:%s:%d", appUser, appPassword, time.Now().UnixNano()),
	}
}

func (a *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/login" {
			c.Next()
			return
		}

		if c.Request.URL.Path == "/static/" || strings.HasPrefix(c.Request.URL.Path, "/static/") {
			c.Next()
			return
		}

		authenticated := false
		cookie, err := c.Cookie("vpn_auth")
		if err == nil {
			parts := strings.SplitN(cookie, ":", 2)
			if len(parts) == 2 {
				decoded, err := base64.StdEncoding.DecodeString(parts[0])
				if err == nil && string(decoded) == a.AppUser {
					expectedMAC := computeHMAC(parts[0], a.secret)
					if hmac.Equal([]byte(parts[1]), []byte(expectedMAC)) {
						authenticated = true
					}
				}
			}
		}

		if !authenticated {
			c.Redirect(http.StatusFound, "/login")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (a *AuthHandler) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{"error": ""})
}

func (a *AuthHandler) Login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	if username != a.AppUser || password != a.AppPassword {
		c.HTML(http.StatusUnauthorized, "login.html", gin.H{"error": "Invalid credentials"})
		return
	}

	encoded := base64.StdEncoding.EncodeToString([]byte(username))
	mac := computeHMAC(encoded, a.secret)
	cookieValue := fmt.Sprintf("%s:%s", encoded, mac)

	c.SetCookie("vpn_auth", cookieValue, 86400, "/", "", true, true)
	c.Redirect(http.StatusFound, "/")
}

func (a *AuthHandler) Logout(c *gin.Context) {
	c.SetCookie("vpn_auth", "", -1, "/", "", true, true)
	c.Redirect(http.StatusFound, "/login")
}

func computeHMAC(message, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(message))
	return fmt.Sprintf("%x", mac.Sum(nil))
}
