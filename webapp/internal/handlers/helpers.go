package handlers

import "github.com/gin-gonic/gin"

func H(activePage string, extra gin.H) gin.H {
	if extra == nil {
		extra = gin.H{}
	}
	extra["ActivePage"] = activePage
	extra["LoggedIn"] = true
	return extra
}
