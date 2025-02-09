package storage

import (
	"sort"
	"sync"
)

// Memtable is a thread-safe in-memory key-value store.
// - Uses RWMutex for concurrency control.
// - Flushes to SSTable when reaching max capacity.
// - Optimized range queries with binary search.
// - Bloom Filters were considered but not included for now.
type Memtable struct {
	data       map[string]string
	mu         sync.RWMutex
	maxEntries int
	flushFunc  func(map[string]string) // Function to flush Memtable data to SSTable
}

// NewMemtable initializes a Memtable with a maximum size and a flush function.
func NewMemtable(maxEntries int, flushFunc func(map[string]string)) *Memtable {
	return &Memtable{
		data:       make(map[string]string, maxEntries),
		maxEntries: maxEntries,
		flushFunc:  flushFunc,
	}
}

// Set inserts or updates a key-value pair and triggers flush if needed.
func (m *Memtable) Set(key, value string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.data[key] = value

	if len(m.data) >= m.maxEntries {
		m.Flush()
	}
}

// flushAndReset writes data to SSTable and resets the Memtable.
func (m *Memtable) Flush() {
	if m.flushFunc != nil {
		m.flushFunc(m.data) // Flush Memtable data to SSTable
	}

	// Reset Memtable to avoid memory leaks
	m.data = make(map[string]string, m.maxEntries)
}

// Get retrieves a value for a given key.
func (m *Memtable) Get(key string) (string, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	val, exists := m.data[key]
	return val, exists
}

// Delete removes a key from the Memtable.
func (m *Memtable) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

// GetRange retrieves keys in a sorted range using binary search.
func (m *Memtable) GetRange(startKey, endKey string) map[string]string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	results := make(map[string]string)
	keys := make([]string, 0, len(m.data))

	for k := range m.data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Binary search for start position
	startIndex := sort.Search(len(keys), func(i int) bool { return keys[i] >= startKey })
	for i := startIndex; i < len(keys) && keys[i] <= endKey; i++ {
		results[keys[i]] = m.data[keys[i]]
	}

	return results
}

// Size returns the current number of keys in the Memtable.
func (m *Memtable) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}
