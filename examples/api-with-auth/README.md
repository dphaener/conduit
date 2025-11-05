# API with Authentication Example

**Difficulty:** Advanced
**Time:** 20 minutes

A complete API authentication system demonstrating account management, API keys, and audit logging. This example shows patterns for building secure, production-ready authentication.

## What You'll Learn

- Account management with email verification
- API key generation and management
- Audit logging for security events
- Optional relationships (nullable foreign keys)
- Different `on_delete` behaviors for security
- Status tracking patterns
- Token-based authentication patterns

## Quick Start

```bash
# Navigate to this directory
cd examples/api-with-auth

# Build the application
conduit build

# The build creates a complete auth API with:
# - Account registration and verification
# - API key management
# - Security audit logging
# - Token-based authentication
```

## What's Inside

This example contains three security-focused resources:

### 1. Account Resource (`app/resources/account.cdt`)

User accounts with email verification:

```conduit
resource Account {
  id: uuid! @primary @auto
  email: string! @unique @min(5) @max(255)
  password_hash: string! @min(60) @max(255)
  status: string! @default("pending")
  email_verified: bool! @default(false)
  verification_token: string?
  api_token: string?
  last_login_at: timestamp?
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  @before create {
    self.email = String.downcase(self.email)
  }
}
```

**Key features:**
- Email verification workflow with `verification_token`
- Account status tracking (pending/active/suspended/deleted)
- API token for session management
- Last login tracking
- Email normalization for consistent lookup

### 2. API Key Resource (`app/resources/api_key.cdt`)

Programmatic API access tokens:

```conduit
resource ApiKey {
  id: uuid! @primary @auto
  account_id: uuid!
  name: string! @min(3) @max(100)
  key_hash: string! @unique @min(64) @max(255)
  description: text?
  scopes: string! @default("read")
  status: string! @default("active")
  expires_at: timestamp?
  last_used_at: timestamp?
  revoked_at: timestamp?
  created_at: timestamp! @auto
  updated_at: timestamp! @auto_update

  account: Account! {
    foreign_key: "account_id"
    on_delete: cascade
  }
}
```

**Key features:**
- Multiple API keys per account
- Named keys for easy identification
- Scope-based permissions
- Optional expiration dates
- Revocation tracking
- Last used tracking for security monitoring
- `on_delete: cascade` - deleting account deletes all API keys

### 3. Audit Log Resource (`app/resources/audit_log.cdt`)

Security event logging:

```conduit
resource AuditLog {
  id: uuid! @primary @auto
  account_id: uuid?
  event_type: string! @min(3) @max(50)
  description: string! @min(1) @max(500)
  ip_address: string?
  user_agent: text?
  success: bool! @default(true)
  error_message: text?
  created_at: timestamp! @auto

  account: Account? {
    foreign_key: "account_id"
    on_delete: set_null
  }
}
```

**Key features:**
- **Immutable logs** - no `updated_at` field
- **Optional account** - can log events before authentication
- Event categorization with `event_type`
- Success/failure tracking
- IP and user agent tracking
- `on_delete: set_null` - preserve logs even if account deleted

## Generated API Endpoints

### Accounts
- `POST /accounts` - Register new account
- `GET /accounts/:id` - Get account details
- `PUT /accounts/:id` - Update account (verify email, change password)
- `DELETE /accounts/:id` - Delete account (cascades to API keys)

### API Keys
- `POST /api_keys` - Create new API key
- `GET /api_keys` - List all API keys
- `GET /api_keys/:id` - Get specific API key
- `PUT /api_keys/:id` - Update API key (revoke, extend expiration)
- `DELETE /api_keys/:id` - Delete API key

### Audit Logs
- `GET /audit_logs` - List audit events
- `GET /audit_logs/:id` - Get specific audit event
- `POST /audit_logs` - Create audit log entry

## Authentication Patterns

### 1. Account Registration Flow

```bash
# Step 1: Create account
curl -X POST http://localhost:3000/accounts \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password_hash": "$2a$10$...",
    "verification_token": "abc123..."
  }'

# Step 2: Verify email (update account)
curl -X PUT http://localhost:3000/accounts/:id \
  -H "Content-Type: application/json" \
  -d '{
    "email_verified": true,
    "verification_token": null,
    "status": "active"
  }'

# Step 3: Log security event
curl -X POST http://localhost:3000/audit_logs \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "...",
    "event_type": "email_verified",
    "description": "Email verified successfully",
    "ip_address": "192.168.1.1",
    "success": true
  }'
```

### 2. API Key Management

```bash
# Create API key
curl -X POST http://localhost:3000/api_keys \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "...",
    "name": "Production API Key",
    "key_hash": "...",  # Secure random hash
    "scopes": "read,write",
    "expires_at": "2026-12-31T23:59:59Z"
  }'

# Revoke API key
curl -X PUT http://localhost:3000/api_keys/:id \
  -H "Content-Type: application/json" \
  -d '{
    "status": "revoked",
    "revoked_at": "2025-11-04T12:00:00Z"
  }'
```

### 3. Audit Logging

```bash
# Log failed login attempt
curl -X POST http://localhost:3000/audit_logs \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": "...",
    "event_type": "login_failed",
    "description": "Invalid password",
    "ip_address": "203.0.113.42",
    "user_agent": "Mozilla/5.0...",
    "success": false,
    "error_message": "Invalid credentials"
  }'

# Log anonymous event (no account)
curl -X POST http://localhost:3000/audit_logs \
  -H "Content-Type: application/json" \
  -d '{
    "account_id": null,
    "event_type": "registration_attempt",
    "description": "New account registration",
    "ip_address": "198.51.100.1",
    "success": true
  }'
```

## Key Concepts Demonstrated

### 1. Optional Relationships

Unlike the blog example where relationships are always required, audit logs demonstrate optional relationships:

```conduit
// Optional foreign key
account_id: uuid?

// Optional relationship
account: Account? {
  foreign_key: "account_id"
  on_delete: set_null
}
```

This allows logging events even when no account exists (anonymous users, failed login attempts).

### 2. on_delete Behaviors

Three different deletion strategies:

**cascade** - Delete dependent records:
```conduit
// Delete account -> delete all API keys
account: Account! {
  foreign_key: "account_id"
  on_delete: cascade
}
```

**restrict** - Prevent deletion if dependencies exist:
```conduit
// Cannot delete account with audit logs (from blog example)
account: Account! {
  foreign_key: "account_id"
  on_delete: restrict
}
```

**set_null** - Preserve records but remove reference:
```conduit
// Delete account -> keep audit logs, set account_id to null
account: Account? {
  foreign_key: "account_id"
  on_delete: set_null
}
```

### 3. Immutable Resources

Audit logs demonstrate immutability:

```conduit
resource AuditLog {
  // ...fields...
  created_at: timestamp! @auto
  // Note: No updated_at field
  // Audit logs should never be modified
}
```

This is a common pattern for compliance and security.

### 4. Status Tracking

Multiple resources use status fields:

```conduit
// Account status
status: string! @default("pending")

// API key status
status: string! @default("active")
```

In production, consider using enums for type safety (when implemented).

### 5. Token Management

Multiple token patterns demonstrated:

```conduit
// Short-lived verification token
verification_token: string?

// Session token
api_token: string?

// Long-lived API key
key_hash: string! @unique
```

### 6. Security Metadata

Tracking security-relevant information:

```conduit
// Who, what, when, where
account_id: uuid?
event_type: string!
created_at: timestamp!
ip_address: string?
user_agent: text?
```

## Database Schema

```sql
-- Accounts table
CREATE TABLE accounts (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email VARCHAR(255) NOT NULL UNIQUE,
  password_hash VARCHAR(255) NOT NULL,
  status VARCHAR(20) NOT NULL DEFAULT 'pending',
  email_verified BOOLEAN NOT NULL DEFAULT false,
  verification_token VARCHAR(255),
  api_token VARCHAR(255),
  last_login_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- API keys table with foreign key
CREATE TABLE api_keys (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  account_id UUID NOT NULL,
  name VARCHAR(100) NOT NULL,
  key_hash VARCHAR(255) NOT NULL UNIQUE,
  description TEXT,
  scopes VARCHAR(255) NOT NULL DEFAULT 'read',
  status VARCHAR(20) NOT NULL DEFAULT 'active',
  expires_at TIMESTAMP,
  last_used_at TIMESTAMP,
  revoked_at TIMESTAMP,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
  FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE CASCADE
);

-- Audit logs table with optional foreign key
CREATE TABLE audit_logs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  account_id UUID,  -- Nullable
  event_type VARCHAR(50) NOT NULL,
  description VARCHAR(500) NOT NULL,
  ip_address VARCHAR(45),
  user_agent TEXT,
  success BOOLEAN NOT NULL DEFAULT true,
  error_message TEXT,
  created_at TIMESTAMP NOT NULL DEFAULT NOW(),
  FOREIGN KEY (account_id) REFERENCES accounts(id) ON DELETE SET NULL
);

-- Example indices for production use (not auto-generated by Conduit)
-- You can add these manually to your database for better query performance:
CREATE INDEX idx_audit_logs_account_id ON audit_logs(account_id);
CREATE INDEX idx_audit_logs_event_type ON audit_logs(event_type);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
CREATE INDEX idx_api_keys_account_id ON api_keys(account_id);
CREATE INDEX idx_api_keys_status ON api_keys(status);
```

## Production Considerations

This example demonstrates patterns, but production apps need:

### Password Hashing
```go
// Use bcrypt in production
import "golang.org/x/crypto/bcrypt"

hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
```

### Token Generation
```go
// Use secure random tokens
import "crypto/rand"
import "encoding/hex"

token := make([]byte, 32)
rand.Read(token)
tokenStr := hex.EncodeToString(token)
```

### Rate Limiting
```go
// Add rate limiting for auth endpoints
// Prevent brute force attacks
```

### Email Sending
```go
// Integrate email service for verification
// Send verification_token via email
```

### Session Management
```go
// Use Redis for session storage
// Set expiration on api_token
```

## Common Patterns

### Email Normalization
```conduit
@before create {
  self.email = String.downcase(self.email)
}
```

Ensures case-insensitive email matching.

### Scope Normalization
```conduit
@before create {
  self.scopes = String.downcase(self.scopes)
}
```

Normalizes permission scopes for consistency.

### Event Type Standardization
```conduit
@before create {
  self.event_type = String.downcase(self.event_type)
}
```

Ensures consistent event categorization.

## Security Best Practices

1. **Never store plain text passwords** - Use bcrypt or similar
2. **Generate secure tokens** - Use crypto/rand, not math/rand
3. **Log all security events** - Authentication, authorization, changes
4. **Use HTTPS in production** - Protect tokens in transit
5. **Implement rate limiting** - Prevent brute force attacks
6. **Expire tokens** - Use `expires_at` for API keys
7. **Revoke compromised tokens** - Use `status` field
8. **Track last used** - Detect unused or suspicious keys

## Features Used (Conservative)

This example **only** uses working features:
- ✅ `belongs_to` relationships (inline metadata)
- ✅ Optional fields with `?`
- ✅ String functions: `downcase()`
- ✅ Field constraints: `@unique`, `@min`, `@max`, `@default`
- ✅ Lifecycle hooks: `@before create/update`
- ✅ Different `on_delete` behaviors

**Not used** (not yet implemented):
- ❌ Enums for status fields (use strings)
- ❌ Query methods like `Account.find_by(email: "...")`
- ❌ `@computed` fields
- ❌ Password hashing in hooks (do in application code)
- ❌ Token generation in hooks (do in application code)

See ROADMAP.md for implementation status.

## Next Steps

1. **Implement authentication middleware** - Validate API tokens in Go
2. **Add password reset flow** - Use verification tokens
3. **Add 2FA support** - Store 2FA secrets in Account
4. **Add role-based access** - Create Role and Permission resources
5. **Add session management** - Create Session resource

## Learn More

- **GETTING-STARTED.md** - Full tutorial
- **LANGUAGE-SPEC.md** - Complete language reference
- **ROADMAP.md** - Implementation status
- **examples/blog/** - Learn about relationships
- **examples/minimal/** - Start with basics
