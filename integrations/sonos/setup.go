package sonos

import (
	"cuore/common"
	"cuore/config"

	"golang.org/x/oauth2"
)

const (
	authURL      = "https://api.sonos.com/login/v3/oauth"
	tokenURL     = "https://api.sonos.com/login/v3/oauth/access"
	redirectURL  = "https://heychr.is/new/home"
	providerName = "sonos"
)

var (
	auth  *oauth2.Config
	token *oauth2.Token
)

func init() {
	auth = &oauth2.Config{
		ClientID:     config.Get().SonosClientId,
		ClientSecret: config.Get().SonosClientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{"playback-control-all"},
		Endpoint: oauth2.Endpoint{
			AuthURL:   authURL,
			TokenURL:  tokenURL,
			AuthStyle: oauth2.AuthStyleInHeader,
		},
	}
}

func setToken(newToken *oauth2.Token) error {
	err := common.SaveTokenForProvider(providerName, newToken)
	if err != nil {
		return err
	}

	token = newToken

	return nil
}

func getToken() (*oauth2.Token, error) {
	if token != nil {
		return token, nil
	}

	token, err := common.GetTokenForProvider(providerName)
	if err != nil {
		return nil, err
	}

	return token, nil
}
