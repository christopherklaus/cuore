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

func (h *Hue) updateGroups() {
	res, _ := hueAPIRequest("groups", "GET", nil)
	body, _ := io.ReadAll(res.Body)

	var groupsResponse map[string]GroupResponse

	if err := json.Unmarshal(body, &groupsResponse); err != nil {
		log.Print("Error decoding JSON: ", err)
		return
	}

	for id, group := range groupsResponse {
		groups[group.Name] = id
	}

	defer res.Body.Close()
}

func (h *Hue) UpdateState(state common.Page) {
	h.updateGroups() // should happen on some cycle
	setGroupBrightness(state.Room, state.CurrentValue)
	setGroupStatus(state.Room, state.Status)
}

func hueAPIRequest(url string, method string, payload io.Reader) (*http.Response, error) {
	fullUrl := fmt.Sprintf("http://%s/api/%s/%s", config.Get().HueBridgeIP, authenticationToken(), url)
	req, _ := http.NewRequest(method, fullUrl, payload)
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

func (h *Hue) Setup()        {}
func (h *Hue) Autodiscover() {}

// TODO: All the discovery stuff
// TODO: Authorisation

func setGroupStatus(room string, state bool) {
	url := fmt.Sprintf("groups/%s/action", groups[room])
	payload := strings.NewReader(fmt.Sprintf("{\"on\": %t}", state))

	res, _ := hueAPIRequest(url, "PUT", payload)

	defer res.Body.Close()
}

func setGroupBrightness(room string, value int) {
	url := fmt.Sprintf("groups/%s/action", groups[room])
	hueValue := int((float64(value) / 100) * 254)
	payload := strings.NewReader(fmt.Sprintf("{\"bri\": %v}", hueValue))

	res, _ := hueAPIRequest(url, "PUT", payload)

	defer res.Body.Close()
}
