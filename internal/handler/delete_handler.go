package handler

import (
	"log"
	"net/http"

	"moniepoint/internal/storage"
	"moniepoint/internal/utils"
)

// DeleteHandler handles key deletion.
type DeleteHandler struct {
	memtable *storage.Memtable
	sstable  *storage.SSTable
}

// NewDeleteHandler initializes DeleteHandler.
func NewDeleteHandler(memtable *storage.Memtable, sstable *storage.SSTable) *DeleteHandler {
	return &DeleteHandler{memtable, sstable}
}

// HandleDelete processes an HTTP DELETE request.
func (dh *DeleteHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	key := utils.GetKeyFromPath(r.URL.Path)
	if key == "" {
		http.Error(w, "Missing key in URL", http.StatusBadRequest)
		return
	}

	if err := dh.Delete(key); err != nil {
		http.Error(w, "Failed to delete key", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Delete removes a key from both Memtable and SSTable.
func (dh *DeleteHandler) Delete(key string) error {
	dh.memtable.Delete(key)
	if err := dh.sstable.Delete(key); err != nil {
		log.Printf("Failed to delete key=%s from SSTable: %v", key, err)
		return err
	}
	return nil
}
