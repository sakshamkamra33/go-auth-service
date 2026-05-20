# 🔐 Go Auth Service

A production-ready, full-stack authentication service built with **Go** (backend) and **React + TypeScript** (frontend). Designed to demonstrate enterprise-grade security patterns including Argon2id password hashing, JWT rotation, RBAC, brute-force protection, audit logging, and real email delivery via SMTP.

[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?style=flat&logo=go)](https://go.dev)
[![React](https://img.shields.io/badge/React-19-61DAFB?style=flat&logo=react)](https://react.dev)
[![TypeScript](https://img.shields.io/badge/TypeScript-5-3178C6?style=flat&logo=typescript)](https://www.typescriptlang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## ✨ Features

### 🔐 Security
- **Argon2id password hashing** — industry-standard, memory-hard algorithm
- **JWT access + refresh token rotation** — short-lived access tokens, long-lived refresh tokens with silent rotation
- **Anti-brute-force protection** — exponential backoff lockout after configurable failed attempts
- **API Rate Limiting** — per-IP request throttling to prevent DDoS/spam
- **Single-use token invalidation** — email verification and password reset tokens are immediately deleted after use

### 👥 User Management
- **Role-Based Access Control (RBAC)** — `user` and `admin` roles enforced at the middleware level
- **Email verification flow** — secure, single-use, 24-hour expiry tokens
- **Password recovery flow** — time-expiring (15-min) reset tokens sent via email
- **Audit logging** — every critical security event (login, logout, failures, resets) persisted to `data/audit.jsonl`

### 💻 Full-Stack SPA
- **Protected routing** — React Router guards redirect unauthenticated users
- **Silent token refresh** — the API client automatically intercepts 401s and retries with a new access token
- **Dashboard** — displays user profile, role, and email verification status
- **Admin Panel** — lists all users, allows deletion of non-admin accounts
- **Forgot/Reset Password** pages with clean, animated UI
- **Email Verification** page (auto-verifies from URL query token)

---

## 🏗️ Tech Stack

| Layer | Technology |
|---|---|
| Backend Language | Go 1.23+ |
| HTTP Router | `go-chi/chi` v5 |
| Password Hashing | Argon2id (`golang.org/x/crypto`) |
| Auth Tokens | JWT (`golang-jwt/jwt` v5) |
| Database | SQLite (`modernc.org/sqlite`) |
| Email | SMTP (built-in `net/smtp`, dev fallback to console) |
| Frontend | React 19 + TypeScript + Vite |
| Animations | Framer Motion |
| Icons | Lucide React |
| Routing | React Router DOM v7 |
| Containerization | Docker (distroless image) |

---

## 🚀 Getting Started

### Prerequisites
- Go 1.23+
- Node.js 20+

### 1. Clone the repo
```bash
git clone https://github.com/sakshamkamra33/go-auth-service.git
cd go-auth-service
```

### 2. Start the Backend
```powershell
# Windows PowerShell
$env:JWT_SECRET="your-secret-min-32-chars-here!!"; go run ./cmd/server
```
```bash
# Linux / macOS
JWT_SECRET="your-secret-min-32-chars-here!!" go run ./cmd/server
```
The server starts on **http://localhost:8080**.

### 3. Start the Frontend
```bash
cd frontend
npm install
npm run dev
```
The app opens on **http://localhost:5173**.

---

## 📧 Email Configuration

By default, the server prints emails to the terminal (dev mode). To send real emails, set these environment variables when starting the backend:

| Variable | Description | Example |
|---|---|---|
| `SMTP_HOST` | SMTP server hostname | `smtp.gmail.com` |
| `SMTP_PORT` | SMTP port (default: 587) | `587` |
| `SMTP_FROM` | Sender email address | `noreply@yourapp.com` |
| `SMTP_PASS` | SMTP password / app password | `your-app-password` |

> **Tip for testing:** Use [Ethereal Email](https://ethereal.email) for a free, zero-setup fake SMTP inbox.

---

## 🐳 Docker

```bash
# Build and run with Docker Compose
make docker-up

# Stop
make docker-down
```

---

## 📁 Project Structure

```
go-auth-service/
├── cmd/server/         # Application entrypoint (main.go)
├── api/                # HTTP handlers
├── internal/
│   ├── auth/           # Business logic (register, login, tokens)
│   ├── config/         # Environment variable configuration
│   ├── crypto/         # Argon2id hashing, token generation
│   ├── email/          # SMTP mailer with dev fallback
│   ├── logger/         # Structured logging (slog)
│   ├── session/        # JWT issuing, validation, revocation
│   └── storage/        # SQLite user store + audit log
├── frontend/           # React + TypeScript SPA
│   └── src/
│       ├── api/        # Centralized API client with auto token refresh
│       ├── context/    # AuthContext (global auth state)
│       ├── components/ # ProtectedRoute
│       └── pages/      # AuthPage, Dashboard, AdminPanel, etc.
└── data/               # Runtime data (SQLite DB + audit log) — gitignored
```

---

## 🔑 API Endpoints

| Method | Endpoint | Auth | Description |
|---|---|---|---|
| `POST` | `/api/v1/auth/register` | Public | Create a new account |
| `POST` | `/api/v1/auth/login` | Public | Authenticate and get tokens |
| `POST` | `/api/v1/auth/refresh` | Public | Exchange refresh token for new pair |
| `POST` | `/api/v1/auth/logout` | Bearer | Revoke tokens |
| `GET` | `/api/v1/auth/me` | Bearer | Get current user profile |
| `POST` | `/api/v1/auth/verify-email` | Public | Verify email with token |
| `POST` | `/api/v1/auth/forgot-password` | Public | Request password reset email |
| `POST` | `/api/v1/auth/reset-password` | Public | Set new password with token |
| `GET` | `/api/v1/admin/users` | Admin | List all users |
| `DELETE` | `/api/v1/admin/users/:id` | Admin | Delete a user |
| `GET` | `/api/v1/admin/audit-logs` | Admin | View audit log |
| `GET` | `/health` | Public | Health check |

---

## 📜 License

MIT © [Saksham Kamra](https://github.com/sakshamkamra33)
