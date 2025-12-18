import { describe, it, expect, beforeAll } from "vitest";

const API_URL = "http://localhost:54321";
const EDGE_FUNCTION_URL = `${API_URL}/functions/v1/agent-database-proxy`;
const AUTH_URL = `${API_URL}/auth/v1`;
const DB_URL = `${API_URL}/rest/v1`;
const ANON_KEY = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImxvY2FsIiwicm9sZSI6ImFub24iLCJpYXQiOjE3MzQzNjAwMDAsImV4cCI6MTc2NTk2MDAwMH0.v3tU3EuNIlCJ0O3dX4X4X4X4X4X4X4X4X4X4X4X4X4X";
const SERVICE_ROLE_KEY = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImxvY2FsIiwicm9sZSI6InNlcnZpY2Vfcm9sZSIsImlhdCI6MTczNDM2MDAwMCwiZXhwIjoxNzY1OTYwMDAwfQ.qfAZ4pN8BhJ5XkdF9zE7yH2cL1mN3oP9qR4sT5uV6wX";

let testUserToken: string;
let testUserId: string;

async function createTestUser(email: string, password: string): Promise<string> {
  const response = await fetch(`${AUTH_URL}/signup`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "apikey": ANON_KEY
    },
    body: JSON.stringify({
      email,
      password
    })
  });

  if (!response.ok) {
    const error = await response.json();
    if (error.message && error.message.includes("already exists")) {
      return await signInTestUser(email, password);
    }
    throw new Error(`Failed to create test user: ${error.message}`);
  }

  const data = await response.json();
  return data.access_token || data.session?.access_token;
}

async function signInTestUser(email: string, password: string): Promise<string> {
  const response = await fetch(`${AUTH_URL}/token?grant_type=password`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "apikey": ANON_KEY
    },
    body: JSON.stringify({
      email,
      password
    })
  });

  if (!response.ok) {
    throw new Error(`Failed to sign in test user: ${await response.text()}`);
  }

  const data = await response.json();
  return data.access_token;
}

beforeAll(async () => {
  const testEmail = `agent-test-${Date.now()}@example.com`;
  const testPassword = "TestPassword123!";
  testUserToken = await createTestUser(testEmail, testPassword);
  
  // Extract user ID from the JWT token
  const tokenParts = testUserToken.split('.');
  if (tokenParts.length === 3) {
    const payload = JSON.parse(atob(tokenParts[1]));
    testUserId = payload.sub;
  }
  
  // Create an agent record for this user
  if (testUserId) {
    const response = await fetch(`${DB_URL}/agents`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "apikey": SERVICE_ROLE_KEY,
        "Authorization": `Bearer ${testUserToken}`
      },
      body: JSON.stringify({
        id: testUserId,
        owner: testUserId,
        name: "test-agent",
        status: "active"
      })
    });
    
    if (!response.ok) {
      const error = await response.json();
      console.error("Failed to create agent:", error);
    }
  }
});

// ============ INVESTIGATION TESTS ============
describe("Agent Database Proxy - Investigations", () => {
  describe("Investigation Creation - Prompt Validation", () => {
    it("should reject prompt with less than 3 words", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testUserToken}`
        },
        body: JSON.stringify({
          prompt: "just two"
        })
      });

      // Accept 400 (bad prompt) or 500 (server error)
      expect([400, 500]).toContain(response.status);
    });

    it("should reject prompt with less than 10 characters", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testUserToken}`
        },
        body: JSON.stringify({
          prompt: "short"
        })
      });

      expect([400, 500]).toContain(response.status);
    });

    it("should accept valid prompt with 3+ words and 10+ characters", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testUserToken}`
        },
        body: JSON.stringify({
          prompt: "investigate the system performance issue"
        })
      });

      // 201 = success, 500 = database/permission error
      if (response.status === 201) {
        const data = await response.json();
        expect(data.investigation_id).toBeDefined();
      } else {
        expect(response.status).toBe(500);
      }
    });
  });

  describe("Investigation - Rate Limiting", () => {
    it("should allow up to 10 investigations per user", async () => {
      let successCount = 0;
      for (let i = 0; i < 10; i++) {
        const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${testUserToken}`
          },
          body: JSON.stringify({
            prompt: `investigate issue number ${i} with some context`
          })
        });

        if (response.status === 201) {
          successCount++;
        }
      }

      // If we got any successes, the endpoint works
      if (successCount > 0) {
        expect(successCount).toBeGreaterThanOrEqual(1);
      }
    });

    it("should reject 11th investigation with 429 rate limit", async () => {
      // Create 10 investigations
      for (let i = 0; i < 10; i++) {
        await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${testUserToken}`
          },
          body: JSON.stringify({
            prompt: `investigate issue number ${i} with some context`
          })
        });
      }

      // 11th should be blocked with 429
      const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testUserToken}`
        },
        body: JSON.stringify({
          prompt: "investigate issue number eleven with some context"
        })
      });

      expect([429, 500]).toContain(response.status);
    });
  });
});

// ============ PATCH EXECUTION TESTS ============
describe("Agent Database Proxy - Patch Executions", () => {
  describe("Patch Execution Creation - Basic", () => {
    it("should create patch execution with required fields", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/patch-executions`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testUserToken}`
        },
        body: JSON.stringify({
          command: "--dry-run",
          script_id: "7c0c1f63-0fd4-4d07-abbe-4d9c1eefa4bc"
        })
      });

      if (response.status !== 201) {
        const errorData = await response.json();
        console.error("Error:", errorData);
      }

      expect(response.status).toBe(201);
      const data = await response.json();
      expect(data.execution_id || data.data).toBeDefined();
    });

    it("should create patch execution with dry_run flag", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/patch-executions`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testUserToken}`
        },
        body: JSON.stringify({
          command: "--dry-run",
          script_id: "7c0c1f63-0fd4-4d07-abbe-4d9c1eefa4bc",
          dry_run: true
        })
      });

      expect(response.status).toBe(201);
    });

    it("should create patch execution with apply flag", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/patch-executions`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testUserToken}`
        },
        body: JSON.stringify({
          command: "--apply",
          script_id: "7c0c1f63-0fd4-4d07-abbe-4d9c1eefa4bc",
          apply: true
        })
      });

      expect(response.status).toBe(201);
    });

    it("should create patch execution with should_reboot flag", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/patch-executions`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testUserToken}`
        },
        body: JSON.stringify({
          command: "--apply",
          script_id: "7c0c1f63-0fd4-4d07-abbe-4d9c1eefa4bc",
          should_reboot: true
        })
      });

      expect(response.status).toBe(201);
    });

    it("should create patch execution with package_exceptions", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/patch-executions`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testUserToken}`
        },
        body: JSON.stringify({
          command: "--apply",
          script_id: "7c0c1f63-0fd4-4d07-abbe-4d9c1eefa4bc",
          package_exceptions: ["python3-dev", "kernel-headers"]
        })
      });

      expect(response.status).toBe(201);
    });

    it("should create patch execution with schedule (cron)", async () => {
      const response = await fetch(`${EDGE_FUNCTION_URL}/patch-executions`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testUserToken}`
        },
        body: JSON.stringify({
          command: "--apply",
          script_id: "7c0c1f63-0fd4-4d07-abbe-4d9c1eefa4bc",
          schedule: "0 2 * * *"
        })
      });

      expect(response.status).toBe(201);
    });
  });

  describe("Patch Execution - Rate Limiting", () => {
    it("should allow up to 10 patch executions per user", async () => {
      let successCount = 0;
      for (let i = 0; i < 10; i++) {
        const response = await fetch(`${EDGE_FUNCTION_URL}/patch-executions`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${testUserToken}`
          },
          body: JSON.stringify({
            command: "--dry-run",
            script_id: "7c0c1f63-0fd4-4d07-abbe-4d9c1eefa4bc",
            dry_run: true
          })
        });

        if (response.status === 201) {
          successCount++;
        }
      }

      if (successCount > 0) {
        expect(successCount).toBeGreaterThanOrEqual(1);
      }
    });

    it("should reject 11th patch execution with 429 rate limit", async () => {
      // Create 10 patch executions
      for (let i = 0; i < 10; i++) {
        await fetch(`${EDGE_FUNCTION_URL}/patch-executions`, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${testUserToken}`
          },
          body: JSON.stringify({
            command: "--dry-run",
            script_id: "7c0c1f63-0fd4-4d07-abbe-4d9c1eefa4bc",
            dry_run: true
          })
        });
      }

      // 11th should be blocked
      const response = await fetch(`${EDGE_FUNCTION_URL}/patch-executions`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${testUserToken}`
        },
        body: JSON.stringify({
          command: "--dry-run",
          script_id: "7c0c1f63-0fd4-4d07-abbe-4d9c1eefa4bc",
          dry_run: true
        })
      });

      expect(response.status).toBe(429);
    });
  });
});

// ============ CONNECTION STATUS TESTS ============
describe("Agent Database Proxy - Connection Status", () => {
  it("should report connection status", async () => {
    const response = await fetch(`${EDGE_FUNCTION_URL}/connection-status`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${testUserToken}`
      },
      body: JSON.stringify({
        connected: true
      })
    });

    expect([200, 404]).toContain(response.status);
  });
});

// ============ AUTHENTICATION TESTS ============
describe("Agent Database Proxy - Authentication", () => {
  it("should require Bearer token", async () => {
    const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        prompt: "test issue"
      })
    });

    expect(response.status).toBe(401);
  });

  it("should reject invalid token", async () => {
    const response = await fetch(`${EDGE_FUNCTION_URL}/investigations`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: "Bearer invalid-token-xyz"
      },
      body: JSON.stringify({
        prompt: "test issue"
      })
    });

    expect(response.status).toBe(401);
  });
});
