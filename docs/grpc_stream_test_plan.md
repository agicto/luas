# gRPC Stream Synchronization Test Plan

> **Note**: This test plan is for the **zgi-api/zgi-console-api** gRPC Stream synchronization feature, documented here for reference.

## 1. Overview

### 1.1 Feature Description

The gRPC Stream synchronization feature enables real-time channel data synchronization between Console API (server) and Gateway (client) using bidirectional streaming.

### 1.2 Architecture

```text
┌─────────────────────────────────────────────────────────────┐
│                     Console API (Server)                     │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ ChannelService (gRPC Server)                           │ │
│  │  • WatchChannels() - Stream endpoint                   │ │
│  │  • EventBroadcaster - Publish events                   │ │
│  │  • ChannelRepository - Data source                     │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                            ↓ gRPC Stream
┌─────────────────────────────────────────────────────────────┐
│                      Gateway (Client)                        │
│  ┌────────────────────────────────────────────────────────┐ │
│  │ RemoteChannelSync                                      │ │
│  │  • Subscribe to events                                 │ │
│  │  • Local cache management                              │ │
│  │  • Auto-reconnect on failure                           │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### 1.3 Event Types

| Event Type | Description | Trigger |
|-----------|-------------|---------|
| `SNAPSHOT` | Full channel snapshot | Initial connection |
| `CREATED` | New channel created | POST /v1/platform/channels |
| `UPDATED` | Channel modified | PUT /v1/platform/channels/:id |
| `ENABLED` | Channel enabled | PATCH /v1/platform/channels/:id/enable |
| `DISABLED` | Channel disabled | PATCH /v1/platform/channels/:id/disable |
| `DELETED` | Channel removed | DELETE /v1/platform/channels/:id |

### 1.4 Synchronized Data Structure

```json
{
  "id": "channel-uuid",
  "name": "Channel Name",
  "provider": "openai",
  "protocol": "openai",
  "api_base_url": "https://api.openai.com/v1",
  "api_key_encrypted": "encrypted-key",
  "models": ["gpt-4", "gpt-3.5-turbo"],
  "priority": 100,
  "weight": 1,
  "is_enabled": true,
  "metadata": {}
}
```

## 2. Test Scope

### 2.1 In-Scope

- ✅ gRPC server implementation (Console API)
- ✅ gRPC client implementation (Gateway)
- ✅ Event broadcasting mechanism
- ✅ Real-time synchronization
- ✅ Connection resilience (reconnect)
- ✅ Data consistency validation
- ✅ Performance under load

### 2.2 Out-of-Scope

- ❌ HTTP REST API testing (separate test suite)
- ❌ Database migration testing
- ❌ Authentication/Authorization (assumes valid credentials)

## 3. Test Environment

### 3.1 Prerequisites

```bash
# Required services
- Console API running on port 8080 (or configured port)
- PostgreSQL database with migrations applied
- Gateway service (or test client)

# Required tools
- grpcurl (for manual testing)
- Go 1.21+ (for running test clients)
- psql (for database verification)
```

### 3.2 Test Data Setup

```sql
-- Verify test channels exist
SELECT id, name, provider, protocol, is_enabled 
FROM llm_system_channels 
WHERE is_system = true;

-- Expected: At least 10 official channels
```

## 4. Test Cases

### 4.1 Functional Tests

#### TC-F-001: Initial Snapshot Synchronization

**Objective**: Verify Gateway receives full channel snapshot on connection

**Preconditions**:
- Console API running with 10+ official channels
- Gateway not yet connected

**Steps**:
1. Start Gateway with gRPC client enabled
2. Observe connection logs
3. Verify SNAPSHOT event received
4. Validate all channels in local cache

**Expected Results**:
- ✅ Connection established successfully
- ✅ SNAPSHOT event type received
- ✅ All 10+ channels present in Gateway cache
- ✅ Channel data matches database records

**Verification**:
```bash
# Check Gateway logs
grep "Received SNAPSHOT" gateway.log

# Verify channel count
curl http://localhost:gateway-port/internal/channels | jq '.data | length'
```

---

#### TC-F-002: Real-time Channel Creation

**Objective**: Verify Gateway receives CREATED event when new channel is added

**Preconditions**:
- Gateway connected and synced
- Initial channel count = N

**Steps**:
1. Create new channel via Console API:
   ```bash
   curl -X POST http://localhost:8080/v1/platform/channels \
     -H 'Content-Type: application/json' \
     -d '{
       "name": "Test Channel",
       "provider": "openai",
       "api_key": "sk-test-key"
     }'
   ```
2. Observe Gateway logs
3. Check Gateway cache

**Expected Results**:
- ✅ CREATED event received within 1 second
- ✅ New channel appears in Gateway cache
- ✅ Channel count = N + 1
- ✅ All channel fields match creation request

**Verification**:
```bash
# Gateway logs
grep "Event: CREATED" gateway.log | tail -1

# Cache verification
curl http://localhost:gateway-port/internal/channels | \
  jq '.data[] | select(.name=="Test Channel")'
```

---

#### TC-F-003: Channel Update Synchronization

**Objective**: Verify Gateway receives UPDATED event when channel is modified

**Preconditions**:
- Test channel exists with ID = `test-channel-id`
- Gateway synced

**Steps**:
1. Update channel via Console API:
   ```bash
   curl -X PUT http://localhost:8080/v1/platform/channels/test-channel-id \
     -H 'Content-Type: application/json' \
     -d '{
       "name": "Updated Channel Name",
       "priority": 200
     }'
   ```
2. Observe Gateway logs
3. Verify cache update

**Expected Results**:
- ✅ UPDATED event received
- ✅ Channel name updated in cache
- ✅ Priority changed to 200
- ✅ Other fields unchanged

---

#### TC-F-004: Channel Enable/Disable Events

**Objective**: Verify status change events propagate correctly

**Preconditions**:
- Test channel enabled

**Steps**:
1. Disable channel:
   ```bash
   curl -X PATCH http://localhost:8080/v1/platform/channels/test-channel-id/disable
   ```
2. Verify DISABLED event
3. Enable channel:
   ```bash
   curl -X PATCH http://localhost:8080/v1/platform/channels/test-channel-id/enable
   ```
4. Verify ENABLED event

**Expected Results**:
- ✅ DISABLED event received, `is_enabled = false` in cache
- ✅ ENABLED event received, `is_enabled = true` in cache

---

#### TC-F-005: Channel Deletion Synchronization

**Objective**: Verify Gateway removes deleted channels from cache

**Preconditions**:
- Test channel exists

**Steps**:
1. Delete channel:
   ```bash
   curl -X DELETE http://localhost:8080/v1/platform/channels/test-channel-id
   ```
2. Observe Gateway logs
3. Verify cache removal

**Expected Results**:
- ✅ DELETED event received
- ✅ Channel removed from Gateway cache
- ✅ Channel count decremented

---

### 4.2 Reliability Tests

#### TC-R-001: Gateway Reconnection After Disconnect

**Objective**: Verify Gateway auto-reconnects and resyncs after connection loss

**Steps**:
1. Establish initial connection
2. Stop Console API (simulate network failure)
3. Observe Gateway reconnection attempts
4. Restart Console API
5. Verify reconnection and resync

**Expected Results**:
- ✅ Gateway detects disconnection
- ✅ Reconnection attempts with exponential backoff
- ✅ Successful reconnection after Console API restart
- ✅ SNAPSHOT event received on reconnect
- ✅ Cache fully resynced

**Verification**:
```bash
# Gateway logs should show
grep "Connection lost" gateway.log
grep "Reconnecting" gateway.log
grep "Reconnected successfully" gateway.log
```

---

#### TC-R-002: Event Ordering Consistency

**Objective**: Verify events are processed in correct order

**Steps**:
1. Rapidly create 10 channels (within 1 second)
2. Verify all CREATED events received
3. Check event sequence numbers

**Expected Results**:
- ✅ All 10 CREATED events received
- ✅ Events processed in order (version numbers increment)
- ✅ No duplicate events

---

#### TC-R-003: Concurrent Client Connections

**Objective**: Verify multiple Gateway instances can subscribe simultaneously

**Steps**:
1. Start 3 Gateway instances
2. Create a new channel
3. Verify all 3 instances receive CREATED event

**Expected Results**:
- ✅ All 3 instances receive event
- ✅ Event data identical across instances
- ✅ No performance degradation

---

### 4.3 Performance Tests

#### TC-P-001: High-Frequency Event Throughput

**Objective**: Measure system performance under high event load

**Steps**:
1. Create 100 channels rapidly (via script)
2. Measure event delivery latency
3. Verify no events lost

**Expected Results**:
- ✅ All 100 CREATED events delivered
- ✅ Average latency < 100ms
- ✅ P99 latency < 500ms
- ✅ No memory leaks

**Test Script**:
```bash
#!/bin/bash
for i in {1..100}; do
  curl -X POST http://localhost:8080/v1/platform/channels \
    -H 'Content-Type: application/json' \
    -d "{\"name\":\"Perf Test $i\",\"provider\":\"openai\",\"api_key\":\"sk-test\"}" &
done
wait
```

---

#### TC-P-002: Long-Running Connection Stability

**Objective**: Verify connection remains stable over extended period

**Steps**:
1. Establish Gateway connection
2. Run for 24 hours
3. Periodically create/update/delete channels (every 5 minutes)
4. Monitor memory and CPU usage

**Expected Results**:
- ✅ Connection remains active for 24 hours
- ✅ All events delivered successfully
- ✅ Memory usage stable (no leaks)
- ✅ CPU usage < 5% average

---

### 4.4 Data Integrity Tests

#### TC-D-001: Field Encryption Validation

**Objective**: Verify sensitive fields are encrypted in transit

**Steps**:
1. Create channel with API key
2. Capture gRPC stream data (using Wireshark or grpcurl)
3. Verify `api_key_encrypted` field is encrypted

**Expected Results**:
- ✅ API key not visible in plaintext
- ✅ Encrypted field format matches encryption scheme

---

#### TC-D-002: Cache Consistency After Restart

**Objective**: Verify Gateway cache matches database after restart

**Steps**:
1. Gateway running with synced cache
2. Restart Gateway
3. Compare cache with database

**Expected Results**:
- ✅ SNAPSHOT event received on restart
- ✅ Cache matches database exactly
- ✅ No stale data

**Verification**:
```bash
# Database query
psql -c "SELECT id, name, provider FROM llm_system_channels ORDER BY id"

# Gateway cache
curl http://localhost:gateway-port/internal/channels | \
  jq '.data | sort_by(.id) | .[] | {id, name, provider}'
```

---

## 5. Test Execution

### 5.1 Manual Testing

#### Using grpcurl

```bash
# List available services
grpcurl -plaintext localhost:50051 list

# Call WatchChannels (streaming)
grpcurl -plaintext localhost:50051 \
  platform.v1.ChannelService/WatchChannels

# Expected output:
{
  "event_type": "SNAPSHOT",
  "channels": [...]
}
```

#### Using Test Client

```bash
# Run test client
cd /path/to/zgi-api
go run cmd/test_grpc_client/main.go

# Expected output:
Connected to gRPC server
Received SNAPSHOT event with 10 channels
Listening for events...
```

---

### 5.2 Automated Testing

#### Unit Tests

```bash
# Test event broadcaster
go test ./internal/infra/grpc/server/... -v -run TestEventBroadcaster

# Test client sync logic
go test ./internal/infra/platform/channel/... -v -run TestRemoteSync
```

#### Integration Tests

```bash
# Full E2E test
go test ./tests/integration/grpc_stream_test.go -v

# Expected:
=== RUN   TestGRPCStreamSync
--- PASS: TestGRPCStreamSync (5.23s)
```

---

### 5.3 Load Testing

```bash
# Use ghz for gRPC load testing
ghz --insecure \
  --proto=pkg/rpc/v1/channel.proto \
  --call=platform.v1.ChannelService.WatchChannels \
  --duration=60s \
  --connections=10 \
  localhost:50051

# Expected metrics:
# - RPS: > 1000
# - Latency P50: < 10ms
# - Latency P99: < 100ms
# - Error rate: 0%
```

---

## 6. Test Reporting

### 6.1 Test Metrics

| Metric | Target | Actual | Status |
|--------|--------|--------|--------|
| Event Delivery Latency (avg) | < 100ms | TBD | ⏳ |
| Event Delivery Latency (P99) | < 500ms | TBD | ⏳ |
| Reconnection Time | < 5s | TBD | ⏳ |
| Event Loss Rate | 0% | TBD | ⏳ |
| Memory Usage (24h) | < 500MB | TBD | ⏳ |
| CPU Usage (avg) | < 5% | TBD | ⏳ |

### 6.2 Test Coverage

| Component | Coverage Target | Actual | Status |
|-----------|----------------|--------|--------|
| gRPC Server | 80% | TBD | ⏳ |
| gRPC Client | 80% | TBD | ⏳ |
| Event Broadcaster | 90% | TBD | ⏳ |
| Cache Management | 85% | TBD | ⏳ |

---

## 7. Known Issues & Limitations

### 7.1 Current Limitations

1. **No Authentication on gRPC Stream**: Currently uses plaintext connection
   - **Impact**: Security risk in production
   - **Mitigation**: Use TLS + mTLS in production

2. **Single Event Broadcaster Instance**: Not distributed
   - **Impact**: Single point of failure
   - **Mitigation**: Future: Redis Pub/Sub for multi-instance

3. **No Event Persistence**: Events not stored
   - **Impact**: Events lost if no clients connected
   - **Mitigation**: Acceptable for current use case (SNAPSHOT on connect)

### 7.2 Future Enhancements

- [ ] Add gRPC authentication (JWT or mTLS)
- [ ] Implement distributed event bus (Redis Streams)
- [ ] Add event replay capability
- [ ] Implement event filtering (subscribe to specific providers)
- [ ] Add metrics and monitoring (Prometheus)

---

## 8. Test Sign-off

### 8.1 Test Completion Criteria

- [ ] All functional tests pass (TC-F-001 to TC-F-005)
- [ ] All reliability tests pass (TC-R-001 to TC-R-003)
- [ ] Performance targets met (TC-P-001 to TC-P-002)
- [ ] Data integrity verified (TC-D-001 to TC-D-002)
- [ ] No critical bugs outstanding
- [ ] Test coverage > 80%

### 8.2 Approval

| Role | Name | Date | Signature |
|------|------|------|-----------|
| Test Lead | TBD | TBD | |
| Tech Lead | TBD | TBD | |
| Product Owner | TBD | TBD | |

---

## 9. Appendix

### 9.1 Test Environment Configuration

```bash
# Console API .env
GRPC_PORT=50051
GRPC_ENABLED=true
DB_HOST=localhost
DB_PORT=5432
DB_NAME=zgi_console

# Gateway .env
GRPC_SERVER_ADDRESS=localhost:50051
GRPC_RECONNECT_INTERVAL=5s
CACHE_TTL=3600s
```

### 9.2 Useful Commands

```bash
# Check gRPC server status
grpcurl -plaintext localhost:50051 grpc.health.v1.Health/Check

# Monitor events in real-time
grpcurl -plaintext localhost:50051 \
  platform.v1.ChannelService/WatchChannels | \
  jq -c '.event_type, .channels[].name'

# Database verification
psql -U postgres -d zgi_console -c \
  "SELECT COUNT(*) FROM llm_system_channels WHERE is_system = true"
```

### 9.3 References

- [gRPC Documentation](https://grpc.io/docs/)
- [Protocol Buffers Guide](https://protobuf.dev/)
- [ZGI API Architecture](../AGENTS.md)
- [Console API Documentation](../CONSOLE_AGENTS.md)

---

**Document Version**: 1.0  
**Last Updated**: 2026-01-12  
**Author**: AI Coding Agent  
**Status**: Draft
