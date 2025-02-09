# **ADR: LSM-Tree-Based Key/Value Store Design & Architecture**

---

## **1. Context**
I need a **low-latency, high-throughput, and crash-resilient** architecture for storing key-value data that can **scale beyond available RAM** while maintaining **predictable performance** under heavy workloads.

### **Key Design Goals**
- **Efficient writes** for random workloads.
- **Reliable crash recovery** to prevent data loss.
- **Optimized range queries** via `ReadKeyRange`.
- **Scalability** to handle large datasets.
- **Predictable performance** under high concurrency.

---

## **2. Decision: LSM-Tree Storage Model**
I chose the **Log-Structured Merge Tree (LSM-Tree) approach** because it balances **write performance, read efficiency, and crash resilience**.

### **2. Decision: LSM-Tree Storage Model**  
The **Log-Structured Merge Tree (LSM-Tree)** was chosen for its **high write performance, efficient reads, and crash resilience**.

### **Storage Layers**
1. **Write-Ahead Log (WAL)**  
   - Ensures **durability** and **crash recovery** via log replay.  
   - **Optimized with:**  
     - **Asynchronous Writes**: Buffered queue for non-blocking operations.  
     - **Log Rotation & Retention**: Auto-rotates at `10MB`, keeping the last `5` logs.  
     - **Crash Recovery with Checksum**: Skips corrupted entries during replay.  

2. **Memtable (In-Memory Storage)**  
   - Provides **low-latency access** before persisting data.  
   - **Optimized with:**  
     - **Thread-Safe Writes**: Uses `sync.RWMutex` for concurrency.  
     - **Auto-Flushing**: Flushes to SSTable upon reaching `maxEntries`.  
     - **Binary Search for Fast Range Queries**.  
     - **Optimized Flush Mechanism** to structure SSTable writes.  

3. **SSTables (Persistent Storage)**  
   - Immutable, **sorted** files for **fast lookups & range queries**.  
   - **Optimized with:**  
     - **In-Memory Indexing**: Quick key lookups within SSTables.  
     - **Concurrent Reads** & append-only writes.  
     - **Compaction-Aware Design** to reduce redundant writes.  

4. **Compaction Process**  
   - **Merges SSTables** to eliminate obsolete data & improve efficiency.  
   - **Optimized with:**  
     - **Incremental Compaction**: Merges only when `30%+` of keys are deleted.  
     - **Adaptive Scheduling**: Adjusts based on system load.  
     - **Automatic Cleanup**: Removes outdated SSTables post-compaction.  

5. **Replication & Consensus (Raft) [Future Scope]**  
   - Ensures **high availability** & **failover handling**.  
   - Future enhancements:  
     - **Log-based replication** for data consistency.  
     - **Leader election mechanisms** for seamless failover.

---

## **3. Architecture Overview**

### **System Architecture**
[Architecture Diagram](images/architecture.png)  
*This diagram illustrates the overall system architecture, including the API layer, request handling, storage engine, and replication components.*

### **Write Path**
[Write Path Diagram](images/write_path.png)  
*Shows the data flow for handling write operations, including WAL logging, Memtable updates, and SSTable persistence.*

### **Read Path**
[Read Path Diagram](images/read_path.png)  
*Depicts the process of retrieving data from memory and persistent storage, optimizing for fast lookups and range queries.*

### **Compaction Process**
[Compaction Flow Diagram](images/compaction_flow.png)  
*Explains how multiple SSTables are merged to optimize storage and improve read efficiency through background compaction.*

### **WAL Process**
[WAL Flow Diagram](images/wal_flow.png)  
*Illustrates how the WAL ensures durability, crash recovery, and data consistency in the system.*


---

## **4. Consequences**
### ✅ **Positive Outcomes**
✔ **High Write Throughput** – Append-only WAL and in-memory writes boost speed.  
✔ **Crash Resilience** – WAL replay ensures no data loss after a crash.  
✔ **Scalable Storage** – Disk-based SSTables allow large datasets to be managed.  
✔ **Efficient Range Queries** – Sorted SSTables make sequential scans fast.

### ⚠ **Trade-offs & Mitigations**
⚠ **Read Amplification** – Queries may scan multiple SSTables.  
   - **Mitigation**: Use **Bloom filters & caching** to reduce lookup overhead (Bloom filters were considered but deferred to keep the solution within the standard library).

⚠ **Compaction Overhead** – Merging SSTables is CPU & I/O intensive.  
   - **Mitigation**: Implement **background compaction & throttling**.

⚠ **Write Latency Spikes** – SSTable flushes may cause temporary slowdowns.  
   - **Mitigation**: Use **adaptive memtable sizing & staggered flushes**.

---

## **5. Next Steps**
1. **Optimize Compaction Strategies**  
   - Implement **level-based or size-tiered compaction** to balance storage efficiency.
2. **Introduce Multi-Node Replication** *(Future Scope)*  
   - Explore **Raft-based replication** for fault tolerance and high availability.
3. **Advanced Caching Strategies**  
   - Implement **hot-data caching** to further minimize read latency.

---

## **6. References**
- [Bigtable: A Distributed Storage System for Structured Data](https://static.googleusercontent.com/media/research.google.com/en//archive/bigtable-osdi06.pdf)
- [Bitcask Intro](https://riak.com/assets/bitcask-intro.pdf)
- [LSM-Tree Paper](https://www.cs.umb.edu/~poneil/lsmtree.pdf)
- [Google Research on Log-Structured Systems](https://research.google/pubs/pub44830/)
