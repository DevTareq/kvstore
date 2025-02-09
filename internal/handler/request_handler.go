package handler

import "net/http"

// RequestHandler routes API requests to ReadHandler and WriteHandler.
type RequestHandler struct {
	readHandler   *ReadHandler
	writeHandler  *WriteHandler
	deleteHandler *DeleteHandler
}

func NewRequestHandler(readHandler *ReadHandler, writeHandler *WriteHandler, deleteHandler *DeleteHandler) *RequestHandler {
	return &RequestHandler{readHandler, writeHandler, deleteHandler}
}

// HandleWrite delegates the write request.
func (h *RequestHandler) HandleWrite(w http.ResponseWriter, r *http.Request) {
	h.writeHandler.HandleWrite(w, r)
}

// HandleBatchWrite delegates batch write requests.
func (h *RequestHandler) HandleBatchWrite(w http.ResponseWriter, r *http.Request) {
	h.writeHandler.HandleBatchWrite(w, r)
}

// HandleRead delegates the read request.
func (h *RequestHandler) HandleRead(w http.ResponseWriter, r *http.Request) {
	h.readHandler.HandleRead(w, r)
}

// HandleReadRange delegates range queries.
func (h *RequestHandler) HandleReadRange(w http.ResponseWriter, r *http.Request) {
	h.readHandler.HandleReadRange(w, r)
}

// HandleDelete delegates the delete request.
func (h *RequestHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	h.deleteHandler.HandleDelete(w, r)
}
