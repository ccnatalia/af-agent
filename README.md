# ActionFabric Agent (af-agent)

## Submit Task

### Download File

```shell
curl http://localhost:18081/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-758373","task_name":"download-file","payload":{"url":"https://github.com/ccnatalia/PublicRelease/releases/download/v2.16.0009.000/muxsingle_freebsd_amd64.tar.gz","filename":"myfile"}}'
```

### Move File

```
curl http://localhost:18081/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-2341-938486","task_name":"move-file","payload":{"source_path":"downloads/myfile","target_path":"downloads/myfile_b"}}'
```

### Make File Executable

```
curl http://localhost:18081/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-9348-2231","task_name":"make-file-executable","payload":{"path":"downloads/myfile_b"}}'
```

### Terminate Processes

```
curl http://localhost:18081/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-5528-1914","task_name":"terminate-processes","payload":{"keyword":"my-process-keyword"}}'
```

### Process Exists

```
curl http://localhost:18081/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-7712-4803","task_name":"process-exists","payload":{"keyword":"my-process-keyword"}}'
```

## Test

### TestListProcesses

```shell
go test ./runner/internal/process -run TestListProcesses -v
```
