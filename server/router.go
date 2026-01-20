package server

import (
	"net/http"

	"github.com/touka-aoi/paralle-vs-single/server/domain"
	"github.com/touka-aoi/paralle-vs-single/server/handler"
)

func Route(dispatcher domain.Dispatcher) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/ws", handler.NewAcceptHandler(dispatcher))
	return mux
}
