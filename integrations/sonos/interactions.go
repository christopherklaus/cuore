package sonos

import (
	"context"
	"cuore/config"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

var (
	baseURL = "https://api.ws.sonos.com/control/api/v1"
)

func authenticationToken() string {
	token, _ := getToken()
	return fmt.Sprintf("Bearer %s", token.AccessToken)
}

func (s Sonos) sonosAPIRequest(url string, method string, payload io.Reader) (*http.Response, error) {
	req, _ := http.NewRequest(method, url, payload)
	token, _ := getToken()

	if token == nil || token.Expiry.Before(time.Now()) {
		// If the access token is expired or not yet obtained, use the refresh token to get a new one
		tokenSource := auth.TokenSource(context.Background(), token)

		newToken, err := tokenSource.Token()
		if err != nil {
			// Handle error, e.g., log out the user or prompt for reauthentication
			return nil, err
		}
		setToken(newToken)
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Authorization", authenticationToken())

	return http.DefaultClient.Do(req)
}

func (s Sonos) GetGroupIdByRoomName(room string) (string, error) {
	url := fmt.Sprintf("%s/households/%s/groups", baseURL, config.Get().SonosHouseholdId)
	res, _ := s.sonosAPIRequest(url, "GET", nil)
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	var response GroupsResponse

	if err := json.Unmarshal(body, &response); err != nil {
		log.Print("Error decoding JSON: ", err)
		return "", err
	}

	for _, group := range response.Groups {
		if group.Name == room {
			return group.Id, nil
		}
	}

	return "", nil // need to return not found error
}

func (s Sonos) VolumeForGroup(value int, groupName string) {
	log.Print("Volume change triggered for ", groupName)
	groupId, _ := s.GetGroupIdByRoomName(groupName)
	url := fmt.Sprintf("%s/groups/%s/groupVolume", baseURL, groupId)

	payload := strings.NewReader(fmt.Sprintf("{\"volume\":%d}", value))

	_, err := s.sonosAPIRequest(url, "POST", payload)
	if err != nil {
		log.Printf("Failed to make request to Sonos API, %v", err)
		return
	}
}

func (s Sonos) PlayPause(groupName string) {
	log.Print("Play / Pause triggered for ", groupName)
	// TODO: toggle can go out of sync, rather do play / pause

	groupId, _ := s.GetGroupIdByRoomName(groupName)
	url := fmt.Sprintf("%s/groups/%s/playback/togglePlayPause", baseURL, groupId)

	_, err := s.sonosAPIRequest(url, "POST", nil)

	if err != nil {
		log.Printf("Failed to make request to Sonos API, %v", err)
		return
	}
}