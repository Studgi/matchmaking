package server

import "github.com/docker/docker/api/types"

const (
	StateStartup = iota
	StateFree
	StateBusy
	StateShutdown
)

type Player struct {
	Username string `json:"username"`
	XUID     string `json:"xuid"`
}

type Server struct {
	Identifier string `json:"identifier"`
	Image      string `json:"image"`

	PlayersMax int `json:"players_max"`
	State      int `json:"state"`
	Type       int `json:"type"`

	Container types.ContainerJSONBase `json:"container"`
	Network   types.NetworkSettings   `json:"network"`

	Extras  map[string]interface{} `json:"extras"`
	Players []Player               `json:"players"`
}
