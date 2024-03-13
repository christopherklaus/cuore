package hue

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

func (h Hue) AuthorizationHandlers(hueRoutes *gin.RouterGroup) {
	hueRoutes.GET("/", func(c *gin.Context) {
		authURL := getAuthConfig().AuthCodeURL("state", oauth2.AccessTypeOffline) //should be random code
		c.Redirect(http.StatusFound, authURL)
	})

	hueRoutes.GET("/auth", func(c *gin.Context) {
		code := c.Query("code")
		newToken, err := getAuthConfig().Exchange(c, code)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		setToken(newToken)

		c.JSON(http.StatusOK, gin.H{"token": newToken})

	})

	hueRoutes.GET("/refresh", func(c *gin.Context) {
		token := &oauth2.Token{
			RefreshToken: c.Query("refresh_token"),
		}
		newToken, err := getAuthConfig().TokenSource(c, token).Token()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		setToken(newToken)

		c.JSON(http.StatusOK, gin.H{"token": newToken})
	})
}
