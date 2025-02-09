package storage_test

import (
	"fmt"
	"reflect"
	"sync"
	"testing"

	"moniepoint/internal/storage"
)

// TestMemtable covers basic operations: Set, Get, and Delete.
func TestMemtable(t *testing.T) {
	memtable := storage.NewMemtable(1000, nil)

	memtable.Set("payment1", "approved")
	value, exists := memtable.Get("payment1")
	if !exists || value != "approved" {
		t.Errorf("Expected 'approved' for payment1, got '%s'", value)
	}

	_, exists = memtable.Get("payment2")
	if exists {
		t.Errorf("Expected key 'payment2' to not exist")
	}

	memtable.Delete("payment1")
	_, exists = memtable.Get("payment1")
	if exists {
		t.Errorf("Expected 'payment1' to be deleted")
	}
}

// TestMemtableGetRange tests the GetRange function for proper range querying.
func TestMemtableGetRange(t *testing.T) {
	memtable := storage.NewMemtable(1000, nil)

	memtable.Set("paymentA", "approved")
	memtable.Set("paymentB", "declined")
	memtable.Set("paymentC", "pending")
	memtable.Set("paymentD", "approved")

	expected := map[string]string{
		"paymentB": "declined",
		"paymentC": "pending",
	}

	result := memtable.GetRange("paymentB", "paymentC")

	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected %v, got %v", expected, result)
	}
}

// flushRecorder is a helper type to record flush events.
type flushRecorder struct {
	called bool
	data   map[string]string
}

func (fr *flushRecorder) flush(data map[string]string) {
	fr.called = true
	fr.data = make(map[string]string)
	for k, v := range data {
		fr.data[k] = v
	}
}

// TestMemtableAutoFlush verifies that the Memtable flushes when the maximum capacity is reached.
func TestMemtableAutoFlush(t *testing.T) {
	recorder := &flushRecorder{}
	memtable := storage.NewMemtable(3, recorder.flush)

	memtable.Set("txn1", "approved")
	memtable.Set("txn2", "declined")
	memtable.Set("txn3", "pending") // This should trigger auto-flush.

	if size := memtable.Size(); size != 0 {
		t.Errorf("Expected memtable to be empty after flush, got size %d", size)
	}

	if !recorder.called {
		t.Error("Expected flush function to be called")
	}
	expectedData := map[string]string{
		"txn1": "approved",
		"txn2": "declined",
		"txn3": "pending",
	}
	if !reflect.DeepEqual(recorder.data, expectedData) {
		t.Errorf("Expected flushed data %v, got %v", expectedData, recorder.data)
	}
}

// TestMemtableConcurrency tests concurrent access to the Memtable.
func TestMemtableConcurrency(t *testing.T) {
	memtable := storage.NewMemtable(1000, nil)
	var wg sync.WaitGroup

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			memtable.Set(key, "value")
		}(i)
	}

	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			key := fmt.Sprintf("key%d", i)
			_, _ = memtable.Get(key)
		}(i)
	}

	wg.Wait()
}
