package common

type Page struct {
	Name         string `json:"name"`
	CurrentValue int    `json:"currentValue"`
	Status       bool   `json:"status"`
	Room         string `json:"room"`
}
