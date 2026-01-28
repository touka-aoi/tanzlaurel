package server

import (
	"net/http"

	"withered/server/domain"
	"withered/server/handler"
)

func Route(pubsub domain.PubSub, roomManager domain.RoomManager) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/ws", handler.NewAcceptHandler(pubsub, roomManager))
	return mux
}
