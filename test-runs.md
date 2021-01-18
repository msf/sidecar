# Test Runs

Goal: to validate successfull serverless maestro

## Baseline

To have a proper baseline the following scenario is proposed:
- web (fake maestro) is coded to behave optimally in FIFO, 1 request at a time. (zero parallelism)
- web (fake maestro) does handle (by queueing) concurrent requests
- each req takes ~400-600 millis, in line with single sentence translations w/ Marian.
- Test runs test for latency on 2rps and zero errors nor losses


### Web (Maestro) HTTP

This tests the following layout:

hey -http-> sender -http-> web

- hey (benchmarking tool)
- sender (fake flowrunner), http server
  - calls web, returns error if web returns error
- web (fake maestro), http server that has CPU bound requests.

```shell
miguel@lovelace sidecar (git)[ampq-sidecar] % make latency-test-web
hey -c 2 -z 80s http://10.152.183.73:8080/web

Summary:
  Total:	80.3665 secs
  Slowest:	0.6912 secs
  Fastest:	0.2796 secs
  Average:	0.5514 secs
  Requests/sec:	3.6209

  Total data:	117855 bytes
  Size/request:	405 bytes

Response time histogram:
  0.280 [1]	|
  0.321 [0]	|
  0.362 [0]	|
  0.403 [0]	|
  0.444 [0]	|
  0.485 [0]	|
  0.527 [63]	|■■■■■■■■■■■■■■■■
  0.568 [156]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.609 [49]	|■■■■■■■■■■■■■
  0.650 [16]	|■■■■
  0.691 [6]	|■■


Latency distribution:
  10% in 0.5199 secs
  25% in 0.5282 secs
  50% in 0.5448 secs
  75% in 0.5670 secs
  90% in 0.5947 secs
  95% in 0.6216 secs
  99% in 0.6812 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0000 secs, 0.2796 secs, 0.6912 secs
  DNS-lookup:	0.0000 secs, 0.0000 secs, 0.0000 secs
  req write:	0.0000 secs, 0.0000 secs, 0.0001 secs
  resp wait:	0.5513 secs, 0.2790 secs, 0.6911 secs
  resp read:	0.0000 secs, 0.0000 secs, 0.0002 secs

Status code distribution:
  [200]	291 responses



hey -c 1 -z 80s http://10.152.183.73:8080/web

Summary:
  Total:	80.2380 secs
  Slowest:	0.4164 secs
  Fastest:	0.2457 secs
  Average:	0.2897 secs
  Requests/sec:	3.4522

  Total data:	112185 bytes
  Size/request:	405 bytes

Response time histogram:
  0.246 [1]	|
  0.263 [32]	|■■■■■■■■■■■■■
  0.280 [95]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.297 [71]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.314 [32]	|■■■■■■■■■■■■■
  0.331 [22]	|■■■■■■■■■
  0.348 [13]	|■■■■■
  0.365 [4]	|■■
  0.382 [1]	|
  0.399 [0]	|
  0.416 [6]	|■■■


Latency distribution:
  10% in 0.2597 secs
  25% in 0.2703 secs
  50% in 0.2819 secs
  75% in 0.3020 secs
  90% in 0.3287 secs
  95% in 0.3464 secs
  99% in 0.4157 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0000 secs, 0.2457 secs, 0.4164 secs
  DNS-lookup:	0.0000 secs, 0.0000 secs, 0.0000 secs
  req write:	0.0000 secs, 0.0000 secs, 0.0001 secs
  resp wait:	0.2896 secs, 0.2456 secs, 0.4163 secs
  resp read:	0.0000 secs, 0.0000 secs, 0.0002 secs

Status code distribution:
  [200]	277 responses

```

### Web (Maestro) RabbitMQ

following layout:

hey -http-> sender -amqp-> web*
    response: web* -http-> sender -http-> hey

- hey (benchmarking tool)
- sender (fake flowrunner), http server
  - calls web by enqueing a message to a rabbitMQ server
  - blocks/waits for the response callback to finish the request
  - upon async reception of resp from web, writes HTTP reply
- web (fake maestro), http server that has CPU bound requests.
  - sidecar that consumes rabbitMQ queue, redirects to web http
  - calls the callback from sender with web http response

```shell
miguel@lovelace sidecar (git)[ampq-sidecar] % make latency-test-queue
hey -c 2 -z 80s http://10.152.183.73:8080/queue

Summary:
  Total:	80.3587 secs
  Slowest:	0.8146 secs
  Fastest:	0.4198 secs
  Average:	0.6099 secs
  Requests/sec:	3.2728

  Total data:	149351 bytes
  Size/request:	567 bytes

Response time histogram:
  0.420 [1]	|
  0.459 [0]	|
  0.499 [0]	|
  0.538 [2]	|■
  0.578 [53]	|■■■■■■■■■■■■■■■■■■■
  0.617 [110]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.657 [67]	|■■■■■■■■■■■■■■■■■■■■■■■■
  0.696 [17]	|■■■■■■
  0.736 [11]	|■■■■
  0.775 [0]	|
  0.815 [2]	|■


Latency distribution:
  10% in 0.5646 secs
  25% in 0.5819 secs
  50% in 0.6019 secs
  75% in 0.6345 secs
  90% in 0.6670 secs
  95% in 0.7009 secs
  99% in 0.7975 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0000 secs, 0.4198 secs, 0.8146 secs
  DNS-lookup:	0.0000 secs, 0.0000 secs, 0.0000 secs
  req write:	0.0000 secs, 0.0000 secs, 0.0007 secs
  resp wait:	0.6098 secs, 0.4194 secs, 0.8145 secs
  resp read:	0.0001 secs, 0.0000 secs, 0.0007 secs

Status code distribution:
  [200]	263 responses



hey -c 1 -z 80s http://10.152.183.73:8080/queue

Summary:
  Total:	80.1661 secs
  Slowest:	0.4463 secs
  Fastest:	0.2566 secs
  Average:	0.3119 secs
  Requests/sec:	3.2058

  Total data:	145947 bytes
  Size/request:	567 bytes

Response time histogram:
  0.257 [1]	|■
  0.276 [8]	|■■■■
  0.295 [75]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.314 [76]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.332 [47]	|■■■■■■■■■■■■■■■■■■■■■■■■■
  0.351 [27]	|■■■■■■■■■■■■■■
  0.370 [11]	|■■■■■■
  0.389 [5]	|■■■
  0.408 [4]	|■■
  0.427 [0]	|
  0.446 [3]	|■■


Latency distribution:
  10% in 0.2833 secs
  25% in 0.2909 secs
  50% in 0.3054 secs
  75% in 0.3271 secs
  90% in 0.3503 secs
  95% in 0.3706 secs
  99% in 0.4413 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0000 secs, 0.2566 secs, 0.4463 secs
  DNS-lookup:	0.0000 secs, 0.0000 secs, 0.0000 secs
  req write:	0.0000 secs, 0.0000 secs, 0.0007 secs
  resp wait:	0.3118 secs, 0.2565 secs, 0.4461 secs
  resp read:	0.0001 secs, 0.0000 secs, 0.0010 secs

Status code distribution:
  [200]	257 responses

```

## SERVERLESS

these runs test a serverless framework.

### KEDA

Settings:
- polling interval set to 5 seconds
- cooldown period set to 300seconds
 (short cooldowns create problems with downscaling during load..)

```shell

miguel@lovelace k8s (git)[ampq-sidecar] % hey -c 2 -z 80s http://10.152.183.73:8080/queue

Summary:
  Total:	80.3716 secs
  Slowest:	6.6918 secs
  Fastest:	0.4578 secs
  Average:	0.5671 secs
  Requests/sec:	3.5211

  Total data:	160721 bytes
  Size/request:	567 bytes

Response time histogram:
  0.458 [1]	|
  1.081 [280]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  1.705 [0]	|
  2.328 [0]	|
  2.951 [0]	|
  3.575 [0]	|
  4.198 [0]	|
  4.822 [0]	|
  5.445 [0]	|
  6.068 [0]	|
  6.692 [2]	|


Latency distribution:
  10% in 0.4885 secs
  25% in 0.4963 secs
  50% in 0.5136 secs
  75% in 0.5415 secs
  90% in 0.5817 secs
  95% in 0.6204 secs
  99% in 6.4625 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0000 secs, 0.4578 secs, 6.6918 secs
  DNS-lookup:	0.0000 secs, 0.0000 secs, 0.0000 secs
  req write:	0.0000 secs, 0.0000 secs, 0.0006 secs
  resp wait:	0.5670 secs, 0.4578 secs, 6.6911 secs
  resp read:	0.0001 secs, 0.0000 secs, 0.0007 secs

Status code distribution:
  [200]	283 responses
```
and
```
# wait for deployment to scale to zero..
miguel@lovelace k8s (git)[ampq-sidecar] % hey -c 1 -z 80s http://10.152.183.73:8080/queue

Summary:
  Total:	80.0330 secs
  Slowest:	4.7368 secs
  Fastest:	0.2421 secs
  Average:	0.2760 secs
  Requests/sec:	3.6235

  Total data:	164687 bytes
  Size/request:	567 bytes

Response time histogram:
  0.242 [1]	|
  0.692 [288]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  1.141 [0]	|
  1.591 [0]	|
  2.040 [0]	|
  2.489 [0]	|
  2.939 [0]	|
  3.388 [0]	|
  3.838 [0]	|
  4.287 [0]	|
  4.737 [1]	|


Latency distribution:
  10% in 0.2440 secs
  25% in 0.2463 secs
  50% in 0.2516 secs
  75% in 0.2691 secs
  90% in 0.2889 secs
  95% in 0.3100 secs
  99% in 0.4252 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0000 secs, 0.2421 secs, 4.7368 secs
  DNS-lookup:	0.0000 secs, 0.0000 secs, 0.0000 secs
  req write:	0.0000 secs, 0.0000 secs, 0.0000 secs
  resp wait:	0.2759 secs, 0.2421 secs, 4.7364 secs
  resp read:	0.0001 secs, 0.0000 secs, 0.0006 secs

Status code distribution:
  [200]	290 responses

```


### Knative Serving

