package sonos

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

func (s Sonos) AuthorizationHandlers(sonosRoutes *gin.RouterGroup) {
	sonosRoutes.GET("/", func(c *gin.Context) {
		authURL := auth.AuthCodeURL("state", oauth2.AccessTypeOffline) //should be random code
		c.Redirect(http.StatusFound, authURL)
	})

	sonosRoutes.GET("/auth", func(c *gin.Context) {
		code := c.Query("code")
		newToken, err := auth.Exchange(c, code)

		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		setToken(newToken)

		c.JSON(http.StatusOK, gin.H{"token": newToken})

	})

	sonosRoutes.GET("/refresh", func(c *gin.Context) {
		token := &oauth2.Token{
			RefreshToken: c.Query("refresh_token"),
		}
		newToken, err := auth.TokenSource(c, token).Token()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		setToken(newToken)

		c.JSON(http.StatusOK, gin.H{"token": newToken})
	})
}
