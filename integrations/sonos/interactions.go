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
	roomMutex.Lock()
	defer roomMutex.Unlock()

	var room *Room
	if room = s.findRoom(state.Room); room == nil {
		// room does not exist yet, creating new room
		room = &Room{
			Name:  state.Room,
			State: &State{},
		}
		s.Rooms = append(s.Rooms, *room)

		if _, ok := players[state.Room]; !ok {
			err := s.updateGroupsAndPlayers()
			if err != nil {
				log.Print("Failed to update groups and players")
			}
		}
	}

	s.setIsPlaying(state.Status, room)
	s.setValue(state.CurrentValue, room)
}

func (s *Sonos) setIsPlaying(isPlaying bool, room *Room) {
	room.State.Playing = isPlaying

	if isPlaying {
		s.play(*room)
	}

	if !isPlaying {
		s.pause(*room)
	}
}

func (s *Sonos) setValue(value int, room *Room) {
	if room.State.Value == value {
		return
	}

	s.volumeForPlayer(value, *room)
	room.State.Value = value
}

func (s *Sonos) findRoom(roomName string) *Room {
	for _, room := range s.Rooms {
		if room.Name == roomName {
			return &room
		}
	}
	return nil
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

func (s *Sonos) updateGroupsAndPlayers() error {
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

func (s *Sonos) volumeForPlayer(value int, room Room) {
	url := fmt.Sprintf(
		"%s/%ss/%s/%sVolume",
		baseURL,
		s.playerOrGroup(),
		players[room.Name],
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
		log.Printf("Volume successfully changed to %d for %s", value, room.Name)
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

func (s *Sonos) play(room Room) {
	log.Print("Start playing music in room ", room.Name)
	url := fmt.Sprintf(
		"%s/groups/%v/playback/play",
		baseURL,
		groupForPlayer(players[room.Name]),
	)

	_, err := s.sonosAPIRequest(url, "POST", nil)

	if err != nil {
		log.Printf("Failed to make request to Sonos API, %v", err)
		return
	}
}

func (s *Sonos) pause(room Room) {
	log.Print("Pause music in room ", room.Name)
	url := fmt.Sprintf(
		"%s/groups/%s/playback/pause",
		baseURL,
		groupForPlayer(players[room.Name]),
	)

	_, err := s.sonosAPIRequest(url, "POST", nil)

	if err != nil {
		log.Printf("Failed to make request to Sonos API, %v", err)
		return
	}
}
