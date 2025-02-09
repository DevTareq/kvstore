package storage_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"moniepoint/internal/storage"
)

// Benchmark WAL Write Performance
func BenchmarkWALWrite(b *testing.B) {
	wal, err := storage.NewWAL()
	if err != nil {
		b.Fatalf("Failed to initialize WAL: %v", err)
	}
	defer wal.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := fmt.Sprintf("txn_bench_%d", i)
		_ = wal.Append(key, "pending")
	}
}

// Benchmark WAL Replay Performance
func BenchmarkWALReplay(b *testing.B) {
	wal, err := storage.NewWAL()
	if err != nil {
		b.Fatalf("Failed to initialize WAL: %v", err)
	}
	defer wal.Close()

	numEntries := 10000
	for i := 0; i < numEntries; i++ {
		wal.Append(fmt.Sprintf("txn_replay_%d", i), "status_pending")
	}

	time.Sleep(2 * time.Second)
	wal.Flush()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = wal.Replay()
	}
}

// Benchmark SSTable Read Performance
func BenchmarkSSTableRead(b *testing.B) {
	sstableFile := "benchmark_sstable.db"

	os.Remove(sstableFile)

	sstable, err := storage.NewSSTable(sstableFile)
	if err != nil {
		b.Fatalf("Failed to initialize SSTable: %v", err)
	}

	defer func() {
		sstable.Close()
		os.Remove(sstableFile)
	}()

	// ✅ Insert test data
	for i := 0; i < 1000; i++ {
		sstable.Write(fmt.Sprintf("key_%d", i), fmt.Sprintf("value_%d", i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = sstable.Read("key_50000")
	}
}

// Benchmark Memtable Write Performance
func BenchmarkMemtableWrite(b *testing.B) {
	memtable := storage.NewMemtable(100000, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		memtable.Set(fmt.Sprintf("key_%d", i), "value")
	}
}

// Benchmark Memtable Read Performance
func BenchmarkMemtableRead(b *testing.B) {
	memtable := storage.NewMemtable(100000, nil)
	for i := 0; i < 100000; i++ {
		memtable.Set(fmt.Sprintf("key_%d", i), "value")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = memtable.Get("key_50000")
	}
}

// Benchmark SSTable Batch Write Performance
func BenchmarkSSTableBatchWrite(b *testing.B) {
	sstableFile := "benchmark_sstable.db"

	os.Remove(sstableFile)

	sstable, err := storage.NewSSTable(sstableFile)
	if err != nil {
		b.Fatalf("Failed to initialize SSTable: %v", err)
	}

	defer func() {
		sstable.Close()
		os.Remove(sstableFile)
	}()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = sstable.Write(fmt.Sprintf("batch_key_%d", i), "batch_value")
	}
}

// Benchmark SSTable Compaction Performance
func BenchmarkSSTableCompaction(b *testing.B) {
	tmpDir := b.TempDir()

	createDummySSTableFiles := func() []string {
		sstables := []string{
			filepath.Join(tmpDir, "sstable1.db"),
			filepath.Join(tmpDir, "sstable2.db"),
			filepath.Join(tmpDir, "sstable3.db"),
		}

		validContent := []byte(`{"key":"dummy1","value":"value1"}` + "\n" +
			`{"key":"dummy2","value":"value2"}` + "\n")
		for _, file := range sstables {
			err := os.WriteFile(file, validContent, 0644)
			if err != nil {
				b.Fatalf("Failed to create dummy SSTable file %s: %v", file, err)
			}
		}
		return sstables
	}

	compactedFile := filepath.Join(tmpDir, "compacted_sstable.db")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sstables := createDummySSTableFiles()

		if err := storage.CompactSSTables(sstables, compactedFile); err != nil {
			b.Fatalf("Compaction failed: %v", err)
		}

		// ✅ Explicit cleanup
		for _, file := range sstables {
			os.Remove(file)
		}
		os.Remove(compactedFile)
	}
}
