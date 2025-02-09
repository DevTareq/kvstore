package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"moniepoint/internal/storage"
	"moniepoint/internal/utils"
)

type WriteHandler struct {
	wal      *storage.WAL
	memtable *storage.Memtable
	sstable  *storage.SSTable
}

type BatchWriteRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// NewWriteHandler initializes WriteHandler.
func NewWriteHandler(wal *storage.WAL, memtable *storage.Memtable, sstable *storage.SSTable) *WriteHandler {
	return &WriteHandler{
		wal:      wal,
		memtable: memtable,
		sstable:  sstable,
	}
}

// HandleWrite processes an HTTP POST request for writing a value.
func (wh *WriteHandler) HandleWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	key := utils.GetKeyFromPath(r.URL.Path)
	if key == "" {
		http.Error(w, "Missing key in URL", http.StatusBadRequest)
		return
	}

	var req struct {
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	// Step 1: Append to WAL (Durability)
	if err := wh.wal.Append(key, req.Value); err != nil {
		log.Printf("[ERROR] WAL write failed for key=%s: %v", key, err)
		http.Error(w, "WAL Write Failed", http.StatusInternalServerError)
		return
	}

	// Step 2: Ensure WAL is flushed before updating Memtable & SSTable
	wh.wal.Flush()

	// Step 3: Store in Memtable (Fast Read Access)
	wh.memtable.Set(key, req.Value)

	// Step 4: Store in SSTable (Persistent Storage)
	if err := wh.sstable.Write(key, req.Value); err != nil {
		log.Printf("[ERROR] SSTable write failed for key=%s: %v", key, err)
		http.Error(w, "SSTable Write Failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// HandleBatchWrite processes an HTTP POST request for batch writes.
func (wh *WriteHandler) HandleBatchWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var batchReq []BatchWriteRequest
	if err := json.NewDecoder(r.Body).Decode(&batchReq); err != nil {
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	// Step 1: Prepare batch for WAL writing
	var walEntries []struct{ Key, Value string }
	for _, entry := range batchReq {
		walEntries = append(walEntries, struct{ Key, Value string }{
			Key:   entry.Key,
			Value: entry.Value,
		})
	}

	// Step 2: Write each entry to WAL
	for _, entry := range walEntries {
		if err := wh.wal.Append(entry.Key, entry.Value); err != nil {
			log.Printf("[ERROR] WAL batch write failed for key=%s: %v", entry.Key, err)
			http.Error(w, "Batch WAL Write Failed", http.StatusInternalServerError)
			return
		}
	}

	// Step 3: Ensure WAL is flushed before updating Memtable & SSTable
	wh.wal.Flush()

	// Step 4: Store batch in Memtable & SSTable concurrently
	var wg sync.WaitGroup

	for _, entry := range walEntries {
		wg.Add(1)
		go func(entry struct{ Key, Value string }) {
			defer wg.Done()
			wh.memtable.Set(entry.Key, entry.Value)
			if err := wh.sstable.Write(entry.Key, entry.Value); err != nil {
				log.Printf("[ERROR] SSTable batch write failed for key=%s: %v", entry.Key, err)
			}
		}(entry)
	}

	wg.Wait()

	w.WriteHeader(http.StatusCreated)
}
