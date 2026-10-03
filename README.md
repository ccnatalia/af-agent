# ActionFabric Agent (af-agent)

## Start

```
AF_AGENT_SECRET=dev-secret AF_AGENT_WORKSPACE=. go run .
```

## Submit Task

### Download File

```shell
curl http://localhost:18081/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-758373","task_name":"download-file","payload":{"url":"https://github.com/ccnatalia/PublicRelease/releases/download/v2.16.0009.000/muxsingle_freebsd_amd64.tar.gz","filename":"myfile"}}'
```

### Download File V2

`url` and `target_path` are required. All timeout fields are optional. The
defaults are 10 seconds for connect and TLS handshake, 30 seconds for response
headers, 120 seconds for an idle response body, and 8 hours total. Timeout
values cannot exceed 28800 seconds (8 hours).

```shell
curl http://localhost:18081/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-download-v2","task_name":"download-file-v2","payload":{"url":"https://example.com/large-file.tar.gz","target_path":"downloads/large-file.tar.gz","timeouts":{"connect_seconds":10,"tls_handshake_seconds":10,"response_header_seconds":30,"body_idle_seconds":60,"total_seconds":28800}}}'
```

### Move File

```
curl http://localhost:18081/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-2341-938486","task_name":"move-file","payload":{"source_path":"downloads/myfile","target_path":"downloads/myfile_b"}}'
```

### Full Update File Manifest

Recursively hashes every regular file under `root_dir` and writes
`root_dir/file-md5.txt` using `MD5,relative-path` records.

```shell
curl http://localhost:8080/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-manifest-full","task_name":"full-update-file-manifest","payload":{"root_dir":"downloads"}}'
```

### Incremental Update File Manifest

Hashes one file and adds or replaces its record in `root_dir/file-md5.txt`.

```shell
curl http://localhost:8080/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-manifest-incremental-0005","task_name":"incremental-update-file-manifest","payload":{"root_dir":"downloads","relative_path":"large-file.tar.gz"}}'
```

### Make File Executable

```
curl http://localhost:8080/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-9348-2231","task_name":"make-file-executable","payload":{"path":"downloads/myfile_b"}}'
```

### File Exists

```
curl http://localhost:8080/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-6274-8042","task_name":"file-exists","payload":{"path":"downloads/myfile_b"}}'
```

### Delete File

```
curl http://localhost:8080/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-7584-1032","task_name":"delete-file","payload":{"path":"downloads/myfile_b"}}'
```

### Read File

`max_bytes` is optional and cannot exceed 65536 bytes (64 KiB).

```
curl http://localhost:8080/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-9417-3285","task_name":"read-file","payload":{"path":"downloads/myfile_b","max_bytes":65536}}'
```

### Terminate Processes

```
curl http://localhost:8080/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-5528-1914","task_name":"terminate-processes","payload":{"keyword":"delay_print"}}'
```

### Process Exists

```
curl http://localhost:8080/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-7712-4804","task_name":"process-exists","payload":{"keyword":"delay_print"}}'
```

### Run Startup Script

```
curl http://localhost:8080/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-8391-4322","task_name":"run-startup-script","payload":{"path":"./delay_print.sh","working_dir":"./","timeout_seconds":30}}'
```

## Test

### TestListProcesses

```shell
go test ./runner/internal/process -run TestListProcesses -v
```
