package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var ErrKeyNotFound = errors.New("key not found")

// SSTable represents a persistent key-value store with an index for fast lookups.
type SSTable struct {
	file       *os.File
	indexPath  string
	mu         sync.RWMutex
	index      map[string]int64
	bufferedRd *bufio.Reader // Buffered Reader for faster reads
	writeBuf   *bufio.Writer // Buffered Writer to batch writes
}

// NewSSTable initializes an SSTable with buffered I/O
func NewSSTable(filePath string) (*SSTable, error) {
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	s := &SSTable{
		file:       file,
		indexPath:  filePath + ".index",
		index:      make(map[string]int64),
		bufferedRd: bufio.NewReader(file),
		writeBuf:   bufio.NewWriter(file),
	}

	if err := s.loadIndex(); err != nil {
		return nil, err
	}

	return s, nil
}

// Write with Immediate Flush & Sync
func (s *SSTable) Write(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	offset, err := s.file.Seek(0, os.SEEK_END)
	if err != nil {
		return err
	}

	entry := map[string]string{"key": key, "value": value}
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	_, err = s.writeBuf.Write(append(jsonData, '\n'))
	if err != nil {
		return err
	}

	err = s.writeBuf.Flush()
	if err != nil {
		return err
	}

	err = s.file.Sync()
	if err != nil {
		return err
	}

	s.index[key] = offset
	return nil
}

// Read using Buffered Reader
func (s *SSTable) Read(key string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	offset, exists := s.index[key]
	if !exists {
		return "", ErrKeyNotFound
	}

	_, err := s.file.Seek(offset, os.SEEK_SET)
	if err != nil {
		return "", err
	}

	s.bufferedRd = bufio.NewReader(s.file)

	line, err := s.bufferedRd.ReadString('\n') // Buffered Read
	if err != nil {
		return "", err
	}

	var entry map[string]string
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &entry); err != nil {
		return "", err
	}

	return entry["value"], nil
}

// ReadRange with Binary Search
func (s *SSTable) ReadRange(startKey, endKey string) (map[string]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	results := make(map[string]string)
	keys := make([]string, 0, len(s.index))

	for k := range s.index {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// binary search to find start and end positions
	startIdx := sort.SearchStrings(keys, startKey)
	endIdx := sort.SearchStrings(keys, endKey)

	if startIdx >= len(keys) || endIdx >= len(keys) || startIdx > endIdx {
		return nil, nil
	}

	for _, key := range keys[startIdx : endIdx+1] {
		value, err := s.Read(key)
		if err == nil {
			results[key] = value
		}
	}

	return results, nil
}

// Delete Function (No Deadlock)
func (s *SSTable) Delete(key string) error {
	s.mu.Lock()
	_, exists := s.index[key]
	if !exists {
		s.mu.Unlock()
		return ErrKeyNotFound
	}

	delete(s.index, key)
	s.mu.Unlock()

	return s.saveIndex()
}

// Save index to disk safely
func (s *SSTable) saveIndex() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Create(s.indexPath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for key, offset := range s.index {
		_, err := writer.WriteString(fmt.Sprintf("%s:%d\n", key, offset))
		if err != nil {
			return err
		}
	}
	return writer.Flush()
}

// Load index from disk on startup
func (s *SSTable) loadIndex() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	file, err := os.Open(s.indexPath)
	if os.IsNotExist(err) {
		return nil // No index file yet
	} else if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ":")
		if len(parts) != 2 {
			continue
		}
		offset, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			continue
		}
		s.index[parts[0]] = offset
	}

	return scanner.Err()
}

func (s *SSTable) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.file != nil {
		err := s.file.Close()
		s.file = nil
		return err
	}
	return nil
}
