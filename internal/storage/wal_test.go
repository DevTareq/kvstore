package storage_test

import (
	"fmt"
	"moniepoint/internal/storage"
	"os"
	"sync"
	"testing"
)

// TestMain is the entry point for testing; it cleans up the WAL directory when done.
func TestMain(m *testing.M) {
	code := m.Run()

	os.RemoveAll("data/wal")

	os.Exit(code)
}

// Helper function to enable test mode.
func enableTestMode() {
	os.Setenv("TEST_MODE", "true")
}

// Helper function to disable test mode.
func disableTestMode() {
	os.Unsetenv("TEST_MODE")
}

func TestWAL(t *testing.T) {
	enableTestMode()
	defer disableTestMode()

	wal, err := storage.NewWAL()
	if err != nil {
		t.Fatalf("Failed to initialize WAL: %v", err)
	}
	defer wal.Close()

	transactions := []struct {
		TxnID  string
		Status string
	}{
		{"txn123", "approved"},
		{"txn456", "failed"},
	}

	for _, txn := range transactions {
		if err := wal.Append(txn.TxnID, txn.Status); err != nil {
			t.Errorf("Failed to append transaction to WAL: %v", err)
		}
	}

	wal.Flush()
	data, err := wal.Replay()
	if err != nil {
		t.Fatalf("Failed to replay WAL: %v", err)
	}

	for _, txn := range transactions {
		status, exists := data[txn.TxnID]
		if !exists {
			t.Errorf("Transaction '%s' not found during replay", txn.TxnID)
		} else if status != txn.Status {
			t.Errorf("Expected status '%s' for transaction '%s', got '%s'", txn.Status, txn.TxnID, status)
		}
	}
}

func TestWALConcurrency(t *testing.T) {
	enableTestMode()
	defer disableTestMode()

	wal, err := storage.NewWAL()
	if err != nil {
		t.Fatalf("Failed to initialize WAL: %v", err)
	}
	defer wal.Close()

	var wg sync.WaitGroup
	numTransactions := 1000
	expectedEntries := make(map[string]string)

	for i := 0; i < numTransactions; i++ {
		txnID := fmt.Sprintf("txn_concurrent_%d", i)
		expectedEntries[txnID] = "success"

		wg.Add(1)
		go func(txnID string) {
			defer wg.Done()
			if err := wal.Append(txnID, "success"); err != nil {
				t.Errorf("Failed to append transaction: %v", err)
			}
		}(txnID)
	}

	wg.Wait()
	wal.Flush()

	data, err := wal.Replay()
	if err != nil {
		t.Fatalf("Failed to replay WAL: %v", err)
	}

	missingCount := 0
	for txnID := range expectedEntries {
		if _, exists := data[txnID]; !exists {
			t.Errorf("Missing concurrent transaction: %s", txnID)
			missingCount++
		}
	}

	if missingCount > 0 {
		t.Fatalf("[FAIL] %d transactions missing in WAL replay!", missingCount)
	}
}

func TestWALCrashRecovery(t *testing.T) {
	enableTestMode()
	defer disableTestMode()

	// Create a WAL, write an entry, flush, and then close to simulate a crash.
	wal, err := storage.NewWAL()
	if err != nil {
		t.Fatalf("Failed to initialize WAL: %v", err)
	}
	if err := wal.Append("txn_before_crash", "processing"); err != nil {
		t.Errorf("Failed to append transaction before crash: %v", err)
	}
	wal.Flush()
	wal.Close() // Simulate a crash.

	// Restart the WAL and replay the log.
	newWal, err := storage.NewWAL()
	if err != nil {
		t.Fatalf("Failed to restart WAL after crash: %v", err)
	}
	defer newWal.Close()

	data, err := newWal.Replay()
	if err != nil {
		t.Fatalf("Failed to replay WAL after simulated crash: %v", err)
	}

	if status, exists := data["txn_before_crash"]; !exists || status != "processing" {
		t.Errorf("Transaction lost after crash: expected 'processing', got '%s'", status)
	}
}
