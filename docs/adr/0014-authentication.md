---
status: accepted
date: 2026-11-13
decision-makers: "@amanuelmr"
---

# ADR-0014: Authentication with server-side sessions, Argon2id, opaque API tokens, OIDC and passkeys

## Context and Problem Statement

Server mode must authenticate users of the web UI and programmatic API clients
(FR-USER-01…07, NFR-SEC-01). Personal mode must stay frictionless but safe. What
authentication mechanisms do we use?

## Decision Drivers

* OWASP guidance and ASVS 5.0 Level 2.
* Immediate revocation (logout, password change, admin action).
* Simplicity for self-hosters; optional single sign-on for teams.

## Considered Options

* **Server-side sessions** (opaque cookie) for browsers, plus **opaque hashed API
  tokens** for programs
* Stateless JWTs for browsers and APIs
* Delegating all authentication to an external identity provider

## Decision Outcome

Chosen option: **server-side sessions plus opaque API tokens**, with optional OIDC
and passkeys.

| Mechanism | Design |
|---|---|
| Browser sessions | `alexedwards/scs` v2 with a database-backed store (SQLite and PostgreSQL). Cookie `__Host-wf_session`: `Secure`, `HttpOnly`, `SameSite=Lax`, `Path=/`. Session ID rotated on login and privilege change. Idle timeout 7 days; absolute lifetime 30 days (configurable). |
| CSRF | `SameSite` cookies plus Go's `http.CrossOriginProtection` (Fetch-metadata `Sec-Fetch-Site` checks, standard library since Go 1.25), plus an `Origin` check on the WebSocket upgrade. |
| Passwords | Argon2id (`golang.org/x/crypto/argon2`), defaults m = 46 MiB, t = 1, p = 1 (one of OWASP's equivalent minimum configurations). Parameters are stored with the hash, so they can be upgraded transparently on login. Minimum length 12, maximum 128; checked against a breached-password list where available offline. Login rate limiting and lockout with backoff. |
| API tokens | Format `wf_<id>_<secret>`. Only `SHA-256(secret)` is stored. Scoped (`read`, `write`, `alerts`, `admin`), with expiry and revocation. Sent as `Authorization: Bearer`. |
| OIDC (M3) | `coreos/go-oidc` v3 with `x/oauth2`, Authorization Code flow with PKCE. Group claim to role mapping is configurable. |
| Passkeys / 2FA (M3) | `go-webauthn/webauthn` (pre-1.0; pinned) for passkeys; TOTP (RFC 6238) as a second factor, with recovery codes. |
| Personal mode | No accounts. The server **refuses** to bind a non-loopback address unless auth is enabled. The desktop shell adds a per-launch random token (ADR-0009). |
| Authorisation | Roles `admin` and `member`, plus resource ownership checks in the service layer (never only in handlers). Deny by default. |

### Consequences

* Good, because sessions can be revoked instantly, and there is no JWT signing-key
  management or `alg` pitfalls.
* Good, because only token hashes are stored, so a database leak does not reveal
  usable tokens.
* Bad, because there is a session lookup per request. Mitigated by caching and
  indexing; it is negligible at our scale.

### Confirmation

* Security test suite: session fixation, CSRF, IDOR (one user accessing another's
  resources), token scope enforcement, and the loopback-binding guard.
* An ASVS L2 checklist is tracked in [security](../architecture/11-security.md).

## Pros and Cons of the Options

### JWTs for browser sessions

* Good, because validation is stateless.
* Bad, because they are hard to revoke, there are key-management and
  algorithm-confusion risks, and OWASP advises care when using them as sessions.

### External identity provider only

* Good, because authentication is fully delegated.
* Bad, because it forces every self-hoster to run an identity provider. It stays
  optional (OIDC).

## More Information

* OWASP Session Management Cheat Sheet: <https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html>
* OWASP Password Storage Cheat Sheet: <https://cheatsheetseries.owasp.org/cheatsheets/Password_Storage_Cheat_Sheet.html>
* OWASP ASVS: <https://owasp.org/www-project-application-security-verification-standard/>
