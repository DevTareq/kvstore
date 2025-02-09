package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"moniepoint/internal/api"
	"moniepoint/internal/handler"
	"moniepoint/internal/middleware"
	"moniepoint/internal/storage"
	"moniepoint/pkg/config"
)

func main() {
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		log.Fatalf("[ERROR] Failed to load config: %v", err)
	}

	wal, err := storage.NewWAL()
	if err != nil {
		log.Fatalf("[ERROR] Failed to initialize WAL: %v", err)
	}
	defer wal.Close()

	sstable, err := storage.NewSSTable(cfg.SSTablePath)
	if err != nil {
		log.Fatalf("[ERROR] Failed to initialize SSTable: %v", err)
	}

	// Initialize Memtable (Flushes to SSTable when full)
	memtable := storage.NewMemtable(cfg.MemtableMaxEntries, func(data map[string]string) {
		for key, value := range data {
			if err := sstable.Write(key, value); err != nil {
				log.Printf("[ERROR] Failed to flush Memtable to SSTable: %v", err)
			}
		}
		wal.Flush()
	})

	// WAL Recovery: Restore Data to Memtable After a Crash
	restoredData, err := wal.Replay()
	if err != nil {
		log.Fatalf("[ERROR] WAL replay failed: %v", err)
	}
	for key, value := range restoredData {
		memtable.Set(key, value) // Reinserting WAL entries into Memtable
	}
	log.Printf("[INFO] WAL replay restored %d entries to Memtable", len(restoredData))

	// Initialize Handlers
	writeHandler := handler.NewWriteHandler(wal, memtable, sstable)
	readHandler := handler.NewReadHandler(memtable, sstable)
	deleteHandler := handler.NewDeleteHandler(memtable, sstable)

	requestHandler := handler.NewRequestHandler(readHandler, writeHandler, deleteHandler)

	router := api.NewRouter(requestHandler)

	rateLimiter := middleware.NewRateLimiter(10, 5*time.Second)

	finalHandler := rateLimiter.LimitMiddleware(router)

	serverAddr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	log.Printf("[INFO] Server is starting on %s 🚀", serverAddr)
	if err = http.ListenAndServe(serverAddr, finalHandler); err != nil {
		log.Fatalf("[ERROR] Server failed: %s", err)
	}
}
