package hue

import (
	"cuore/config"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Hue struct {
	State State
}

type State struct {
	Name    string `json:"name"`
	Value   int    `json:"value"`
	Playing bool   `json:"isPlaying"`
}

var (
	authToken = config.Get().HueAuthToken
)

// kitchen := []int{22, 23, 25}
// bedroom := []int{6, 4}
// living_room = []int{2, 3}
var space = []int{7, 19}

func (h Hue) Switch(state bool) {
	for _, light := range space {
		h.setLightState(state, light)
	}
}

func (h Hue) Setup()        {}
func (h Hue) Autodiscover() {}

// TODO: All the discovery stuff
// TODO: Authorisation

func (h Hue) setLightState(state bool, lightId int) {
	url := fmt.Sprintf("http://192.168.178.34/api/%s/lights/%d/state", authToken, lightId)
	payload := strings.NewReader(fmt.Sprintf("{\"on\": %t}", state))
	req, _ := http.NewRequest("PUT", url, payload)

	req.Header.Add("accept", "application/json")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Authorization", authToken)

	res, _ := http.DefaultClient.Do(req)
	b, _ := io.ReadAll(res.Body)

	fmt.Println(string(b))

	defer res.Body.Close()
}

func (h Hue) Play(room string) {
	for _, light := range space {
		h.setLightState(true, light)
	}
}

func (h Hue) Pause(room string) {
	for _, light := range space {
		h.setLightState(false, light)
	}
}

func (h Hue) LongPress(room string) {
	for _, light := range space {
		h.setLightState(false, light)
	}
}
