package parallel

import (
	"net/http"

	parallelhandler "github.com/touka-aoi/paralle-vs-single/handler/parallel"
)

func NewMux(handler *parallelhandler.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /move", handler.HandleMove)
	mux.HandleFunc("POST /buff", handler.HandleBuff)
	mux.HandleFunc("POST /attack", handler.HandleAttack)
	mux.HandleFunc("POST /trade", handler.HandleTrade)

	return mux
}
