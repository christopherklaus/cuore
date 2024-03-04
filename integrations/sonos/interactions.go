package sonos

import (
	"context"
	"cuore/common"
	"cuore/config"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

var (
	baseURL      = "https://api.ws.sonos.com/control/api/v1"
	groups       = map[string]string{}
	players      = map[string]string{}
	groupPlayers = map[string][]string{}
	roomMutex    sync.Mutex
)

func authenticationToken() string {
	token, _ := getToken()
	return fmt.Sprintf("Bearer %s", token.AccessToken)
}

func (s *Sonos) playerOrGroup() string {
	if s.ControlPlayers {
		return "player"
	}
	return "group"
}

func (s *Sonos) UpdateState(state common.Page) {
	s.SetRoom(state.Room) // need to wait for room being set

	s.SetValue(state.CurrentValue)
	s.SetIsPlaying(state.State)
}

func (s *Sonos) SetIsPlaying(isPlaying bool) {
	if s.State.Playing == isPlaying {
		return
	}

	s.State.Playing = isPlaying

	if isPlaying {
		s.Play()
	}

	if !isPlaying {
		s.Pause()
	}
}

func (s *Sonos) SetValue(value int) {
	if s.State.Value == value {
		return
	}

	s.VolumeForGroup(value)
	s.State.Value = value
}

func (s *Sonos) SetRoom(room string) {
	roomMutex.Lock()
	defer roomMutex.Unlock()

	s.State.Room = room

	// There should be an option to decide whether we should go by players or by groups
	if _, ok := players[room]; !ok {
		err := s.UpdateGroupsAndPlayers()
		if err != nil {
			log.Print("Failed to update groups and players")
		}
	}
}

func (s *Sonos) sonosAPIRequest(url string, method string, payload io.Reader) (*http.Response, error) {
	req, _ := http.NewRequest(method, url, payload)
	token, _ := getToken()

	if token == nil || token.Expiry.Before(time.Now()) {
		log.Print("Token expired, refreshing")
		// If the access token is expired or not yet obtained, use the refresh token to get a new one
		tokenSource := getAuthConfig().TokenSource(context.Background(), token)

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

func (s *Sonos) UpdateGroupsAndPlayers() error {
	url := fmt.Sprintf(
		"%s/households/%s/groups",
		baseURL,
		config.Get().SonosHouseholdId,
	)
	res, err := s.sonosAPIRequest(url, "GET", nil)

	if err != nil || res.StatusCode != 200 {
		log.Printf("Failed to make request to Sonos Groups API, %v, %v", res.StatusCode, err)
		return err
	}

	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)

	var response GroupsResponse

	if err := json.Unmarshal(body, &response); err != nil {
		log.Print("Error decoding JSON: ", err)
		return err
	}

	for _, group := range response.Groups {
		groups[group.Name] = group.Id
		groupPlayers[group.Id] = group.PlayerIds
	}

	for _, player := range response.Players {
		players[player.Name] = player.Id
	}

	return nil
}

func (s *Sonos) VolumeForGroup(value int) {
	url := fmt.Sprintf(
		"%s/%ss/%s/%sVolume",
		baseURL,
		s.playerOrGroup(),
		players[s.State.Room],
		s.playerOrGroup(),
	)

	payload := strings.NewReader(fmt.Sprintf("{\"volume\":%d}", value))

	res, err := s.sonosAPIRequest(url, "POST", payload)
	if err != nil {
		log.Printf("Failed to make request to Sonos API, %v", err)
		return
	}
	if res.StatusCode != 200 {
		log.Printf("Failed to make request to Sonos API, %v", res)
		body, err := io.ReadAll(res.Body)
		if err != nil {
			fmt.Println(err)
			return
		}

		log.Printf("Body to String: %s", string(body))
		return
	} else {
		log.Printf("Volume successfully changed to %d for %s", value, s.State.Room)
	}

	defer res.Body.Close()
}

func groupForPlayer(player string) string {
	for group, players := range groupPlayers {
		for _, p := range players {
			if p == player {
				return group
			}
		}
	}

	return ""
}

func (s *Sonos) Play() {
	log.Print("Start playing music in room ", s.State.Room)
	url := fmt.Sprintf(
		"%s/groups/%v/playback/play",
		baseURL,
		groupForPlayer(players[s.State.Room]),
	)

	_, err := s.sonosAPIRequest(url, "POST", nil)

	if err != nil {
		log.Printf("Failed to make request to Sonos API, %v", err)
		return
	}
}

func (s *Sonos) Pause() {
	log.Print("Pause music in room ", s.State.Room)
	url := fmt.Sprintf(
		"%s/groups/%s/playback/pause",
		baseURL,
		groupForPlayer(players[s.State.Room]),
	)

	_, err := s.sonosAPIRequest(url, "POST", nil)

	if err != nil {
		log.Printf("Failed to make request to Sonos API, %v", err)
		return
	}
}
