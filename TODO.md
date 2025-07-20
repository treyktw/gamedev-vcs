# Phase Implementation Plan - Files & Changes

## **Phase 1: Git-Style Object Storage + Stat Optimization**

### Files to Modify:
- `client.go`
- `operations.go` 
- `handlers.go`
- `commands.go`
- `main.go`

### Files to Create:
- `object_store.go`
- `file_index.go`

### Files to Remove/Clean:
- Remove individual file logging from `handlers.go`
- Remove per-file database calls from `operations.go`

### Changes:

**`object_store.go` (NEW)**
- Create GitStyleObjectStore struct with Store/Exists/Get methods
- Implement zlib compression for objects
- Use Git-style directory structure (objects/aa/bbcc...)
- Add atomic file writing for object storage

**`file_index.go` (NEW)**
- Create IndexEntry struct (path, hash, size, mtime, inode)
- Implement NeedsUpdate() method using stat() optimization
- Add binary index file reading/writing (like .git/index)
- Include batch update methods for index

**`client.go`**
- Replace UploadFile with ProcessFilesBatchGitStyle method
- Add stat-based file change detection before hashing
- Remove individual file upload loops
- Add batch object upload method
- Remove FileCache struct (replaced by IndexEntry system)

**`operations.go`**
- Remove individual UploadFile method
- Add batch object processing
- Remove per-file storage calls
- Remove individual analytics logging
- Add batch content storage using object store

**`handlers.go`**
- Remove individual uploadFile handler database calls
- Remove per-file analytics logging
- Keep uploadFile for single file uploads but make it use object store
- Remove individual file database insertions

**`commands.go`**
- Replace sequential file loop with batch processing call
- Remove individual file progress reporting
- Add batch progress reporting
- Remove per-file error handling (batch at end)

**`main.go`**
- Remove individual file upload routes if any
- Keep existing routes but modify to use object store

---

## **Phase 2: Batch Database Operations**

### Files to Modify:
- `handlers.go`
- `client.go`
- `database.go`
- `models.go`

### Files to Create:
- `batch_operations.go`

### Changes:

**`batch_operations.go` (NEW)**
- Create BatchDatabaseUpdate struct
- Implement server-side batch file updates
- Add single transaction for multiple files
- Create batch upsert operations

**`handlers.go`**
- Add batchUpdateFiles handler
- Remove all individual database calls from uploadFile
- Add single batch endpoint for file metadata
- Remove per-file database transactions

**`client.go`**
- Add batchUpdateDatabase method
- Remove individual file database update calls
- Add single HTTP request for all file metadata
- Remove per-file server communication

**`database.go`**
- Add batch insert/update methods using GORM's CreateInBatches
- Remove individual file creation methods
- Add batch transaction handling
- Remove per-file database queries

**`models.go`**
- Add BatchDatabaseUpdate struct
- Keep existing File model but remove individual operations
- Add batch validation methods

---

## **Phase 3: Content Deduplication + Server Optimization**

### Files to Modify:
- `handlers.go`
- `operations.go`
- `client.go`

### Files to Create:
- `content_dedup.go`

### Changes:

**`content_dedup.go` (NEW)**
- Add content-addressable storage interface
- Implement hash-based deduplication
- Add exists checking by content hash
- Create compression utilities

**`handlers.go`**
- Add checkObjectExists endpoint (HEAD /objects/:hash)
- Modify upload to check for existing content first
- Remove duplicate content storage
- Add batch object existence checking

**`operations.go`**
- Add content deduplication before storage
- Remove duplicate file storage logic
- Add hash-first storage approach
- Remove path-based storage

**`client.go`**
- Add batch existence checking before upload
- Remove uploading of existing content
- Add content-hash-based skipping
- Remove path-based upload logic

---

## **Phase 4: Advanced Optimizations**

### Files to Modify:
- `client.go`
- `handlers.go`

### Files to Create:
- `compression.go`
- `streaming.go`

### Changes:

**`compression.go` (NEW)**
- Add streaming compression for large files
- Implement zlib compression like Git
- Add decompression utilities
- Create compressed batch uploads

**`streaming.go` (NEW)**
- Add streaming hash calculation (no full file reads)
- Implement memory-mapped file reading for large files
- Add chunked processing for massive files
- Create streaming upload capabilities

**`client.go`**
- Add streaming hash calculation
- Remove full file reading into memory
- Add memory-mapped file processing
- Remove memory-intensive operations

**`handlers.go`**
- Add streaming upload handling
- Remove full request body reading
- Add chunked processing
- Remove memory-intensive request handling

---

## **Code to Remove/Delete:**

### From `client.go`:
- Remove `FileCache` struct and all methods
- Remove `ParallelUploadFiles` (replace with batch)
- Remove individual `UploadFile` calls in loops
- Remove per-file error handling
- Remove individual progress reporting

### From `handlers.go`:
- Remove individual database insertions in `uploadFile`
- Remove per-file analytics logging
- Remove individual file processing loops
- Remove per-file transaction handling

### From `operations.go`:
- Remove individual `UploadFile` method
- Remove per-file storage operations
- Remove individual analytics recording
- Remove path-based file operations

### From `commands.go`:
- Remove sequential file processing loop
- Remove individual file open/close operations
- Remove per-file success/error reporting
- Remove individual file state management

### From `database.go`:
- Remove individual file creation methods
- Remove per-file database queries
- Remove individual transaction handling
- Remove single-file repository methods

## **Performance Impact by Phase:**

**Phase 1**: 2-5x improvement (stat optimization + object storage)
**Phase 2**: 5-10x improvement (batch database operations)
**Phase 3**: 10-20x improvement (content deduplication)
**Phase 4**: 20-50x improvement (streaming + compression)

**Expected Final Result**: 566 files processed in 5-15 seconds instead of 5 minutes, achieving Git-like performance in Go.


okay this is still taking forever so i want to modify our approach by alot 

first git is basically a file / project store so a db is not really needed so that was some overhead by me 

so instead can we have this flow 

instead of using json lets use a binary index 

Stores content as zlib-compressed blobs

No extra metadata storage per file unless changed

Commits only reference trees of blob hashes, not files

we are still logging every file 

2025/06/26 03:36:35 /Users/treymurray/DevSpace/Low-Level/gamedev-vcs/database/database.go:513

[84.748ms] [rows:1] INSERT INTO "files" ("id","project_id","path","content_hash","size","mime_type","branch","is_locked","locked_by","locked_at","last_modified_by","last_modified_at") VALUES ('file_1750923394998791000','proj_1750923171427936000','fbf3e4cbd107692c9e0ab30741b73c8c35d7022c226c0912e17418b9ba1fc211','fbf3e4cbd107692c9e0ab30741b73c8c35d7022c226c0912e17418b9ba1fc211',17163,'application/octet-stream','main',false,NULL,NULL,'2c974a19-41b6-4236-a1b3-2f0bd23f363a','2025-06-26 03:36:34.998') RETURNING "last_modified_at"

Failed to record file event in ClickHouse: code: 60, message: Table default.collaboration_events does not exist. Maybe you meant vcs_analytics.collaboration_events?

