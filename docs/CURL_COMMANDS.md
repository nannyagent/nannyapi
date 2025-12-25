# NannyAPI - Working curl Commands for Postman Testing

## 1. Create User
```bash
curl -X POST http://localhost:8090/api/collections/users/records \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"TestPassword123!","passwordConfirm":"TestPassword123!"}'
```

## 2. Authenticate User
```bash
curl -X POST http://localhost:8090/api/collections/users/auth-with-password \
  -H "Content-Type: application/json" \
  -d '{"identity":"user@example.com","password":"TestPassword123!"}'
```
Returns: `{"token":"...","record":{...}}`

## 3. Update Password
```bash
curl -X PATCH http://localhost:8090/api/collections/users/records/USER_ID \
  -H "Authorization: Bearer USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"oldPassword":"TestPassword123!","password":"NewPassword123!","passwordConfirm":"NewPassword123!"}'
```

## 4. Delete User
```bash
curl -X DELETE http://localhost:8090/api/collections/users/records/USER_ID \
  -H "Authorization: Bearer USER_TOKEN"
```

## 5. Start Device Auth
```bash
curl -X POST http://localhost:8090/api/agent \
  -H "Content-Type: application/json" \
  -d '{"action":"device-auth-start"}'
```
Returns: `{"device_code":"...","user_code":"...","verification_uri":"...","expires_in":600}`

## 6. Authorize Device (requires user token)
```bash
curl -X POST http://localhost:8090/api/agent \
  -H "Authorization: Bearer USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"authorize","user_code":"USER_CODE"}'
```

## 7. Register Agent
```bash
curl -X POST http://localhost:8090/api/agent \
  -H "Content-Type: application/json" \
  -d '{
    "action":"register",
    "device_code":"DEVICE_CODE",
    "hostname":"my-laptop",
    "platform":"linux",
    "version":"1.0.0",
    "primary_ip":"192.168.1.100",
    "all_ips":["192.168.1.100","10.0.0.5"]
  }'
```
Returns: `{"agent_id":"...","access_token":"...","refresh_token":"...","expires_in":3600}`

## 8. Ingest Metrics (Bearer token in header)
```bash
curl -X POST http://localhost:8090/api/agent \
  -H "Authorization: Bearer AGENT_ID" \
  -H "Content-Type: application/json" \
  -d '{
    "action":"ingest-metrics",
    "metrics":{
      "cpu_usage":75.5,
      "memory_usage":82.3,
      "network_rx_bytes":5242880,
      "network_tx_bytes":2621440,
      "memory_used_gb":12.5,
      "memory_total_gb":16.0,
      "disk_used_gb":450.0,
      "disk_total_gb":500.0
    }
  }'
```

## 9. Refresh Token
```bash
curl -X POST http://localhost:8090/api/agent \
  -H "Content-Type: application/json" \
  -d '{"action":"refresh","refresh_token":"REFRESH_TOKEN"}'
```

## 10. List Agents (requires user token)
```bash
curl -X POST http://localhost:8090/api/agent \
  -H "Authorization: Bearer USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"list"}'
```

## 11. Revoke Agent (requires user token)
```bash
curl -X POST http://localhost:8090/api/agent \
  -H "Authorization: Bearer USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"revoke","agent_id":"AGENT_ID"}'
```

## 12. Check Agent Health
```bash
curl -X POST http://localhost:8090/api/agent \
  -H "Content-Type: application/json" \
  -d '{"action":"health","agent_id":"AGENT_ID"}'
```

## Complete Flow Example
```bash
# 1. Create and auth user
EMAIL="test@example.com"
USER_ID=$(curl -s -X POST http://localhost:8090/api/collections/users/records \
  -H "Content-Type: application/json" \
  -d '{"email":"'$EMAIL'","password":"TestPassword123!","passwordConfirm":"TestPassword123!"}' | jq -r '.id')

USER_TOKEN=$(curl -s -X POST http://localhost:8090/api/collections/users/auth-with-password \
  -H "Content-Type: application/json" \
  -d '{"identity":"'$EMAIL'","password":"TestPassword123!"}' | jq -r '.token')

# 2. Device auth flow
DEVICE_RESPONSE=$(curl -s -X POST http://localhost:8090/api/agent \
  -H "Content-Type: application/json" \
  -d '{"action":"device-auth-start"}')

USER_CODE=$(echo "$DEVICE_RESPONSE" | jq -r '.user_code')
DEVICE_CODE=$(echo "$DEVICE_RESPONSE" | jq -r '.device_code')

# 3. User authorizes
curl -s -X POST http://localhost:8090/api/agent \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"authorize","user_code":"'$USER_CODE'"}'

# 4. Agent registers
REGISTER_RESPONSE=$(curl -s -X POST http://localhost:8090/api/agent \
  -H "Content-Type: application/json" \
  -d '{"action":"register","device_code":"'$DEVICE_CODE'","hostname":"test","platform":"linux","version":"1.0.0"}')

AGENT_ID=$(echo "$REGISTER_RESPONSE" | jq -r '.agent_id')

# 5. Send metrics
curl -s -X POST http://localhost:8090/api/agent \
  -H "Content-Type: application/json" \
  -d '{"action":"ingest-metrics","access_token":"'$AGENT_ID'","metrics":{"cpu_usage":45.5}}'
```
