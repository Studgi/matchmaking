package matchmaking

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/thronesmc/matchmaking/internal"
	"github.com/thronesmc/matchmaking/server"
)

type Handler struct {
	cli              *client.Client
	hostConfig       *container.HostConfig
	networkingConfig *network.NetworkingConfig
	logger           internal.Logger
	registry         *server.Registry
}

func (h *Handler) find(response http.ResponseWriter, request *http.Request) {
	resp := map[string]interface{}{}
	status := http.StatusInternalServerError

	defer func() {
		response.WriteHeader(status)
		_ = json.NewEncoder(response).Encode(resp)
	}()

	serverType, err := strconv.Atoi(request.URL.Query().Get("type"))
	if err != nil {
		resp["message"] = "type not set."
		status = http.StatusBadRequest
		return
	}

	players, err := strconv.Atoi(request.URL.Query().Get("players"))
	if err != nil {
		resp["message"] = "players not set."
		status = http.StatusBadRequest
		return
	}

	var matches []server.Server
	for _, s := range h.registry.GetServers() {
		if s.Type != serverType || s.State != server.StateFree || s.PlayersMax-len(s.Players) < players {
			continue
		}

		matches = append(matches, *s)
	}

	resp["message"] = "OK."
	resp["matches"] = matches
	status = http.StatusOK
}

func (h *Handler) get(response http.ResponseWriter, request *http.Request) {
	resp := map[string]interface{}{}
	status := http.StatusInternalServerError

	defer func() {
		response.WriteHeader(status)
		_ = json.NewEncoder(response).Encode(resp)
	}()

	identifier := request.URL.Query().Get("identifier")
	if identifier == "" {
		resp["message"] = "identifier not set."
		status = http.StatusBadRequest
		return
	}

	s := h.registry.GetServer(identifier)
	if s == nil {
		resp["message"] = "server not found."
		status = http.StatusNotFound
		return
	}

	resp["message"] = "OK."
	resp["data"] = *s
	status = http.StatusOK
}

func (h *Handler) patch(response http.ResponseWriter, request *http.Request) {
	resp := map[string]interface{}{}
	status := http.StatusInternalServerError

	defer func() {
		response.WriteHeader(status)
		_ = json.NewEncoder(response).Encode(resp)
	}()

	identifier := request.URL.Query().Get("identifier")
	if identifier == "" {
		resp["message"] = "identifier not set."
		status = http.StatusBadRequest
		return
	}

	s := h.registry.GetServer(identifier)
	if s == nil {
		resp["message"] = "server not found."
		status = http.StatusNotFound
		return
	}

	players := request.URL.Query().Get("players")
	if players != "" {
		if err := json.Unmarshal([]byte(players), &s.Players); err != nil {
			resp["message"] = "players invalid."
			status = http.StatusBadRequest
			return
		}
	}

	playersMax := request.URL.Query().Get("playersMax")
	if playersMax != "" {
		playersMaxInt, err := strconv.Atoi(playersMax)
		if err != nil {
			resp["message"] = "players invalid."
			status = http.StatusBadRequest
			return
		}
		s.PlayersMax = playersMaxInt
	}

	state := request.URL.Query().Get("state")
	if state != "" {
		stateInt, err := strconv.Atoi(state)
		if err != nil {
			resp["message"] = "state invalid."
			status = http.StatusBadRequest
			return
		}
		s.State = stateInt
	}

	extras := request.URL.Query().Get("extras")
	if extras != "" {
		if err := json.Unmarshal([]byte(extras), &s.Extras); err != nil {
			resp["message"] = "extras invalid."
			status = http.StatusBadRequest
			return
		}
	}

	resp["message"] = "OK."
	status = http.StatusOK
}

func (h *Handler) post(response http.ResponseWriter, request *http.Request) {
	resp := map[string]interface{}{}
	status := http.StatusInternalServerError

	defer func() {
		response.WriteHeader(status)
		_ = json.NewEncoder(response).Encode(resp)
	}()

	image := request.URL.Query().Get("image")
	if image == "" {
		resp["message"] = "image not set."
		status = http.StatusBadRequest
		return
	}

	serverType, err := strconv.Atoi(request.URL.Query().Get("type"))
	if err != nil {
		resp["message"] = "type not set."
		status = http.StatusBadRequest
		return
	}

	c, err := h.cli.ContainerCreate(
		context.Background(),
		&container.Config{
			Image: image,
			Env: []string{
				"SERVER_CONFIGURATION=" + request.URL.Query().Get("configuration"),
				"SERVER_IMAGE=" + image,
				"SERVER_TYPE=" + request.URL.Query().Get("type"),
			},
		},
		h.hostConfig,
		h.networkingConfig,
		nil,
		"",
	)
	if err != nil {
		resp["message"] = "failed to find container."
		h.logger.Errorf("Failed to find container due to %v", err)
		return
	}

	if err := h.cli.ContainerStart(context.Background(), c.ID, container.StartOptions{}); err != nil {
		resp["message"] = "failed to start container."
		h.logger.Errorf("Failed to start container due to %v", err)
		return
	}

	inspection, err := h.cli.ContainerInspect(context.Background(), c.ID)
	if err != nil {
		resp["message"] = "failed to inspect container."
		h.logger.Errorf("Failed to inspect container due to %v", err)
		return
	}

	s := &server.Server{
		Identifier: c.ID[:12],
		Image:      image,

		PlayersMax: 0,
		State:      server.StateStartup,
		Type:       serverType,

		Container: *inspection.ContainerJSONBase,
		Network:   *inspection.NetworkSettings,

		Extras:  map[string]interface{}{},
		Players: []server.Player{},
	}
	h.registry.AddServer(s.Identifier, s)

	resp["message"] = "instance created"
	resp["data"] = *s
	status = http.StatusCreated
}

func (h *Handler) delete(response http.ResponseWriter, request *http.Request) {
	resp := map[string]interface{}{}
	status := http.StatusInternalServerError

	defer func() {
		response.WriteHeader(status)
		_ = json.NewEncoder(response).Encode(resp)
	}()

	identifier := request.URL.Query().Get("identifier")
	if identifier == "" {
		resp["message"] = "identifier not set."
		status = http.StatusBadRequest
		return
	}

	h.registry.RemoveServer(identifier)
	resp["message"] = "OK."
	status = http.StatusOK
}

func (h *Handler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	switch request.Method {
	case http.MethodGet:
		if request.URL.Path == "/find" {
			h.find(response, request)
			return
		}
		h.get(response, request)
	case http.MethodPatch:
		h.patch(response, request)
	case http.MethodPost:
		h.post(response, request)
	case http.MethodDelete:
		h.delete(response, request)
	}
}

func NewHandler(client *client.Client, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, logger internal.Logger) *Handler {
	return &Handler{
		cli:              client,
		hostConfig:       hostConfig,
		networkingConfig: networkingConfig,
		logger:           logger,
		registry:         server.NewRegistry(),
	}
}
