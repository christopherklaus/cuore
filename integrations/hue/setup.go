package hue

import (
	"cuore/common"
	"cuore/config"

	"golang.org/x/oauth2"
)

const (
	authURL      = "https://api.meethue.com/v2/oauth2/authorize"
	tokenURL     = "https://api.meethue.com/v2/oauth2/token"
	redirectURL  = "http://localhost/integrations/hue/auth"
	providerName = "hue"
)

var (
	token *oauth2.Token
)

func getAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     config.Get().HueClientId,
		ClientSecret: config.Get().HueClientSecret,
		RedirectURL:  redirectURL,
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
