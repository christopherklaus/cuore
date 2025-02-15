package hue

import (
	"cuore/common"
	"cuore/config"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

var (
	groups = map[string]string{} // room -> groupId
)

func (h *Hue) HandleControl(msg common.ControlMessage) error {
	if err := h.updateGroups(); err != nil {
		return fmt.Errorf("failed to update groups: %w", err)
	}

	switch msg.Action {
	case "on":
		return h.setGroupStatus(msg.Room, true)
	case "off":
		return h.setGroupStatus(msg.Room, false)
	case "brightness":
		if msg.Value == nil {
			return fmt.Errorf("brightness action requires a value")
		}
		return h.setGroupBrightness(msg.Room, *msg.Value)
	default:
		return fmt.Errorf("unknown action: %s", msg.Action)
	}
}

func (h *Hue) HandleSetup(msg common.SetupMessage) error {
	switch msg.Command {
	case "discover":
		return h.Autodiscover()
	case "authorize":
		// TODO: Implement authorization
		return fmt.Errorf("authorization not implemented")
	default:
		return fmt.Errorf("unknown setup command: %s", msg.Command)
	}
}

func (h *Hue) updateGroups() error {
	res, err := hueAPIRequest("groups", "GET", nil)
	if err != nil {
		return fmt.Errorf("failed to make request to Hue API: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	var groupsResponse map[string]GroupResponse
	if err := json.Unmarshal(body, &groupsResponse); err != nil {
		return fmt.Errorf("error decoding JSON: %w", err)
	}

	for id, group := range groupsResponse {
		groups[group.Name] = id
	}

	return nil
}

func hueAPIRequest(url string, method string, payload io.Reader) (*http.Response, error) {
	fullUrl := fmt.Sprintf("http://%s/api/%s/%s", config.Get().HueBridgeIP, authenticationToken(), url)
	req, err := http.NewRequest(method, fullUrl, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Authorization", authenticationToken())

	return http.DefaultClient.Do(req)
}

func authenticationToken() string {
	// token, _ := getToken()
	// log.Print("Hue Token: ", token.AccessToken)
	return config.Get().HueAuthToken
}

func (h *Hue) Autodiscover() error {
	// TODO: Implement autodiscovery
	log.Print("Autodiscovery not implemented")
	return nil
}

func (h *Hue) setGroupStatus(room string, state bool) error {
	groupId, ok := groups[room]
	if !ok {
		return fmt.Errorf("room %s not found", room)
	}

	url := fmt.Sprintf("groups/%s/action", groupId)
	payload := strings.NewReader(fmt.Sprintf("{\"on\": %t}", state))

	res, err := hueAPIRequest(url, "PUT", payload)
	if err != nil {
		return fmt.Errorf("failed to make request to Hue API: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to set group status: %s", string(body))
	}

	return nil
}

func (h *Hue) setGroupBrightness(room string, value int) error {
	groupId, ok := groups[room]
	if !ok {
		return fmt.Errorf("room %s not found", room)
	}

	url := fmt.Sprintf("groups/%s/action", groupId)
	hueValue := int((float64(value) / 100) * 254)
	payload := strings.NewReader(fmt.Sprintf("{\"bri\": %v}", hueValue))

	res, err := hueAPIRequest(url, "PUT", payload)
	if err != nil {
		return fmt.Errorf("failed to make request to Hue API: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("failed to set group brightness: %s", string(body))
	}

	return nil
}
