package storage_test

import (
	"bufio"
	"encoding/json"
	"moniepoint/internal/storage"
	"os"
	"testing"
)

// Test SSTable Compaction with proper cleanup
func TestSSTableCompaction(t *testing.T) {
	os.Setenv("TEST_MODE", "true")
	defer os.Unsetenv("TEST_MODE")

	testFiles := []string{
		"sstable_1.db",
		"sstable_2.db",
		"sstable_3.db",
	}

	defer cleanTestFiles(testFiles)

	for _, filePath := range testFiles {
		file, err := os.Create(filePath)
		if err != nil {
			t.Fatalf("[FATAL] Failed to create SSTable %s: %v", filePath, err)
		}
		defer file.Close()

		entries := []map[string]string{
			{"key": "key1", "value": "value1"},
			{"key": "key2", "value": "value2"},
			{"key": "key3", "value": "DELETE"},
		}

		for _, entry := range entries {
			jsonData, _ := json.Marshal(entry)
			_, err = file.WriteString(string(jsonData) + "\n")
			if err != nil {
				t.Fatalf("[FATAL] Failed to write to SSTable %s: %v", filePath, err)
			}
		}
	}

	compactedFile := "compacted_sstable.db"
	defer os.Remove(compactedFile)

	err := storage.CompactSSTables(testFiles, compactedFile)
	if err != nil {
		t.Fatalf("[FATAL] Compaction failed: %v", err)
	}

	if _, err := os.Stat(compactedFile); os.IsNotExist(err) {
		t.Fatalf("[FATAL] Compacted SSTable not found")
	}

	compactedFileHandle, err := os.Open(compactedFile)
	if err != nil {
		t.Fatalf("[FATAL] Failed to open compacted SSTable: %v", err)
	}
	defer compactedFileHandle.Close()

	expectedData := map[string]string{
		"key1": "value1",
		"key2": "value2",
	}

	scanner := bufio.NewScanner(compactedFileHandle)
	for scanner.Scan() {
		var entry map[string]string
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			t.Fatalf("[FATAL] Failed to unmarshal entry in compacted SSTable: %v", err)
		}

		if entry["key"] == "key3" {
			t.Fatalf("[FATAL] Deleted key 'key3' was found in compacted SSTable")
		}

		if val, exists := expectedData[entry["key"]]; exists {
			if val != entry["value"] {
				t.Errorf("[ERROR] Expected value '%s' for key '%s', got '%s'", val, entry["key"], entry["value"])
			}
		} else {
			t.Errorf("[ERROR] Unexpected key '%s' found in compacted SSTable", entry["key"])
		}
	}

	for _, filePath := range testFiles {
		if _, err := os.Stat(filePath); !os.IsNotExist(err) {
			t.Fatalf("[FATAL] Old SSTable %s still exists after compaction", filePath)
		}
	}
}

func cleanTestFiles(files []string) {
	for _, file := range files {
		_ = os.Remove(file)
	}
}
