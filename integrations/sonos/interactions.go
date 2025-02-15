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
	groups       = map[string]Group{}
	players      = map[string]Player{}
	groupPlayers = map[string][]string{}
	roomMutex    sync.Mutex
)

// Add these types to match the API response structure
type GroupsResponse struct {
	Groups    []Group  `json:"groups"`
	Players   []Player `json:"players"`
	Household struct {
		Id string `json:"id"`
	} `json:"household"`
}

type Group struct {
	Id            string   `json:"id"`
	Name          string   `json:"name"`
	CoordinatorId string   `json:"coordinatorId"`
	PlaybackState string   `json:"playbackState"`
	PlayerIds     []string `json:"playerIds"`
}

type Player struct {
	Id             string   `json:"id"`
	Name           string   `json:"name"`
	WebSocketUrl   string   `json:"websocketUrl"`
	SWVersion      string   `json:"softwareVersion"`
	APIVersion     string   `json:"apiVersion"`
	MinAPIVersion  string   `json:"minApiVersion"`
	IsUnregistered bool     `json:"isUnregistered"`
	Capabilities   []string `json:"capabilities"`
	DeviceIds      []string `json:"deviceIds"`
}

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

func (s *Sonos) discoverHouseholds() error {
	url := fmt.Sprintf(
		"%s/households",
		baseURL,
	)
	res, err := s.sonosAPIRequest(url, "GET", nil)

	if err != nil || res.StatusCode != 200 {
		log.Printf("Failed to make request to Sonos Households API, %v, %v", res.StatusCode, err)
		return err
	}

	body, _ := io.ReadAll(res.Body)
	fmt.Println(string(body))

	defer res.Body.Close()

	return nil
}

func (s *Sonos) setHousehold(householdId string) error {
	config.Get().SonosHouseholdId = householdId
	return nil
}

func (s *Sonos) updateGroupsAndPlayers() error {
	url := fmt.Sprintf(
		"%s/households/%s/groups",
		baseURL,
		config.Get().SonosHouseholdId,
	)
	res, err := s.sonosAPIRequest(url, "GET", nil)
	if err != nil {
		return fmt.Errorf("failed to make request to Sonos Groups API: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("unexpected status code %d from Sonos API", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var response GroupsResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("error decoding JSON: %w", err)
	}

	// Clear existing maps before updating
	groupPlayers = make(map[string][]string)

	// Update groups and their players
	for _, group := range response.Groups {
		groups[group.Name] = group
		groupPlayers[group.Id] = group.PlayerIds
		log.Printf("🔈 Found group: %s with %d players", group.Name, len(group.PlayerIds))
	}

	// Update players
	for _, player := range response.Players {
		players[player.Name] = player
	}

	return nil
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

func (s *Sonos) HandleControl(msg common.ControlMessage) error {
	roomMutex.Lock()
	defer roomMutex.Unlock()

	s.updateGroupsAndPlayers()

	var room Room
	if r := s.findRoom(msg.Room); r == nil {
		// room does not exist yet, creating new room
		room = Room{
			Name: msg.Room,
		}
		s.Rooms = append(s.Rooms, room)
	} else {
		room = *r
	}

	switch msg.Action {
	case "play":
		return s.Play(room)
	case "pause":
		return s.Pause(room)
	case "volume":
		if msg.Value == nil {
			return fmt.Errorf("volume action requires a value")
		}
		return s.SetVolume(*msg.Value, room)
	case "join":
		return s.JoinPlayingGroup(room)
	case "leave":
		return s.LeaveGroup(room)
	case "solo":
		return s.PlaySolo(room)
	default:
		return fmt.Errorf("unknown action: %s", msg.Action)
	}
}

func (s *Sonos) HandleSetup(msg common.SetupMessage) error {
	switch msg.Command {
	case "discover-households":
		return s.discoverHouseholds()
	case "set-household":
		return s.setHousehold(msg.Value)
	default:
		return fmt.Errorf("unknown setup command: %s", msg.Command)
	}
}

func (s *Sonos) Play(room Room) error {
	url := fmt.Sprintf(
		"%s/groups/%v/playback/play",
		baseURL,
		groupForPlayer(players[room.Name].Id),
	)

	_, err := s.sonosAPIRequest(url, "POST", nil)
	log.Print("🔈 Start playing music in room ", room.Name)
	return err
}

func (s *Sonos) Pause(room Room) error {
	url := fmt.Sprintf(
		"%s/groups/%s/playback/pause",
		baseURL,
		groupForPlayer(players[room.Name].Id),
	)

	_, err := s.sonosAPIRequest(url, "POST", nil)
	log.Print("🔈 Pause music in room ", room.Name)
	return err
}

func (s *Sonos) SetVolume(value int, room Room) error {
	url := fmt.Sprintf(
		"%s/%ss/%s/%sVolume",
		baseURL,
		s.playerOrGroup(),
		players[room.Name].Id,
		s.playerOrGroup(),
	)

	payload := strings.NewReader(fmt.Sprintf("{\"volume\":%d}", value))

	res, err := s.sonosAPIRequest(url, "POST", payload)
	if err != nil {
		return fmt.Errorf("failed to make request to Sonos API: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to set volume: %s", string(body))
	}

	log.Printf("🔈 Volume changed to %d for %s", value, room.Name)
	return nil
}

func (s *Sonos) setGroupMembers(groupId string, members []string) error {
	url := fmt.Sprintf(
		"%s/groups/%s/groups/setGroupMembers",
		baseURL,
		groupId,
	)

	payload := strings.NewReader(fmt.Sprintf(`{"playerIds": %s}`, marshalPlayerIds(members)))

	res, err := s.sonosAPIRequest(url, "POST", payload)
	if err != nil {
		return fmt.Errorf("failed to make request to Sonos API: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to set group members: %s", string(body))
	}

	return s.updateGroupsAndPlayers() // Refresh our local state
}

func (s *Sonos) JoinPlayingGroup(room Room) error {
	// Find the currently playing group
	playingGroupId, err := s.GetPlayingGroup()
	fmt.Println(playingGroupId)
	if err != nil {
		return fmt.Errorf("failed to get playing group: %w", err)
	}
	if playingGroupId == "" {
		return fmt.Errorf("no group is currently playing")
	}

	// Get current group members
	currentMembers := groupPlayers[playingGroupId]
	fmt.Println(currentMembers)

	// Add the new player to the members list if not already present
	playerToAdd := players[room.Name].Id
	for _, member := range currentMembers {
		if member == playerToAdd {
			return nil // Already in group
		}
	}
	fmt.Println(playerToAdd)
	currentMembers = append(currentMembers, playerToAdd)
	fmt.Println(currentMembers)

	if err := s.setGroupMembers(playingGroupId, currentMembers); err != nil {
		return fmt.Errorf("failed to join group: %w", err)
	}

	log.Printf("🔈 Room %s joined the playing group", room.Name)
	return nil
}

func (s *Sonos) LeaveGroup(room Room) error {
	playerId := players[room.Name].Id
	if playerId == "" {
		return fmt.Errorf("room %s not found", room.Name)
	}

	// Find which group this player is in
	currentGroupId := groupForPlayer(playerId)
	if currentGroupId == "" {
		return fmt.Errorf("room %s is not in any group", room.Name)
	}

	// Get current group members
	currentMembers := groupPlayers[currentGroupId]
	if len(currentMembers) <= 1 {
		return fmt.Errorf("room %s is the only member of its group", room.Name)
	}

	// Remove the player from the members list
	newMembers := make([]string, 0, len(currentMembers)-1)
	for _, member := range currentMembers {
		if member != playerId {
			newMembers = append(newMembers, member)
		}
	}

	if err := s.setGroupMembers(currentGroupId, newMembers); err != nil {
		return fmt.Errorf("failed to leave group: %w", err)
	}

	log.Printf("🔈 Room %s left its group", room.Name)
	return nil
}

// Helper function to marshal player IDs array to JSON
func marshalPlayerIds(playerIds []string) string {
	bytes, err := json.Marshal(playerIds)
	if err != nil {
		return "[]"
	}
	return string(bytes)
}

// Add this method to get the currently playing group
func (s *Sonos) GetPlayingGroup() (string, error) {
	// Look through the groups we already have
	for _, group := range groups {
		if group.PlaybackState == "PLAYBACK_STATE_PLAYING" {
			return group.Id, nil
		}
	}

	return "", nil // No group is currently playing
}

func (s *Sonos) PlaySolo(room Room) error {
	playerId := players[room.Name].Id
	if playerId == "" {
		return fmt.Errorf("room %s not found", room.Name)
	}

	// Find which group this player is in
	currentGroupId := groupForPlayer(playerId)
	if currentGroupId == "" {
		return fmt.Errorf("room %s is not in any group", room.Name)
	}

	// Get the current playback state before we change groups
	isPlaying := groups[room.Name].PlaybackState == "PLAYBACK_STATE_PLAYING"

	// Create a new group with just this player
	if err := s.setGroupMembers(currentGroupId, []string{playerId}); err != nil {
		return fmt.Errorf("failed to set solo group: %w", err)
	}

	// If it was playing before, ensure it continues playing
	if isPlaying {
		if err := s.Play(room); err != nil {
			return fmt.Errorf("failed to resume playback: %w", err)
		}
	}

	log.Printf("🔈 Room %s is now playing solo", room.Name)
	return nil
}
