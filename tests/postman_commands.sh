#!/bin/bash
# Quick Postman-ready curl commands

# === USER OPERATIONS ===

# Create User
curl -X POST http://localhost:8090/api/collections/users/records \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"TestPassword123!","passwordConfirm":"TestPassword123!"}'

# Authenticate
curl -X POST http://localhost:8090/api/collections/users/auth-with-password \
  -H "Content-Type: application/json" \
  -d '{"identity":"test@example.com","password":"TestPassword123!"}'

# === AGENT OPERATIONS ===

# 1. Start Device Auth
curl -X POST http://localhost:8090/api/agent \
  -H "Content-Type: application/json" \
  -d '{"action":"device-auth-start"}'

# 2. Authorize Device (replace USER_TOKEN and USER_CODE)
curl -X POST http://localhost:8090/api/agent \
  -H "Authorization: Bearer USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"authorize","user_code":"USER_CODE"}'

# 3. Register Agent (replace DEVICE_CODE)
curl -X POST http://localhost:8090/api/agent \
  -H "Content-Type: application/json" \
  -d '{"action":"register","device_code":"DEVICE_CODE","hostname":"test-laptop","platform":"linux","version":"1.0.0"}'

# 4. Ingest Metrics (replace AGENT_ID from registration response)
curl -X POST http://localhost:8090/api/agent \
  -H "Content-Type: application/json" \
  -d '{"action":"ingest-metrics","access_token":"AGENT_ID","metrics":{"cpu_usage":45.5,"memory_usage":60.2}}'

# 5. Refresh Token (replace REFRESH_TOKEN)
curl -X POST http://localhost:8090/api/agent \
  -H "Content-Type: application/json" \
  -d '{"action":"refresh","refresh_token":"REFRESH_TOKEN"}'

# 6. List Agents (replace USER_TOKEN)
curl -X POST http://localhost:8090/api/agent \
  -H "Authorization: Bearer USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"list"}'

# 7. Revoke Agent (replace USER_TOKEN and AGENT_ID)
curl -X POST http://localhost:8090/api/agent \
  -H "Authorization: Bearer USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action":"revoke","agent_id":"AGENT_ID"}'

# 8. Health Check (replace AGENT_ID)
curl -X POST http://localhost:8090/api/agent \
  -H "Content-Type: application/json" \
  -d '{"action":"health","agent_id":"AGENT_ID"}'
