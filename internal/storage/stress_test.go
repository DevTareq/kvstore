package storage_test

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"moniepoint/internal/storage"
)

// / aggregateWALEntries scans all WAL files in the WAL directory and returns a map of all entries.
func aggregateWALEntries() (map[string]string, error) {
	entries := make(map[string]string)

	files, err := filepath.Glob(filepath.Join(storage.WALDirectory, "wal_*.log"))
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		files = append(files, filepath.Join(storage.WALDirectory, "wal_1.log"))
	}

	for _, file := range files {
		f, err := os.Open(file)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue // Skip malformed entries.
			}
			entries[parts[0]] = parts[1]
		}
		f.Close()
	}

	return entries, nil
}

func TestWALStress(t *testing.T) {
	wal, err := storage.NewWAL()
	if err != nil {
		t.Fatalf("Failed to initialize WAL: %v", err)
	}
	defer wal.Close()

	numGoroutines := 100
	numOpsPerGoroutine := 10000
	totalExpected := numGoroutines * numOpsPerGoroutine

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				key := fmt.Sprintf("stress_%d_%d", gid, j)
				if err := wal.Append(key, "stress_value"); err != nil {
					t.Errorf("Append error in goroutine %d: %v", gid, err)
				}
			}
		}(i)
	}

	wg.Wait()

	wal.Flush()

	time.Sleep(2 * time.Second)

	entries, err := aggregateWALEntries()
	if err != nil {
		t.Fatalf("Failed to aggregate WAL entries: %v", err)
	}

	if len(entries) != totalExpected {
		t.Fatalf("Expected %d entries in WAL, but got %d", totalExpected, len(entries))
	}

	t.Logf("WAL Stress Test passed: %d entries logged and aggregated successfully", totalExpected)
}

// TestMemtableStress runs a stress test on the Memtable by concurrently writing many key-value pairs.
// After all writes complete, it samples keys from each goroutine to verify that the expected value is returned.
func TestMemtableStress(t *testing.T) {
	numGoroutines := 50
	numOpsPerGoroutine := 20000
	totalExpected := numGoroutines * numOpsPerGoroutine

	memtable := storage.NewMemtable(totalExpected*2, nil)

	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for j := 0; j < numOpsPerGoroutine; j++ {
				key := fmt.Sprintf("stress_mem_%d_%d", gid, j)
				memtable.Set(key, "value")
			}
		}(i)
	}

	wg.Wait()

	for i := 0; i < numGoroutines; i++ {
		sampleIndexes := []int{0, numOpsPerGoroutine / 2, numOpsPerGoroutine - 1}
		for _, j := range sampleIndexes {
			key := fmt.Sprintf("stress_mem_%d_%d", i, j)
			val, found := memtable.Get(key)
			if !found || val != "value" {
				t.Errorf("Expected key %s to have value 'value', got '%v'", key, val)
			}
		}
	}

	t.Logf("Memtable Stress Test passed: %d operations executed successfully", totalExpected)
}
