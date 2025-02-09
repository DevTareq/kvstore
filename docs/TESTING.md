# **Testing Guide**

## **📌 Overview**
This document provides a structured guide to test the project, including manual testing steps, automated tests, and a checklist of completed and pending features.

---

## 1. Setting Up for Fresh Testing
Before running tests, ensure the project is in a clean state.

### **1.1 Stop the Server**
If the server is running, stop it:
```sh
kill -9 $(pgrep main)
```
Or, if running interactively:
```sh
CTRL + C
```

### 1.2 Remove Old Data (WAL & SSTable)
To start fresh, delete the persistence files:
```sh
rm -rf data/wal.log data/sstable.db
```

### **1.3 Clear Build & Test Cache**
```sh
go clean -testcache
go clean -cache
go clean -modcache
```

### **1.4 Rebuild the Project**
```sh
go mod tidy
go build ./...
```

### **1.5 Restart the Server**
```sh
go run cmd/server/main.go
```

---

## 2. Manual Testing

### **2.1 Insert Sample Key-Value Pairs (Write API)**
```sh
curl -X POST http://localhost:8080/kv/txn123 -d '{"value":"approved"}' -H "Content-Type: application/json"
curl -X POST http://localhost:8080/kv/txn456 -d '{"value":"pending"}' -H "Content-Type: application/json"
curl -X POST http://localhost:8080/kv/txn789 -d '{"value":"failed"}' -H "Content-Type: application/json"
curl -X POST http://localhost:8080/kv/txn999 -d '{"value":"approved"}' -H "Content-Type: application/json"
```
📌 **Expected Response:** `(Empty response with HTTP 201 Created)`

---

### 2.2 Insert Multiple Key-Value Pairs (Batch Write)**
```sh
curl -X POST http://localhost:8080/kv/batch \
   -H "Content-Type: application/json" \
   -d '[
         {"key": "txn1001", "value": "approved"},
         {"key": "txn1002", "value": "failed"},
         {"key": "txn1003", "value": "pending"}
      ]'
```
📌 **Expected Response:** `(Empty response with HTTP 201 Created)`

```sh 
curl -X GET http://localhost:8080/kv/txn1001
```
📌 **Expected Response:** `{"key":"txn1001","value":"approved"}`

```sh 
curl -X GET http://localhost:8080/kv/txn1002
```
📌 **Expected Response:** `{"key":"txn1002","value":"failed"}`

```sh 
curl -X GET http://localhost:8080/kv/txn1003
```
📌 **Expected Response:** `{"key":"txn1003","value":"pending"}`

---

### **2.3 Retrieve a Single Key (Read API)**
```sh
curl -X GET http://localhost:8080/kv/txn456
```
📌 **Expected Response:**
```json
{"key":"txn456","value":"pending"}
```

---

### **2.4 Retrieve Keys in a Range (ReadKeyRange API)**
```sh
curl -X GET "http://localhost:8080/kv/?start=txn123&end=txn789"
```
📌 **Expected Response:**
```json
{"txn123":"approved","txn456":"pending","txn789":"failed"}
```

---

### **2.5 Delete a Key**
```sh
curl -X DELETE http://localhost:8080/kv/txn456
```
📌 **Expected Response:** `(Empty response with HTTP 204 No Content)`

---

### **2.6 Verify Key is Deleted**
```sh
curl -X GET http://localhost:8080/kv/txn456
```
📌 **Expected Response:** `Key not found (HTTP 404)`


### **2.7 Crash Recovery (WAL Test)**
1. **Kill the running server**:
   ```sh
   kill -9 $(pgrep main)
   ```
2. **Restart the server**:
   ```sh
   go run cmd/server/main.go
   ```
3. **Verify data persistence**:
   ```sh
   curl -X GET http://localhost:8080/kv/txn123
   ```
   📌 **Expected Response:**
   ```json
   {"key":"txn123","value":"approved"}
   ```

## 3. Automated Tests

Run tests for a specific package:
```sh
go test -count=1 ./internal/storage
```

Run a specific test:
```sh
go test -count=1 -run TestSSTable_Delete ./internal/storage
```

## 4. Micro-Benchmarks

```sh
go test -bench=. -benchtime=3s ./internal/storage/
```

Run a specific Micro-Benchmark:
```sh
go test -bench=BenchmarkWALWrite -benchtime=3s ./internal/storage/
```

## 5. Stress-Test

```sh
go test -v ./internal/storage -run 'Stress'
```

Run a specific Stress-Test:
```sh
go test -v ./internal/storage -run '^Test(WALStress|MemtableStress)$'
```


