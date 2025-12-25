# LLM Verifier Performance Benchmark Report

Generated: Thu Dec 25 10:34:34 PM MSK 2025
Test Duration: 60 seconds
Concurrent Users: 10

## Health Endpoint Performance

Summary:
  Total:	10.0007 secs
  Slowest:	0.0009 secs
  Fastest:	0.0001 secs
  Average:	0.0002 secs
  Requests/sec:	99.9925
  
  Total data:	44000 bytes
  Size/request:	44 bytes

Response time histogram:
  0.000 [1]	|
  0.000 [59]	|■■■■■
  0.000 [507]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.000 [314]	|■■■■■■■■■■■■■■■■■■■■■■■■■
  0.000 [88]	|■■■■■■■
  0.000 [13]	|■
  0.001 [1]	|
  0.001 [12]	|■
  0.001 [1]	|
  0.001 [3]	|
  0.001 [1]	|


Latency distribution:
  10% in 0.0002 secs
  25% in 0.0002 secs
  50% in 0.0002 secs
  75% in 0.0003 secs
  90% in 0.0003 secs
  95% in 0.0004 secs
  99% in 0.0006 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0000 secs, 0.0001 secs, 0.0009 secs
  DNS-lookup:	0.0000 secs, 0.0000 secs, 0.0003 secs
  req write:	0.0000 secs, 0.0000 secs, 0.0003 secs
  resp wait:	0.0002 secs, 0.0000 secs, 0.0004 secs
  resp read:	0.0000 secs, 0.0000 secs, 0.0003 secs

Status code distribution:
  [200]	1000 responses

## Providers Endpoint Performance

Summary:
  Total:	20.0007 secs
  Slowest:	0.0008 secs
  Fastest:	0.0001 secs
  Average:	0.0002 secs
  Requests/sec:	24.9991
  
  Total data:	155500 bytes
  Size/request:	311 bytes

Response time histogram:
  0.000 [1]	|
  0.000 [95]	|■■■■■■■■■■■■■■■■■■
  0.000 [215]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.000 [141]	|■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.000 [35]	|■■■■■■■
  0.000 [8]	|■
  0.001 [0]	|
  0.001 [2]	|
  0.001 [2]	|
  0.001 [0]	|
  0.001 [1]	|


Latency distribution:
  10% in 0.0002 secs
  25% in 0.0002 secs
  50% in 0.0002 secs
  75% in 0.0003 secs
  90% in 0.0003 secs
  95% in 0.0004 secs
  99% in 0.0006 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0000 secs, 0.0001 secs, 0.0008 secs
  DNS-lookup:	0.0000 secs, 0.0000 secs, 0.0003 secs
  req write:	0.0000 secs, 0.0000 secs, 0.0001 secs
  resp wait:	0.0002 secs, 0.0001 secs, 0.0004 secs
  resp read:	0.0000 secs, 0.0000 secs, 0.0001 secs

Status code distribution:
  [200]	500 responses

## Models Endpoint Performance

Summary:
  Total:	33.3341 secs
  Slowest:	0.0007 secs
  Fastest:	0.0001 secs
  Average:	0.0002 secs
  Requests/sec:	8.9998
  
  Total data:	113400 bytes
  Size/request:	378 bytes

Response time histogram:
  0.000 [1]	|
  0.000 [14]	|■■■■■
  0.000 [77]	|■■■■■■■■■■■■■■■■■■■■■■■■■
  0.000 [121]	|■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■■
  0.000 [46]	|■■■■■■■■■■■■■■■
  0.000 [36]	|■■■■■■■■■■■■
  0.000 [2]	|■
  0.000 [0]	|
  0.001 [1]	|
  0.001 [1]	|
  0.001 [1]	|


Latency distribution:
  10% in 0.0002 secs
  25% in 0.0002 secs
  50% in 0.0002 secs
  75% in 0.0003 secs
  90% in 0.0003 secs
  95% in 0.0004 secs
  99% in 0.0005 secs

Details (average, fastest, slowest):
  DNS+dialup:	0.0000 secs, 0.0001 secs, 0.0007 secs
  DNS-lookup:	0.0000 secs, 0.0000 secs, 0.0002 secs
  req write:	0.0000 secs, 0.0000 secs, 0.0001 secs
  resp wait:	0.0002 secs, 0.0001 secs, 0.0003 secs
  resp read:	0.0000 secs, 0.0000 secs, 0.0001 secs

Status code distribution:
  [200]	300 responses

## Recommendations

### Performance Targets Met:
- [ ] Average response time < 200ms
- [ ] 95th percentile < 500ms
- [ ] Error rate < 1%
- [ ] Throughput > 100 req/sec

### Optimization Opportunities:
- Database query optimization
- Caching implementation
- Connection pooling tuning
- Horizontal scaling evaluation

