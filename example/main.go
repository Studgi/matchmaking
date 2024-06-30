package main

import (
	"net/http"

	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"github.com/thronesmc/matchmaking"
)

func main() {
	logger := logrus.New()
	cli, err := client.NewClientWithOpts()
	if err != nil {
		logger.Errorf("Failed to start client due to %v", err)
		return
	}

	defer cli.Close()

	mux := http.NewServeMux()
	mux.Handle("/", matchmaking.NewHandler(cli, nil, nil, logger))
	if err := http.ListenAndServe(":8080", mux); err != nil {
		logger.Errorf("Failed to serve http due to %v", err)
		return
	}
}
