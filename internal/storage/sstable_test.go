package storage_test

import (
	"os"
	"testing"

	"moniepoint/internal/storage"
)

// cleanup removes the test SSTable file and its associated index file.
func cleanup(filePath string) {
	os.Remove(filePath)
	os.Remove(filePath + ".index")
}

func TestSSTable_WriteAndRead(t *testing.T) {
	filePath := "test_sstable.db"
	defer cleanup(filePath)

	sstable, err := storage.NewSSTable(filePath)
	if err != nil {
		t.Fatalf("Failed to initialize SSTable: %v", err)
	}

	err = sstable.Write("txn123", "status:approved")
	if err != nil {
		t.Fatalf("Failed to write to SSTable: %v", err)
	}

	value, err := sstable.Read("txn123")
	if err != nil || value != "status:approved" {
		t.Errorf("Expected 'status:approved', got '%s'", value)
	}
}

func TestSSTable_Delete(t *testing.T) {
	filePath := "test_sstable.db"
	defer cleanup(filePath)

	sstable, err := storage.NewSSTable(filePath)
	if err != nil {
		t.Fatalf("Failed to initialize SSTable: %v", err)
	}

	sstable.Write("txn789", "status:pending")

	value, err := sstable.Read("txn789")
	if err != nil || value != "status:pending" {
		t.Fatalf("Expected 'status:pending', got '%s'", value)
	}

	sstable.Delete("txn789")

	_, err = sstable.Read("txn789")
	if err != storage.ErrKeyNotFound {
		t.Errorf("Expected key 'txn789' to be deleted, but found it")
	}
}
