
# Performance Benchmarks

Full **Testing Document**, see the [Testing Document](docs/TESTING.md).


I ran a series of benchmarks on my storage subsystem (including the Write-Ahead Log and SSTable) on a macOS (Darwin) system with an ARM64 architecture. These benchmarks demonstrate the efficiency of my design for both write and read operations.

### Benchmark Results

| **Benchmark**               | **Iterations** | **Average Time per Operation**         | **Description**                                                                                  |
|-----------------------------|----------------|----------------------------------------|--------------------------------------------------------------------------------------------------|
| **WAL Write**               | 3,759,094      | 2,303 ns/op (2.30 µs)                  | fast transaction logging with minimal latency, enabling high throughput.               |
| **WAL Replay**              | 220            | 17,719,864 ns/op (17.72 ms)             | Replaying the WAL (used for recovery/startup) takes ~17.7 ms, should be acceptable given its infrequent use. |
| **SSTable Read**            | 247,731,052    | 14.27 ns/op                            | Ultra-fast read operations ensure near-instantaneous data retrieval.                             |
| **Memtable Write**          | 30,471,546     | 114.3 ns/op                            | Highly efficient in-memory writes ensure rapid data ingestion.                                   |
| **Memtable Read**           | 297,253,634    | 12.12 ns/op                            | low-latency in-memory reads enable fast access to data.                                |
| **SSTable Batch Write**     | 819            | 4,101,589 ns/op (4.10 ms)              | Batch writes to the SSTable occur in the low-millisecond range, suitable for bulk operations.    |
| **SSTable Compaction**      | 8,794          | 398,629 ns/op (0.40 ms)                | Compaction runs efficiently, taking less than half a millisecond per operation.                  |

Below is the raw benchmark output for reference:

```bash
goos: darwin
goarch: arm64
pkg: moniepoint/internal/storage
BenchmarkWALWrite-12             	 3759094	      2303 ns/op
BenchmarkWALReplay-12            	     220	  17719864 ns/op
BenchmarkSSTableRead-12          	247731052	        14.27 ns/op
BenchmarkMemtableWrite-12        	30471546	       114.3 ns/op
BenchmarkMemtableRead-12         	297253634	        12.12 ns/op
BenchmarkSSTableBatchWrite-12    	     819	   4101589 ns/op
BenchmarkSSTableCompaction-12    	    8794	    398629 ns/op
PASS
ok  	moniepoint/internal/storage	68.546s
```

## Stress Test Results

I conducted stress tests on my storage subsystem to verify its robustness under heavy concurrent load. Here are the details:


### Summary Table

| **Component**   | **Total Operations** | **Execution Time** | **Outcome**                                      |
|-----------------|----------------------|--------------------|--------------------------------------------------|
| WAL             | 1,000,000            | 3.61 seconds       | All entries logged & aggregated successfully.    |
| Memtable        | 1,000,000            | 0.40 seconds       | All operations executed successfully.            |


Below is the raw benchmark output for reference:

```bash 
=== RUN   TestWALStress
    stress_test.go:102: WAL Stress Test passed: 1000000 entries logged and aggregated successfully
--- PASS: TestWALStress (3.61s)
=== RUN   TestMemtableStress
    stress_test.go:143: Memtable Stress Test passed: 1000000 operations executed successfully
--- PASS: TestMemtableStress (0.40s)
PASS
ok  	moniepoint/internal/storage	4.213s 
```

### WAL Stress Test
- **Test Parameters:**  
  100 goroutines concurrently performed 10,000 append operations each, totaling **1,000,000 WAL entries**.
- **Outcome:**  
  All 1,000,000 entries were successfully logged and aggregated, ensuring no data loss even with log rotation.
- **Execution Time:**  
  Approximately **3.61 seconds**.


### Memtable Stress Test
- **Test Parameters:**  
50 goroutines concurrently executed 20,000 write operations each, totaling **1,000,000 operations**.
- **Outcome:**  
All operations were executed successfully with the expected data retrieved.
- **Execution Time:**  
Approximately **0.40 seconds**.

