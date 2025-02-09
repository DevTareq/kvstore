package handler

import (
	"encoding/json"
	"errors"
	"log"
	"moniepoint/internal/storage"
	"moniepoint/internal/utils"
	"net/http"
)

// ReadHandler manages key-value retrieval.
type ReadHandler struct {
	memtable *storage.Memtable
	sstable  *storage.SSTable
}

// NewReadHandler initializes ReadHandler.
func NewReadHandler(memtable *storage.Memtable, sstable *storage.SSTable) *ReadHandler {
	return &ReadHandler{memtable, sstable}
}

// HandleRead processes an HTTP GET request for a single key.
func (rh *ReadHandler) HandleRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	key := utils.GetKeyFromPath(r.URL.Path)
	if key == "" {
		http.Error(w, "Missing key in URL", http.StatusBadRequest)
		return
	}

	value, err := rh.Read(key)
	if err != nil {
		if errors.Is(err, storage.ErrKeyNotFound) {
			http.Error(w, "Key not found", http.StatusNotFound)
			return
		}
		log.Printf("Error retrieving key '%s': %v", key, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"key": key, "value": value})
}

// HandleReadRange processes an HTTP GET request for a range of keys.
func (rh *ReadHandler) HandleReadRange(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	startKey := r.URL.Query().Get("start")
	endKey := r.URL.Query().Get("end")
	if startKey == "" || endKey == "" {
		http.Error(w, "Missing start or end key in query", http.StatusBadRequest)
		return
	}

	results, err := rh.ReadKeyRange(startKey, endKey)
	if err != nil {
		log.Printf("Error retrieving range '%s' - '%s': %v", startKey, endKey, err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// Read retrieves a key from Memtable, falling back to SSTable if needed.
func (rh *ReadHandler) Read(key string) (string, error) {
	if value, found := rh.memtable.Get(key); found {
		return value, nil
	}

	// If not found, check SSTable (persistent storage)
	value, err := rh.sstable.Read(key)
	if err != nil {
		log.Printf("SSTable Read failed for key '%s': %v", key, err)
		return "", err
	}
	return value, nil
}

// ReadKeyRange fetches keys within a range from Memtable and SSTable.
func (rh *ReadHandler) ReadKeyRange(startKey, endKey string) (map[string]string, error) {
	results := make(map[string]string)

	memResults := rh.memtable.GetRange(startKey, endKey)
	for k, v := range memResults {
		results[k] = v
	}

	// Fetch from SSTable
	sstableResults, err := rh.sstable.ReadRange(startKey, endKey)
	if err != nil {
		log.Printf("SSTable ReadRange failed for range '%s' - '%s': %v", startKey, endKey, err)
		return nil, err
	}

	// Merge results from both sources
	for k, v := range sstableResults {
		results[k] = v
	}

	return results, nil
}
