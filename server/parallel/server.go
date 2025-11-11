package parallel

import (
	"net/http"

	parallelhandler "github.com/touka-aoi/paralle-vs-single/handler/parallel"
)

func NewServer(addr string, handler *parallelhandler.Handler) *http.Server {
	mux := newMux(handler)
	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}

func newMux(handler *parallelhandler.Handler) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("POST /move", handler.HandleMove)
	mux.HandleFunc("POST /buff", handler.HandleBuff)
	mux.HandleFunc("POST /attack", handler.HandleAttack)
	mux.HandleFunc("POST /trade", handler.HandleTrade)

	return mux
}
