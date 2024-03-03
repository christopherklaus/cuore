package common

type Page struct {
	Name         string `json:"name"`
	CurrentValue int    `json:"currentValue"`
	State        bool   `json:"state"`
	Room         string `json:"room"`
}
