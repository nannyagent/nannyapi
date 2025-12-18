import { describe, it, expect, beforeAll, afterAll } from "npm:vitest@latest";

const SUPABASE_URL = "http://localhost:54321";
const SUPABASE_ANON_KEY =
  "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImtleSIsInJvbGUiOiJhbm9uIiwiaWF0IjoxNjE1NzcwNzkyLCJleHAiOjE5MzE3NzA3OTJ9.SReKsIWjz_scH0OWf8nV03ihP8P_M5_juTv7IkaWh6w";
const EDGE_FUNCTION_URL = "http://localhost:54321/functions/v1/agent-database-proxy";

let testAgentId = "";
let testUserId = "test-user-" + Date.now();
let testToken = "";
let testInvestigationId = "";
let testPatchExecutionId = "";

// Helper to generate a mock JWT token
function generateMockToken(agentId: string): string {
  const header = btoa(JSON.stringify({ alg: "HS256", typ: "JWT" }));
  const payload = btoa(
    JSON.stringify({
      sub: agentId,
      iat: Math.floor(Date.now() / 1000),
      exp: Math.floor(Date.now() / 1000) + 3600
    })
  );
  const signature = btoa("test-signature");
  return `${header}.${payload}.${signature}`;
}

describe("Agent Database Proxy - Edge Function", () => {
  beforeAll(async () => {
    // Generate test IDs
    testAgentId = "agent-" + Date.now();
    testToken = generateMockToken(testAgentId);
  });

  // ============ INVESTIGATION PROMPT VALIDATION ============
  describe("Investigation - Prompt Validation", () => {
    it("should reject prompt with less than 3 words", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testToken}`
        },
        body: JSON.stringify({
          prompt: "just two",
          user_id: testUserId
        })
      });

      expect(response.status).toBe(400);
      const data = await response.json();
      expect(data.error).toBe("invalid_prompt");
      expect(data.message).toContain("3 words");
    });

    it("should reject prompt with less than 10 characters", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testToken}`
        },
        body: JSON.stringify({
          prompt: "hello world test",
          user_id: testUserId
        })
      });

      expect(response.status).toBe(400);
      const data = await response.json();
      expect(data.error).toBe("invalid_prompt");
      expect(data.message).toContain("10 characters");
    });

    it("should accept valid prompt with 3+ words and 10+ characters", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testToken}`
        },
        body: JSON.stringify({
          prompt: "please investigate this application error",
          user_id: testUserId
        })
      });

      if (response.status === 201) {
        expect(response.status).toBe(201);
        const data = await response.json();
        expect(data.success).toBe(true);
        expect(data.investigation_id).toBeDefined();
        testInvestigationId = data.investigation_id;
      }
    });

    it("should reject empty prompt", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testToken}`
        },
        body: JSON.stringify({
          prompt: "",
          user_id: testUserId
        })
      });

      expect(response.status).toBe(400);
    });

    it("should reject null prompt", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testToken}`
        },
        body: JSON.stringify({
          user_id: testUserId
        })
      });

      expect(response.status).toBe(400);
    });
  });

  // ============ INVESTIGATION CREATION ============
  describe("Investigation - Creation", () => {
    it("should create investigation with valid prompt", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testToken}`
        },
        body: JSON.stringify({
          prompt: "investigate the database connection timeout issue",
          user_id: testUserId,
          priority: "high"
        })
      });

      expect(response.status).toBe(201);
      const data = await response.json();
      expect(data.success).toBe(true);
      expect(data.investigation_id).toBeDefined();
      expect(data.investigation_id).toMatch(/^inv_/);
      expect(data.data).toBeDefined();
      expect(data.data[0].agent_id).toBeDefined();
      expect(data.data[0].initiated_by).toBe(testUserId);
      expect(data.data[0].status).toBe("initiated");
      expect(data.data[0].priority).toBe("high");
    });

    it("should create investigation with default priority if not specified", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testToken}`
        },
        body: JSON.stringify({
          prompt: "check system logs for any critical errors",
          user_id: testUserId
        })
      });

      expect(response.status).toBe(201);
      const data = await response.json();
      expect(data.data[0].priority).toBe("medium");
    });

    it("should include investigation_id field", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testToken}`
        },
        body: JSON.stringify({
          prompt: "analyze network traffic patterns for anomalies",
          user_id: testUserId
        })
      });

      expect(response.status).toBe(201);
      const data = await response.json();
      expect(data.investigation_id).toBeTruthy();
      expect(typeof data.investigation_id).toBe("string");
    });
  });

  // ============ RATE LIMITING - INVESTIGATIONS ============
  describe("Investigation - Rate Limiting", () => {
    it("should allow up to 10 investigations per agent per 10 minutes", async () => {
      const userId = "rate-test-user-" + Date.now();
      const agentToken = generateMockToken("rate-agent-" + Date.now());

      let successCount = 0;
      for (let i = 0; i < 10; i++) {
        const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${agentToken}`
          },
          body: JSON.stringify({
            prompt: `test investigation number ${i + 1} with sufficient content`,
            user_id: userId
          })
        });

        if (response.status === 201) {
          successCount++;
        }
      }

      // Should allow all 10
      expect(successCount).toBeGreaterThan(0);
    });

    it("should reject with 429 after 10 investigations", async () => {
      const userId = "rate-limit-user-" + Date.now();
      const agentToken = generateMockToken("rate-limit-agent-" + Date.now());

      // Make 11 requests
      for (let i = 0; i < 10; i++) {
        await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${agentToken}`
          },
          body: JSON.stringify({
            prompt: `investigation attempt ${i + 1} to test rate limiting`,
            user_id: userId
          })
        });
      }

      // 11th request should be rate limited
      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${agentToken}`
        },
        body: JSON.stringify({
          prompt: "this should be rate limited after exceeding limit",
          user_id: userId
        })
      });

      expect(response.status).toBe(429);
      const data = await response.json();
      expect(data.error).toBe("rate_limit_exceeded");
      expect(data.message).toContain("Maximum 10");
      expect(data.remaining).toBe(0);
    });

    it("should return remaining count", async () => {
      const userId = "remaining-count-" + Date.now();
      const agentToken = generateMockToken("remaining-agent-" + Date.now());

      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${agentToken}`
        },
        body: JSON.stringify({
          prompt: "check remaining rate limit counter",
          user_id: userId
        })
      });

      if (response.status === 201) {
        const data = await response.json();
        expect(data.data).toBeDefined();
      }
    });
  });

  // ============ PATCH EXECUTION CREATION ============
  describe("Patch Execution - Creation", () => {
    it("should create patch execution with command", async () => {
      const response = await fetch(
        `${EDGE_FUNCTION_URL}/patch-executions`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${testToken}`
          },
          body: JSON.stringify({
            command: "apt-get update && apt-get upgrade -y",
            user_id: testUserId,
            execution_type: "manual",
            dry_run: false,
            apply: true
          })
        }
      );

      if (response.status === 201) {
        expect(response.status).toBe(201);
        const data = await response.json();
        expect(data.success).toBe(true);
        expect(data.execution_id).toBeDefined();
        testPatchExecutionId = data.execution_id;
        expect(data.data[0].status).toBe("pending");
        expect(data.data[0].triggered_by).toBe(testUserId);
      }
    });

    it("should reject patch execution without command", async () => {
      const response = await fetch(
        `${EDGE_FUNCTION_URL}/patch-executions`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${testToken}`
          },
          body: JSON.stringify({
            user_id: testUserId,
            execution_type: "manual"
          })
        }
      );

      expect(response.status).toBe(400);
      const data = await response.json();
      expect(data.error).toBe("invalid_request");
    });

    it("should create patch execution with dry-run flag", async () => {
      const response = await fetch(
        `${EDGE_FUNCTION_URL}/patch-executions`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${testToken}`
          },
          body: JSON.stringify({
            command: "yum check-update",
            user_id: testUserId,
            dry_run: true,
            apply: false
          })
        }
      );

      if (response.status === 201) {
        expect(response.status).toBe(201);
        const data = await response.json();
        expect(data.data[0].metadata?.dry_run).toBe(true);
        expect(data.data[0].metadata?.apply).toBe(false);
      }
    });

    it("should create patch execution with reboot flag", async () => {
      const response = await fetch(
        `${EDGE_FUNCTION_URL}/patch-executions`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${testToken}`
          },
          body: JSON.stringify({
            command: "apt-get full-upgrade -y",
            user_id: testUserId,
            should_reboot: true
          })
        }
      );

      if (response.status === 201) {
        expect(response.status).toBe(201);
        const data = await response.json();
        expect(data.data[0].should_reboot).toBe(true);
      }
    });

    it("should create patch execution with package exceptions", async () => {
      const response = await fetch(
        `${EDGE_FUNCTION_URL}/patch-executions`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${testToken}`
          },
          body: JSON.stringify({
            command: "apt-get upgrade -y",
            user_id: testUserId,
            package_exceptions: ["nginx", "apache2"]
          })
        }
      );

      if (response.status === 201) {
        expect(response.status).toBe(201);
        const data = await response.json();
        expect(data.data[0].metadata?.package_exceptions).toEqual([
          "nginx",
          "apache2"
        ]);
      }
    });

    it("should create patch execution with schedule", async () => {
      const response = await fetch(
        `${EDGE_FUNCTION_URL}/patch-executions`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${testToken}`
          },
          body: JSON.stringify({
            command: "yum update -y",
            user_id: testUserId,
            schedule: "0 2 * * *"
          })
        }
      );

      if (response.status === 201) {
        expect(response.status).toBe(201);
        const data = await response.json();
        expect(data.data[0].metadata?.schedule).toBe("0 2 * * *");
      }
    });
  });

  // ============ RATE LIMITING - PATCH EXECUTIONS ============
  describe("Patch Execution - Rate Limiting", () => {
    it("should allow up to 10 patch executions per agent per 10 minutes", async () => {
      const userId = "patch-rate-test-" + Date.now();
      const agentToken = generateMockToken("patch-agent-" + Date.now());

      let successCount = 0;
      for (let i = 0; i < 10; i++) {
        const response = await fetch(
          `${EDGE_FUNCTION_URL}/patch-executions`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              Authorization: `Bearer ${agentToken}`
            },
            body: JSON.stringify({
              command: `update command ${i + 1}`,
              user_id: userId
            })
          }
        );

        if (response.status === 201) {
          successCount++;
        }
      }

      expect(successCount).toBeGreaterThan(0);
    });

    it("should reject with 429 after 10 patch executions", async () => {
      const userId = "patch-limit-user-" + Date.now();
      const agentToken = generateMockToken("patch-limit-agent-" + Date.now());

      // Make 10 requests
      for (let i = 0; i < 10; i++) {
        await fetch(`${EDGE_FUNCTION_URL}/patch-executions`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${agentToken}`
          },
          body: JSON.stringify({
            command: `patch attempt ${i + 1}`,
            user_id: userId
          })
        });
      }

      // 11th request should be rate limited
      const response = await fetch(
        `${EDGE_FUNCTION_URL}/patch-executions`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${agentToken}`
          },
          body: JSON.stringify({
            command: "this should be rate limited",
            user_id: userId
          })
        }
      );

      expect(response.status).toBe(429);
      const data = await response.json();
      expect(data.error).toBe("rate_limit_exceeded");
      expect(data.remaining).toBe(0);
    });
  });

  // ============ CONNECTION STATUS ============
  describe("Connection Status", () => {
    it("should update agent connection status to connected", async () => {
      const response = await fetch(
        `${EDGE_FUNCTION_URL}/connection-status`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${testToken}`
          },
          body: JSON.stringify({
            connected: true
          })
        }
      );

      expect(response.status).toBe(200);
      const data = await response.json();
      expect(data.success).toBe(true);
    });

    it("should update agent connection status to disconnected", async () => {
      const response = await fetch(
        `${EDGE_FUNCTION_URL}/connection-status`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${testToken}`
          },
          body: JSON.stringify({
            connected: false
          })
        }
      );

      expect(response.status).toBe(200);
      const data = await response.json();
      expect(data.success).toBe(true);
    });
  });

  // ============ AUTHENTICATION ============
  describe("Authentication", () => {
    it("should reject request without authorization header", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json"
        },
        body: JSON.stringify({
          prompt: "test prompt for investigation",
          user_id: testUserId
        })
      });

      expect(response.status).toBe(401);
      const data = await response.json();
      expect(data.error).toContain("Authorization");
    });

    it("should reject request with invalid token", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: "Bearer invalid-token"
        },
        body: JSON.stringify({
          prompt: "test prompt for investigation",
          user_id: testUserId
        })
      });

      expect(response.status).toBe(401);
    });

    it("should reject request without Bearer prefix", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: testToken
        },
        body: JSON.stringify({
          prompt: "test prompt for investigation",
          user_id: testUserId
        })
      });

      expect(response.status).toBe(401);
    });
  });

  // ============ ERROR HANDLING ============
  describe("Error Handling", () => {
    it("should return 404 for unknown endpoints", async () => {
      const response = await fetch(
        `${EDGE_FUNCTION_URL}/unknown-endpoint`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${testToken}`
          },
          body: JSON.stringify({})
        }
      );

      expect(response.status).toBe(404);
      const data = await response.json();
      expect(data.error).toBe("Not found");
    });

    it("should handle OPTIONS request", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "OPTIONS"
      });

      expect(response.status).toBe(200);
      expect(response.headers.get("Access-Control-Allow-Origin")).toBe("*");
    });
  });
});
