# Integration Request Packet: Centralized SSO Authentication

## 1. Meta Information
* **VAJRA Capability:** User Wallet Authentication & Canonical Identity Assertion.
* **Adjacent Participant:** Central TANTRA Auth Service (`bhiv-auth`).
* **Purpose of Connection:** To securely verify user identity across the TANTRA ecosystem, map SSO profiles to VAJRA wallets, and enforce global access control.
* **VAJRA-side Owner:** Prakash (Trust Layer Architect)
* **Adjacent-side Owner:** Identity / Auth Service Owner
* **Status:** 🔴 **BLOCKED**

## 2. API / Event / SDK Required
VAJRA Wallet expects the following endpoints to be live and resolvable at `https://bhiv-auth.onrender.com`:
1. `GET /api/health` -> Returns `200 OK` (Heartbeat).
2. `POST /api/auth/login` -> Returns a signed JWT in a `Set-Cookie` header (`bhiv_token`) upon successful login.
3. `POST /api/auth/register` -> Accepts JSON user payload and creates canonical identity.

## 3. Schema Required
### Login Payload (Expected Input)
```json
{
  "username": "user123",
  "password": "securepassword"
}
```

### JWT Claims (Expected Output)
```json
{
  "user_id": "mongo_object_id",
  "email": "user@example.com",
  "tenant_id": "bhiv-global",
  "roles": ["user"],
  "permissions": [],
  "allowedApps": ["vajra"]
}
```

## 4. Current Blocker & Acceptance Test
### Current Blocker
The service at `https://bhiv-auth.onrender.com` is actively returning an HTTP `503 Service Unavailable`. An investigation into the `blackhole_auth` repository shows that it contains client middleware, but lacks the core server, database connections, and login routes. The application is crashing on startup. 

Because VAJRA enforces strict constitutional boundaries, we cannot rebuild the auth service locally. We rely on the auth owner to restore this service.

### Acceptance Test
1. Send `GET https://bhiv-auth.onrender.com/api/health` -> Receives `200 OK`.
2. Authenticate a test user -> Receives valid JWT cookie containing `"allowedApps": ["vajra"]`.

## 5. Action Required
The Auth Owner must restore the MongoDB-backed Node.js express server logic, ensure `JWT_SECRET` is defined in the Render environment, and deploy the working build to the production endpoint. Once restored, notify the VAJRA team for bilateral testing.
