package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"webapp/internal/vici"
)

type MonitorHandler struct {
	ViciClient *vici.Client
}

func NewMonitorHandler(vc *vici.Client) *MonitorHandler {
	return &MonitorHandler{ViciClient: vc}
}

func (h *MonitorHandler) ActiveSessions(c *gin.Context) {
	sessions := h.ViciClient.GetActiveSessions()
	c.HTML(http.StatusOK, "monitor.html", H("monitor", gin.H{"sessions": sessions}))
}
