# 🔐 Secure Auth — Production-Grade Authentication Microservice in Go

> **Evolved from:** A single-file C project using SHA-256 + `rand()` + flat text files.  
> **Now:** A REST API microservice with Argon2id, JWT sessions, rate limiting, Docker, and CI/CD.

[![CI](https://github.com/sakshamkamra/secure-auth/actions/workflows/ci.yml/badge.svg)](https://github.com/sakshamkamra/secure-auth/actions)
[![Go Version](https://img.shields.io/badge/Go-1.22-blue)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-green)](LICENSE)

---

## 📐 Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        HTTP Client                              │
└───────────────────────┬─────────────────────────────────────────┘
                        │ REST JSON
┌───────────────────────▼─────────────────────────────────────────┐
│              Middleware Stack (chi router)                       │
│  CORS → RequestID → Logging → RateLimit → Auth → RBAC          │
└───────────────────────┬─────────────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────────────┐
│                    API Handlers                                  │
│  /auth/register  /auth/login  /auth/logout  /auth/me  /refresh  │
└───────────────────────┬─────────────────────────────────────────┘
                        │
┌───────────────────────▼─────────────────────────────────────────┐
│                   Auth Service                                   │
│  • Input validation      • Exponential backoff lockout          │
│  • Argon2id hashing      • Audit logging (JSON/slog)            │
└────────────┬──────────────────────────────┬────────────────────┘
             │                              │
┌────────────▼──────────┐     ┌────────────▼──────────────────────┐
│    Session Manager    │     │         Storage Layer              │
│  • JWT access tokens  │     │  interface UserStore { ... }      │
│  • Refresh rotation   │     │  ↳ JSONStore (file, mutex, atomic)│
│  • JTI blacklist      │     │  ↳ swap → SQLite / Postgres       │
└───────────────────────┘     └───────────────────────────────────┘
```

---

## 🚀 Quick Start

### Run locally
```bash
# Clone
git clone https://github.com/sakshamkamra/secure-auth
cd secure-auth

# Run
JWT_SECRET=my-32-byte-super-secret-key-here go run ./cmd/server
```

### Run with Docker
```bash
docker compose up --build
```

Server starts on `http://localhost:8080`.

---

## 📡 API Reference

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/api/v1/auth/register` | None | Register new user |
| `POST` | `/api/v1/auth/login` | None | Login → JWT pair |
| `POST` | `/api/v1/auth/logout` | Bearer | Revoke tokens |
| `POST` | `/api/v1/auth/refresh` | None | Refresh access token |
| `GET`  | `/api/v1/auth/me` | Bearer | Get own profile |
| `GET`  | `/api/v1/admin/users` | Bearer + Admin | List all users |
| `GET`  | `/health` | None | Health check |

### Register
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","email":"alice@example.com","password":"StrongPass@123!"}'
```

### Login
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"alice","password":"StrongPass@123!"}'

# Response:
# {
#   "access_token": "eyJ...",
#   "refresh_token": "abc123...",
#   "expires_at": "2024-01-01T00:15:00Z"
# }
```

### Protected route
```bash
curl http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer eyJ..."
```

---

## 🔒 Security Design

| Feature | Implementation | Why |
|---------|---------------|-----|
| **Password hashing** | Argon2id (64MiB, 3 iter, p=4) | Memory-hard; GPU/ASIC resistant |
| **Salt generation** | `crypto/rand` (CSPRNG) | Unpredictable; defeats rainbow tables |
| **Timing safety** | `crypto/subtle.ConstantTimeCompare` | Prevents timing side-channel |
| **Session tokens** | HS256 JWT + opaque refresh token | Stateless access, revocable refresh |
| **Token revocation** | JTI blacklist + refresh map | Logout works immediately |
| **Brute force** | Exponential backoff (30s→60s→120s…) | Effective against automated attacks |
| **Rate limiting** | Token bucket per IP | API-level DDoS mitigation |
| **Input validation** | Username charset + NIST password policy | Prevents injection, weak passwords |
| **Storage** | RWMutex + atomic write (temp+rename) | Thread-safe, crash-safe |
| **Error messages** | Generic ("invalid credentials") | Prevents username enumeration |
| **Container** | Distroless base image | No shell → minimal attack surface |

---

## 🗂️ Project Structure

```
secure-auth/
├── cmd/server/main.go          # Entry point, graceful shutdown
├── api/handlers.go             # HTTP handlers (REST)
├── internal/
│   ├── auth/
│   │   ├── service.go          # Business logic
│   │   └── auth_test.go        # Unit tests (fake store)
│   ├── crypto/
│   │   ├── crypto.go           # Argon2id, CSPRNG, password policy
│   │   └── crypto_test.go      # Table-driven tests
│   ├── session/
│   │   ├── session.go          # JWT + refresh token manager
│   │   └── session_test.go     # Token lifecycle tests
│   ├── storage/
│   │   ├── storage.go          # UserStore interface + models
│   │   └── json_store.go       # File-backed implementation
│   ├── middleware/middleware.go # RequestID, logging, rate limit, auth, RBAC
│   ├── logger/logger.go        # Structured JSON logger (slog)
│   └── config/config.go        # Env-based configuration
├── Dockerfile                  # 2-stage: builder → distroless
├── docker-compose.yml
├── .github/workflows/ci.yml    # Build, test, lint, vuln scan, Docker
└── go.mod
```

---

## ⚙️ Configuration

All config via environment variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP listen port |
| `JWT_SECRET` | *(change me)* | HMAC signing key — **must be set in production** |
| `ACCESS_TOKEN_EXPIRY` | `15m` | Access token lifetime |
| `REFRESH_TOKEN_EXPIRY` | `168h` | Refresh token lifetime (7 days) |
| `DATA_DIR` | `./data` | Directory for persistent storage |
| `MAX_LOGIN_ATTEMPTS` | `5` | Failures before lockout |
| `LOCKOUT_BASE_DURATION` | `30s` | Base lockout duration (doubles each failure) |
| `RATE_LIMIT_REQUESTS` | `20` | Max requests per window per IP |
| `RATE_LIMIT_WINDOW` | `1m` | Rate limit window |

---

## 🧪 Testing

```bash
# Run all tests
go test ./...

# With race detector (detects concurrency bugs)
go test -race ./...

# With coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## 🔄 Evolution: C → Go

| Concern | Original C | This Go Version |
|---------|-----------|-----------------|
| Language | C (single file) | Go (modular packages) |
| Hashing | SHA-256 (speed hash) | Argon2id (memory-hard) |
| Randomness | `rand()` (predictable) | `crypto/rand` (CSPRNG) |
| Concurrency | `fork()` demo | Goroutines + `sync.RWMutex` |
| Storage | Plain `.txt` | Atomic JSON + mutex locking |
| Sessions | None | JWT access + refresh tokens |
| Brute force | Max 3 attempts | Exponential backoff lockout |
| Rate limiting | None | Per-IP token bucket |
| Logging | Plain text | Structured JSON (SIEM-ready) |
| Transport | CLI | REST HTTP API |
| Build | `gcc main.c` | CMake-equivalent: `go build` |
| Tests | None | Unit tests with fake stores |
| CI/CD | None | GitHub Actions (build+lint+scan+docker) |
| Deployment | Binary | Docker (distroless image) |

---

## 👨‍💻 Author

**Saksham Kamra** — Systems & Backend Engineer  
[GitHub](https://github.com/sakshamkamra33) · [LinkedIn](https://linkedin.com/in/sakshamkamra)
