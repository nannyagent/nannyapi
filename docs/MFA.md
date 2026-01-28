# Multi-Factor Authentication (MFA) Documentation

NannyAPI supports TOTP-based Multi-Factor Authentication for enhanced security. This document covers the MFA API endpoints, enrollment flow, and usage.

## Overview

MFA adds an extra layer of security by requiring users to provide a time-based one-time password (TOTP) from an authenticator app in addition to their password.

### Features

- **TOTP Support**: Compatible with authenticator apps like Google Authenticator, Authy, Microsoft Authenticator
- **QR Code Enrollment**: Easy setup by scanning a QR code
- **Backup Codes**: 10 one-time recovery codes for emergency access
- **Replay Prevention**: Each TOTP code can only be used once
- **Sensitive Operations**: Additional verification for critical actions
- **AAL Levels**: Authenticator Assurance Levels (aal1 for password, aal2 for MFA verified)

## API Endpoints

### 1. Enroll in MFA

Start the MFA enrollment process by generating a TOTP secret and QR code.

```bash
POST /api/mfa/enroll
Authorization: Bearer <token>
Content-Type: application/json

{
  "factor_type": "totp",
  "friendly_name": "My Phone"  // optional
}
```

**Response:**
```json
{
  "factor_id": "abc123...",
  "qr_code": "data:image/png;base64,...",
  "secret": "JBSWY3DPEHPK3PXP",
  "totp_uri": "otpauth://totp/NannyAPI:user@example.com?secret=..."
}
```

The user should scan the QR code with their authenticator app or manually enter the secret.

### 2. Verify Enrollment

Complete enrollment by verifying a TOTP code from the authenticator app.

```bash
POST /api/mfa/enroll/verify
Authorization: Bearer <token>
Content-Type: application/json

{
  "factor_id": "abc123...",
  "code": "123456"
}
```

**Response:**
```json
{
  "success": true,
  "backup_codes": [
    "ABCD-1234",
    "EFGH-5678",
    ...
  ]
}
```

**Important:** Save the backup codes securely! They can only be viewed once.

### 3. List MFA Factors

Get all enrolled MFA factors for the authenticated user.

```bash
GET /api/mfa/factors
Authorization: Bearer <token>
```

**Response:**
```json
{
  "factors": [
    {
      "id": "abc123...",
      "factor_type": "totp",
      "friendly_name": "My Phone",
      "status": "verified",
      "created": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### 4. Create Challenge

Create an MFA challenge for verification during login or sensitive operations.

```bash
POST /api/mfa/challenge
Authorization: Bearer <token>
Content-Type: application/json

{
  "factor_id": "abc123..."
}
```

**Response:**
```json
{
  "challenge_id": "xyz789..."
}
```

### 5. Verify MFA

Verify the MFA challenge with a TOTP code or backup code.

```bash
POST /api/mfa/verify
Authorization: Bearer <token>
Content-Type: application/json

{
  "challenge_id": "xyz789...",
  "code": "123456"
}
```

**Response:**
```json
{
  "success": true,
  "assurance_level": "aal2"
}
```

### 6. Unenroll (Disable MFA)

Remove an MFA factor. Requires valid TOTP code for security.

```bash
POST /api/mfa/unenroll
Authorization: Bearer <token>
Content-Type: application/json

{
  "factor_id": "abc123...",
  "code": "123456"
}
```

### 7. Get Backup Codes

Retrieve unused backup codes (must already have MFA enabled).

```bash
POST /api/mfa/backup-codes
Authorization: Bearer <token>
```

**Response:**
```json
{
  "backup_codes": [
    {"code": "ABCD-****", "used": false},
    {"code": "EFGH-****", "used": true},
    ...
  ],
  "unused_count": 8
}
```

Note: Backup codes are partially masked for security.

### 8. Regenerate Backup Codes

Generate new backup codes, invalidating all previous ones.

```bash
POST /api/mfa/backup-codes/regenerate
Authorization: Bearer <token>
Content-Type: application/json

{
  "code": "123456"  // Current TOTP code required
}
```

**Response:**
```json
{
  "backup_codes": [
    "WXYZ-9876",
    "MNOP-5432",
    ...
  ]
}
```

### 9. Get Assurance Level

Check the current authentication assurance level.

```bash
GET /api/mfa/assurance-level
Authorization: Bearer <token>
```

**Response:**
```json
{
  "current_level": "aal1",
  "next_level": "aal2",
  "mfa_enabled": true
}
```

### 10. Verify Sensitive Operation (Future-Ready)

Initiate MFA verification for sensitive operations.

```bash
POST /api/mfa/verify-sensitive
Authorization: Bearer <token>
Content-Type: application/json

{
  "operation": "delete_account",
  "code": "123456"
}
```

**Response:**
```json
{
  "verification_id": "ver123...",
  "expires_at": "2024-01-15T10:35:00Z"
}
```

### 11. Check Sensitive Verification

Verify that a sensitive operation verification is still valid.

```bash
GET /api/mfa/verify-sensitive/{verificationId}
Authorization: Bearer <token>
```

## Authentication Flow with MFA

### Login Flow

1. User submits email/password to `/api/collections/users/auth-with-password`
2. If MFA is enabled, server returns:
   ```json
   {
     "mfa_required": true,
     "challenge_id": "xyz789...",
     "factor_id": "abc123..."
   }
   ```
3. User submits TOTP code to `/api/mfa/verify`
4. On success, user receives full authentication token with `aal2`

### Using Backup Codes

Backup codes work exactly like TOTP codes during verification. Each backup code can only be used once and is permanently invalidated after use.

```bash
POST /api/mfa/verify
Authorization: Bearer <token>
Content-Type: application/json

{
  "challenge_id": "xyz789...",
  "code": "ABCD-1234"  // Backup code instead of TOTP
}
```

## Security Considerations

### Replay Prevention

Each TOTP code can only be used once within its validity window (30 seconds + skew allowance). The system tracks used codes and rejects replay attempts.

### Backup Code Security

- 10 backup codes are generated during enrollment
- Each code can only be used once
- Regenerating codes invalidates ALL previous codes
- Store backup codes securely offline

### Assurance Levels

- **aal1**: Password authentication only
- **aal2**: Password + MFA verification

Use the assurance level endpoint to check the current authentication strength before sensitive operations.

## Sensitive Operations Endpoint

The `/api/mfa/verify-sensitive` endpoint is designed for future use to protect sensitive operations like:

- Account deletion
- Password changes
- API key generation
- Role/permission changes
- Billing operations

Example integration:

```go
// Before performing sensitive operation
verification, err := verifyMFASensitiveOperation(userID, "delete_account", totpCode)
if err != nil {
    return errors.New("MFA verification required")
}

// Perform the sensitive operation within the verification window
if time.Now().Before(verification.ExpiresAt) {
    performSensitiveOperation()
}
```

## Error Codes

| Error | Description |
|-------|-------------|
| `mfa_required` | MFA verification needed to complete authentication |
| `invalid_code` | TOTP code is incorrect or expired |
| `code_already_used` | TOTP code has already been used (replay attack) |
| `factor_not_found` | MFA factor doesn't exist |
| `factor_not_verified` | MFA factor hasn't completed enrollment |
| `no_unused_backup_codes` | All backup codes have been used |
| `verification_expired` | Sensitive operation verification has expired |

## Best Practices

1. **Always save backup codes** when enabling MFA
2. **Use HTTPS** for all MFA-related API calls
3. **Implement rate limiting** on verification endpoints
4. **Log MFA events** for security auditing
5. **Prompt users to regenerate** backup codes if most are used
