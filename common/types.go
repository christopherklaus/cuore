package common

type ControlMessage struct {
	Target string `json:"target"` // e.g. "music", "light"
	Room   string `json:"room"`
	Action string `json:"action"`          // e.g. "play", "pause", "volume"
	Value  *int   `json:"value,omitempty"` // optional, used for volume
}

type SetupMessage struct {
	Target  string `json:"target"`          // e.g. "music", "light"
	Command string `json:"command"`         // e.g. "discover", "authorize"
	Value   string `json:"value,omitempty"` // optional value for the command
}
