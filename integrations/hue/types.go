package hue

type GroupResponse struct {
	Name   string
	Lights []string
}

type Hue struct {
	State State
}

type State struct {
	Name    string `json:"name"`
	Value   int    `json:"value"`
	Playing bool   `json:"isPlaying"`
}
