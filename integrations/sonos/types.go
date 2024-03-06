package sonos

type Sonos struct {
	Rooms          []Room
	ControlPlayers bool
}

type Room struct {
	Name      string
	State     State
	PlayerIds []string
}

type State struct {
	Value   int  `json:"value"`
	Playing bool `json:"isPlaying"`
}

type GroupsResponse struct {
	Groups  []Group  `json:"groups"`
	Players []Player `json:"players"`
}

type Group struct {
	Id            string   `json:"id"`
	Name          string   `json:"name"`
	CoordinatorId string   `json:"coordinatorId"`
	State         string   `json:"playbackState"`
	PlayerIds     []string `json:"playerIds"`
}
type Player struct {
	Id           string   `json:"id"`
	Name         string   `json:"name"`
	WebsocketUrl string   `json:"websocketUrl"`
	Capabilities []string `json:"capabilities"`
}
