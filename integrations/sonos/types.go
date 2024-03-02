package sonos

type Sonos struct {
	State State
}

type State struct {
	Name    string `json:"name"`
	Value   int    `json:"value"`
	Playing bool   `json:"isPlaying"`
	Room    string `json:"room"`
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
