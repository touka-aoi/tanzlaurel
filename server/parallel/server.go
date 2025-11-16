package parallel

import "net/http"

func NewServer(addr string, wsHandler http.Handler) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("GET /ws", wsHandler)
	return &http.Server{
		Addr:    addr,
		Handler: mux,
	}
}
