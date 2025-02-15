package sonos

type Sonos struct {
	Rooms          []Room
	ControlPlayers bool
}

type Room struct {
	Name      string
	State     *State
	PlayerIds []string
}

type State struct {
	Value   int  `json:"value"`
	Playing bool `json:"isPlaying"`
}
