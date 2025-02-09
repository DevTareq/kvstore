package api

import (
	"moniepoint/internal/handler"
	"net/http"
)

func NewRouter(requestHandler *handler.RequestHandler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/kv/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			requestHandler.HandleWrite(w, r)
		case http.MethodGet:
			if r.URL.Query().Has("start") && r.URL.Query().Has("end") {
				requestHandler.HandleReadRange(w, r)
			} else {
				requestHandler.HandleRead(w, r)
			}
		case http.MethodDelete:
			requestHandler.HandleDelete(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/kv/batch", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			requestHandler.HandleBatchWrite(w, r)
		} else {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	})

	return mux
}
