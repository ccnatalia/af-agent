# Submit task

```shell
curl http://localhost:18081/api/tasks/submit \
     -H 'secret: dev-secret' \
     -H 'Content-Type: application/json' \
     -d '{"request_id":"req-758373","task_name":"download-file","payload":{"url":"https://github.com/ccnatalia/PublicRelease/releases/download/v2.16.0009.000/muxsingle_freebsd_amd64.tar.gz","filename":"myfile"}}'
```