package storage

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	WALFlushInterval  = 500 * time.Millisecond // Flush interval for WAL (every 500ms)
	WALMaxSize        = 10 * 1024 * 1024       // Rotate WAL when it exceeds 10MB
	WALDirectory      = "data/wal/"
	WALRetentionCount = 5           // Keep the last 5 WAL files
	BufferSize        = 64 * 1024   // 64 KB buffer
	MaxBufferSize     = 1024 * 1024 // Max 1MB buffer
)

type WAL struct {
	mu        sync.Mutex
	file      *os.File
	writer    *bufio.Writer
	logQueue  chan string
	wg        sync.WaitGroup
	closeChan chan struct{}
}

// NewWAL initializes a new Write-Ahead Log with asynchronous writes and rotation.
func NewWAL() (*WAL, error) {
	if err := os.MkdirAll(WALDirectory, 0755); err != nil {
		return nil, err
	}

	// Get the latest WAL file or create a new one.
	filePath := getLatestWALFile()
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	wal := &WAL{
		file:      file,
		writer:    bufio.NewWriterSize(file, BufferSize),
		logQueue:  make(chan string, 1000),
		closeChan: make(chan struct{}),
	}

	wal.wg.Add(1)
	go wal.processQueue()

	return wal, nil
}

// Append queues a log entry for asynchronous writing.
func (w *WAL) Append(key, value string) error {
	if key == "" || value == "" {
		return fmt.Errorf("invalid WAL entry: empty key or value")
	}

	entry := fmt.Sprintf("%s:%s\n", key, value)

	if os.Getenv("TEST_MODE") == "true" {
		w.syncWrite(entry)
		return nil
	}

	select {
	case w.logQueue <- entry:
	default:
		w.syncWrite(entry)
	}
	return nil
}

// processQueue handles asynchronous writes and periodic flushing.
func (w *WAL) processQueue() {
	defer w.wg.Done()

	ticker := time.NewTicker(WALFlushInterval)
	defer ticker.Stop()

	for {
		select {
		case entry := <-w.logQueue:
			w.syncWrite(entry)
		case <-ticker.C:
			w.Flush()
		case <-w.closeChan:
			w.Flush()
			return
		}
	}
}

// syncWrite writes directly to the WAL with mutex protection.
func (w *WAL) syncWrite(entry string) {
	w.mu.Lock()
	defer w.mu.Unlock()

	_, err := w.writer.WriteString(entry)
	if err != nil {
		log.Printf("[ERROR] WAL write failed: %v", err)
	}

	w.checkRotation()
}

// checkRotation rotates WAL logs when the file exceeds the maximum size.
func (w *WAL) checkRotation() {
	info, err := w.file.Stat()
	if err != nil {
		log.Printf("[ERROR] Failed to get WAL file size: %v", err)
		return
	}

	if info.Size() > WALMaxSize {
		w.writer.Flush()
		w.file.Close()

		newFilePath := filepath.Join(WALDirectory, fmt.Sprintf("wal_%d.log", time.Now().Unix()))
		newFile, err := os.OpenFile(newFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			log.Printf("[ERROR] Failed to create new WAL file: %v", err)
			return
		}

		w.file = newFile
		w.writer = bufio.NewWriterSize(newFile, BufferSize)

		w.cleanupOldWALs()
	}
}

// cleanupOldWALs removes older WAL logs, keeping only the latest WALRetentionCount files.
func (w *WAL) cleanupOldWALs() {
	files, err := filepath.Glob(filepath.Join(WALDirectory, "wal_*.log"))
	if err != nil {
		log.Printf("[ERROR] Failed to list WAL files: %v", err)
		return
	}

	if len(files) > WALRetentionCount {
		for _, file := range files[:len(files)-WALRetentionCount] {
			os.Remove(file)
		}
	}
}

// Flush forces the buffered WAL data to be written to disk.
func (w *WAL) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.writer.Flush()
	w.file.Sync()
}

// Close gracefully shuts down the WAL by stopping the async goroutine, flushing, and closing the file.
func (w *WAL) Close() {
	close(w.closeChan)
	w.wg.Wait()
	w.Flush()
	w.file.Close()
}

// Replay reads the WAL logs and reconstructs the state.
func (w *WAL) Replay() (map[string]string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	data := make(map[string]string)
	file, err := os.Open(w.file.Name())
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, BufferSize), MaxBufferSize)

	batchSize := 10000
	tempBatch := make([]string, 0, batchSize)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		tempBatch = append(tempBatch, line)

		if len(tempBatch) >= batchSize {
			processBatch(tempBatch, &data)
			tempBatch = tempBatch[:0]
		}
	}

	if len(tempBatch) > 0 {
		processBatch(tempBatch, &data)
	}

	return data, scanner.Err()
}

// processBatch parses a batch of WAL lines and updates the provided data map.
func processBatch(batch []string, data *map[string]string) {
	for _, line := range batch {
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			log.Printf("[WARN] Skipping malformed WAL entry: %s", line)
			continue
		}
		(*data)[parts[0]] = parts[1]
	}
}

// getLatestWALFile finds the most recent WAL file or returns a default file path.
func getLatestWALFile() string {
	files, err := filepath.Glob(filepath.Join(WALDirectory, "wal_*.log"))
	if err != nil || len(files) == 0 {
		return filepath.Join(WALDirectory, "wal_1.log")
	}

	return files[len(files)-1]
}
