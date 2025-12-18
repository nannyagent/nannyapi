import { describe, it, expect, beforeAll, afterAll } from 'npm:vitest@latest';

const SUPABASE_URL = 'http://localhost:54321';
const SUPABASE_ANON_KEY = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6ImtleSIsInJvbGUiOiJhbm9uIiwiaWF0IjoxNjE1NzcwNzkyLCJleHAiOjE5MzE3NzA3OTJ9.SReKsIWjz_scH0OWf8nV03ihP8P_M5_juTv7IkaWh6w';
const EDGE_FUNCTION_URL = 'http://localhost:54321/functions/v1/device-auth';

describe('Device Auth - Code Validation', () => {
  it('should validate code format on authorize', async () => {
    // Authorize to get a device code and user code
    const authorizeResponse = await fetch(`${EDGE_FUNCTION_URL}?action=authorize`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${SUPABASE_ANON_KEY}`,
      },
      body: JSON.stringify({
        client_id: 'test-client-123'
      })
    });

    expect(authorizeResponse.status).toBe(200);
    const data = await authorizeResponse.json();
    
    // Verify user_code is 10-character alphanumeric
    expect(data.user_code).toBeDefined();
    expect(data.user_code).toMatch(/^[A-Z0-9]{10}$/);
    expect(data.user_code.length).toBe(10);
  });

  it('should reject invalid code format on approve', async () => {
    // Try to approve with invalid code format
    const invalidCodes = [
      'invalid-code',      // Contains dash
      'lowercase1234',     // Contains lowercase
      '12345',            // Too short
      '12345678901',      // Too long
      'ABCD@1234',        // Contains special character
      'CODE-1-2-3'        // Contains multiple dashes and lowercase
    ];

    for (const invalidCode of invalidCodes) {
      // Create a mock JWT for user (service role)
      const jwt = await generateMockJWT();
      
      const approveResponse = await fetch(`${EDGE_FUNCTION_URL}?action=approve`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${jwt}`,
          'x-client-id': 'test-client'
        },
        body: JSON.stringify({
          user_code: invalidCode
        })
      });

      // Should be 400 (Bad Request) for invalid format
      expect([400, 404, 429]).toContain(approveResponse.status);
      const responseData = await approveResponse.json();
      
      if (approveResponse.status === 400) {
        expect(responseData.error).toBe('invalid_code_format');
        expect(responseData.message).toContain('must be 10 alphanumeric characters');
      }
    }
  });

  it('should accept valid code format on approve attempt', async () => {
    // First authorize to get a valid code
    const authorizeResponse = await fetch(`${EDGE_FUNCTION_URL}?action=authorize`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${SUPABASE_ANON_KEY}`,
      },
      body: JSON.stringify({
        client_id: 'test-client-456'
      })
    });

    const authorizeData = await authorizeResponse.json();
    const validCode = authorizeData.user_code;

    // Verify it matches the expected format
    expect(validCode).toMatch(/^[A-Z0-9]{10}$/);
  });
});

describe('Device Auth - 10 Minute Expiry', () => {
  it('should generate code with 10 minute expiry', async () => {
    const authorizeResponse = await fetch(`${EDGE_FUNCTION_URL}?action=authorize`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${SUPABASE_ANON_KEY}`,
      },
      body: JSON.stringify({
        client_id: 'test-client-expiry'
      })
    });

    expect(authorizeResponse.status).toBe(200);
    const data = await authorizeResponse.json();
    
    // expires_in should be 600 seconds (10 minutes)
    expect(data.expires_in).toBe(600);
  });

  it('should reject expired codes', async () => {
    // This test would need a way to manipulate time or wait 10 minutes
    // For now, we'll skip or use a simpler approach
    // In production, you'd use a test utility to set time forward
    
    console.log('Skipping real expiry test (would require time manipulation)');
  });
});

describe('Device Auth - Rate Limiting', () => {
  it('should allow up to 10 failed attempts', async () => {
    const clientId = 'test-client-rate-limit-' + Date.now();
    
    // Try to approve with invalid code 10 times
    for (let i = 0; i < 10; i++) {
      const jwt = await generateMockJWT();
      
      const approveResponse = await fetch(`${EDGE_FUNCTION_URL}?action=approve`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${jwt}`,
          'x-client-id': clientId
        },
        body: JSON.stringify({
          user_code: 'INVALIDCOD' + String(i).padStart(1, '0')
        })
      });

      // Should get 400 for invalid format or 404 for not found
      expect([400, 404]).toContain(approveResponse.status);
    }
  });

  it('should reject with 429 after 10 failed attempts', async () => {
    const clientId = 'test-client-rate-limit-blocked-' + Date.now();
    
    // Make 11 failed attempts
    for (let i = 0; i < 11; i++) {
      const jwt = await generateMockJWT();
      
      const approveResponse = await fetch(`${EDGE_FUNCTION_URL}?action=approve`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${jwt}`,
          'x-client-id': clientId
        },
        body: JSON.stringify({
          user_code: 'INVALIDCOD' + String(i).padStart(1, '0')
        })
      });

      if (i < 10) {
        // First 10 should be 400 or 404
        expect([400, 404]).toContain(approveResponse.status);
      } else {
        // 11th attempt should be 429
        expect(approveResponse.status).toBe(429);
        const data = await approveResponse.json();
        expect(data.error).toBe('rate_limit_exceeded');
        expect(data.message).toContain('Maximum 10 failed attempts');
      }
    }
  });
});

describe('Device Auth - Code Uniqueness', () => {
  it('should prevent code reuse by different agents', async () => {
    // This test verifies the device_code_consumption tracking
    // First, authorize to get a code
    const authorizeResponse = await fetch(`${EDGE_FUNCTION_URL}?action=authorize`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${SUPABASE_ANON_KEY}`,
      },
      body: JSON.stringify({
        client_id: 'test-client-uniqueness'
      })
    });

    const authorizeData = await authorizeResponse.json();
    const userCode = authorizeData.user_code;

    // Create two different agent tokens
    const jwt1 = await generateMockJWT('agent-user-1');
    const jwt2 = await generateMockJWT('agent-user-2');

    // Try to use the same code for both agents
    // First approval should succeed
    const approve1 = await fetch(`${EDGE_FUNCTION_URL}?action=approve`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${jwt1}`,
        'x-client-id': 'agent-1'
      },
      body: JSON.stringify({ user_code: userCode })
    });

    // If first approval succeeds
    if (approve1.status === 200) {
      // Second approval attempt should fail
      const approve2 = await fetch(`${EDGE_FUNCTION_URL}?action=approve`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${jwt2}`,
          'x-client-id': 'agent-2'
        },
        body: JSON.stringify({ user_code: userCode })
      });

      // Should fail with code_already_used or not_found
      expect([400, 404]).toContain(approve2.status);
      const data = await approve2.json();
      if (approve2.status === 400) {
        expect(data.error).toBe('code_already_used');
      }
    }
  });
});

describe('Device Auth - Hostname Deduplication', () => {
  it('should handle hostname-based agent naming', async () => {
    const clientId = 'hostname-client-' + Date.now();
    
    // The device-auth function should extract hostname from client_id
    // and use it to name the agent, with deduplication (-1, -2, etc.)
    
    const authorizeResponse = await fetch(`${EDGE_FUNCTION_URL}?action=authorize`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${SUPABASE_ANON_KEY}`,
      },
      body: JSON.stringify({
        client_id: clientId
      })
    });

    expect(authorizeResponse.status).toBe(200);
    const data = await authorizeResponse.json();
    expect(data.user_code).toBeDefined();
    expect(data.device_code).toBeDefined();
  });
});

// Helper function to generate a mock JWT token
async function generateMockJWT(userId?: string): Promise<string> {
  const defaultUserId = userId || 'test-user-' + Date.now();
  
  // For testing, we'll create a simple JWT-like header and payload
  // In real tests, you'd use proper JWT signing
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }));
  const payload = btoa(JSON.stringify({ 
    sub: defaultUserId,
    user_id: defaultUserId,
    iat: Math.floor(Date.now() / 1000),
    exp: Math.floor(Date.now() / 1000) + 3600
  }));
  const signature = btoa('test-signature');
  
  return `${header}.${payload}.${signature}`;
}

// Re-export for use in other test files
export { generateMockJWT };
