# Sprint 1 - Core Product: Authentication

นี่คือโครงสร้างไฟล์และโค้ดทั้งหมดสำหรับ Sprint 1 ครับ

## 📁 Project Structure

```
DevOpsAgents/
.
├── README.md
├── backend
│   ├── auth.db
│   ├── go.mod
│   ├── go.sum
│   ├── internal
│   │   ├── auth
│   │   │   ├── handler.go
│   │   │   ├── handler_test.go
│   │   │   ├── password.go
│   │   │   ├── password_test.go
│   │   │   └── service.go
│   │   ├── middleware
│   │   │   └── cors.go
│   │   └── storage
│   │       └── storage.go
│   └── main.go
├── checkpassword.py
├── frontend
│   ├── __tests__
│   │   └── validation.test.ts
│   ├── app
│   │   ├── dashboard
│   │   │   └── page.tsx
│   │   ├── globals.css
│   │   ├── layout.tsx
│   │   └── page.tsx
│   ├── bun.lock
│   ├── bunfig.toml
│   ├── lib
│   │   ├── api.ts
│   │   └── validation.ts
│   ├── next-env.d.ts
│   ├── next.config.mjs
│   ├── package.json
│   └── tsconfig.json
├── plan
│   └── plan.md
└── test
    └── e2e
        └── auth.e2e.test.ts

14 directories, 28 files
```
---

## 🔧 Backend (Go)

### `backend/go.mod`
```go
module github.com/kong/devopsagents/backend

go 1.22

require (
	github.com/golang-jwt/jwt/v5 v5.2.1
	golang.org/x/crypto v0.24.0
	modernc.org/sqlite v1.30.1
)
```

> ใช้ `modernc.org/sqlite` (pure-Go, ไม่ต้อง CGO) เพื่อให้ build/test ง่ายที่สุด

### `backend/main.go`
```go
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/kong/devopsagents/backend/internal/auth"
	"github.com/kong/devopsagents/backend/internal/middleware"
	"github.com/kong/devopsagents/backend/internal/storage"
)

func main() {
	dbPath := getEnv("DB_PATH", "./auth.db")
	jwtSecret := getEnv("JWT_SECRET", "dev-secret-change-me")
	port := getEnv("PORT", "8080")

	store, err := storage.New(dbPath)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer store.Close()

	svc := auth.NewService(store, jwtSecret)
	h := auth.NewHandler(svc)

	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/register", h.Register)
	mux.HandleFunc("POST /api/login", h.Login)
	mux.HandleFunc("GET /api/me", h.Me)
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})

	handler := middleware.CORS(mux)

	log.Printf("backend listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
```

### `backend/.env.example`
```bash
PORT=8080
DB_PATH=./auth.db
JWT_SECRET=replace-me-with-a-strong-secret
```

---

### `backend/internal/storage/storage.go`
```go
package storage

import (
	"database/sql"
	"errors"

	_ "modernc.org/sqlite"
)

var ErrUserNotFound = errors.New("user not found")
var ErrUserExists = errors.New("user already exists")

type User struct {
	ID           int64
	Username     string
	PasswordHash string
}

type Store struct {
	db *sql.DB
}

func New(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL
	);`
	if _, err := db.Exec(schema); err != nil {
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) CreateUser(username, hash string) (*User, error) {
	res, err := s.db.Exec(
		`INSERT INTO users (username, password_hash) VALUES (?, ?)`,
		username, hash,
	)
	if err != nil {
		// SQLite UNIQUE violation
		return nil, ErrUserExists
	}
	id, _ := res.LastInsertId()
	return &User{ID: id, Username: username, PasswordHash: hash}, nil
}

func (s *Store) GetUserByUsername(username string) (*User, error) {
	row := s.db.QueryRow(
		`SELECT id, username, password_hash FROM users WHERE username = ?`,
		username,
	)
	var u User
	if err := row.Scan(&u.ID, &u.Username, &u.PasswordHash); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}
	return &u, nil
}
```

---

### `backend/internal/auth/password.go`
```go
package auth

import (
	"errors"
	"unicode"
)

var (
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrPasswordNoNumber = errors.New("password must contain at least 1 number")
	ErrUsernameEmpty    = errors.New("username must not be empty")
)

// ValidatePassword enforces the rules:
//   - At least 8 characters
//   - At least 1 number
func ValidatePassword(pw string) error {
	if len(pw) < 8 {
		return ErrPasswordTooShort
	}
	hasNumber := false
	for _, r := range pw {
		if unicode.IsDigit(r) {
			hasNumber = true
			break
		}
	}
	if !hasNumber {
		return ErrPasswordNoNumber
	}
	return nil
}

func ValidateUsername(u string) error {
	if len(u) == 0 {
		return ErrUsernameEmpty
	}
	return nil
}
```

---

### `backend/internal/auth/service.go`
```go
package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/kong/devopsagents/backend/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

var ErrInvalidCredentials = errors.New("invalid username or password")

type Service struct {
	store     *storage.Store
	jwtSecret []byte
}

func NewService(store *storage.Store, secret string) *Service {
	return &Service{store: store, jwtSecret: []byte(secret)}
}

func (s *Service) Register(username, password string) (string, error) {
	if err := ValidateUsername(username); err != nil {
		return "", err
	}
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	u, err := s.store.CreateUser(username, string(hash))
	if err != nil {
		return "", err
	}
	return s.issueToken(u.Username)
}

func (s *Service) Login(username, password string) (string, error) {
	u, err := s.store.GetUserByUsername(username)
	if err != nil {
		return "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}
	return s.issueToken(u.Username)
}

func (s *Service) issueToken(username string) (string, error) {
	claims := jwt.MapClaims{
		"sub": username,
		"exp": time.Now().Add(24 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

func (s *Service) ParseToken(tokenStr string) (string, error) {
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil || !tok.Valid {
		return "", errors.New("invalid token")
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	sub, _ := claims["sub"].(string)
	return sub, nil
}
```

---

### `backend/internal/auth/handler.go`
```go
package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/kong/devopsagents/backend/internal/storage"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type response struct {
	Message string `json:"message,omitempty"`
	Token   string `json:"token,omitempty"`
	User    string `json:"user,omitempty"`
	Error   string `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, body response) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeJSON(w, http.StatusBadRequest, response{Error: "invalid JSON"})
		return
	}
	token, err := h.svc.Register(c.Username, c.Password)
	if err != nil {
		switch {
		case errors.Is(err, ErrPasswordTooShort),
			errors.Is(err, ErrPasswordNoNumber),
			errors.Is(err, ErrUsernameEmpty):
			writeJSON(w, http.StatusBadRequest, response{Error: err.Error()})
		case errors.Is(err, storage.ErrUserExists):
			writeJSON(w, http.StatusConflict, response{Error: "username already taken"})
		default:
			writeJSON(w, http.StatusInternalServerError, response{Error: "internal error"})
		}
		return
	}
	writeJSON(w, http.StatusCreated, response{
		Message: "registration successful",
		Token:   token,
		User:    c.Username,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var c credentials
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeJSON(w, http.StatusBadRequest, response{Error: "invalid JSON"})
		return
	}
	token, err := h.svc.Login(c.Username, c.Password)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, response{Error: "invalid username or password"})
		return
	}
	writeJSON(w, http.StatusOK, response{
		Message: "login successful",
		Token:   token,
		User:    c.Username,
	})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	authz := r.Header.Get("Authorization")
	if !strings.HasPrefix(authz, "Bearer ") {
		writeJSON(w, http.StatusUnauthorized, response{Error: "missing bearer token"})
		return
	}
	tokenStr := strings.TrimPrefix(authz, "Bearer ")
	user, err := h.svc.ParseToken(tokenStr)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, response{Error: "invalid token"})
		return
	}
	writeJSON(w, http.StatusOK, response{User: user})
}
```

---

### `backend/internal/middleware/cors.go`
```go
package middleware

import "net/http"

func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

---

## 🧪 Backend Tests

### `backend/internal/auth/password_test.go`
```go
package auth

import (
	"errors"
	"testing"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{"too short", "ab12", ErrPasswordTooShort},
		{"no number", "abcdefgh", ErrPasswordNoNumber},
		{"valid simple", "abcdefg1", nil},
		{"valid complex", "P@ssw0rd!", nil},
		{"exact 8 chars with digit", "1234abcd", nil},
		{"empty", "", ErrPasswordTooShort},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.input)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("got %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUsername(t *testing.T) {
	if err := ValidateUsername(""); !errors.Is(err, ErrUsernameEmpty) {
		t.Fatalf("expected ErrUsernameEmpty, got %v", err)
	}
	if err := ValidateUsername("kong"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

---

### `backend/internal/auth/handler_test.go`
```go
package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/kong/devopsagents/backend/internal/storage"
)

func newTestHandler(t *testing.T) *Handler {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := storage.New(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return NewHandler(NewService(store, "test-secret"))
}

func doJSON(h http.HandlerFunc, method, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(method, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h(rr, req)
	return rr
}

func TestRegister_Success(t *testing.T) {
	h := newTestHandler(t)
	rr := doJSON(h.Register, http.MethodPost, "/api/register", credentials{
		Username: "kong", Password: "secret123",
	})
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	var resp response
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Token == "" {
		t.Fatal("expected token in response")
	}
}

func TestRegister_InvalidPassword(t *testing.T) {
	h := newTestHandler(t)
	rr := doJSON(h.Register, http.MethodPost, "/api/register", credentials{
		Username: "kong", Password: "short",
	})
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}

func TestRegister_DuplicateUser(t *testing.T) {
	h := newTestHandler(t)
	body := credentials{Username: "dup", Password: "passw0rd"}
	if rr := doJSON(h.Register, http.MethodPost, "/api/register", body); rr.Code != http.StatusCreated {
		t.Fatalf("first register failed: %d", rr.Code)
	}
	rr := doJSON(h.Register, http.MethodPost, "/api/register", body)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d", rr.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	h := newTestHandler(t)
	body := credentials{Username: "kong", Password: "passw0rd"}
	doJSON(h.Register, http.MethodPost, "/api/register", body)

	rr := doJSON(h.Login, http.MethodPost, "/api/login", body)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	h := newTestHandler(t)
	doJSON(h.Register, http.MethodPost, "/api/register", credentials{
		Username: "kong", Password: "passw0rd",
	})
	rr := doJSON(h.Login, http.MethodPost, "/api/login", credentials{
		Username: "kong", Password: "wrong123",
	})
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestMe_WithValidToken(t *testing.T) {
	h := newTestHandler(t)
	rr := doJSON(h.Register, http.MethodPost, "/api/register", credentials{
		Username: "kong", Password: "passw0rd",
	})
	var reg response
	_ = json.Unmarshal(rr.Body.Bytes(), &reg)

	req := httptest.NewRequest(http.MethodGet, "/api/me", nil)
	req.Header.Set("Authorization", "Bearer "+reg.Token)
	w := httptest.NewRecorder()
	h.Me(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
```

---

## 🎨 Frontend (Next.js + Bun + TypeScript)

### `frontend/package.json`
```json
{
  "name": "devopsagents-frontend",
  "version": "0.1.0",
  "private": true,
  "scripts": {
    "dev": "next dev",
    "build": "next build",
    "start": "next start",
    "lint": "next lint",
    "test": "bun test"
  },
  "dependencies": {
    "next": "14.2.5",
    "react": "18.3.1",
    "react-dom": "18.3.1"
  },
  "devDependencies": {
    "@types/bun": "latest",
    "@types/node": "20.14.10",
    "@types/react": "18.3.3",
    "@types/react-dom": "18.3.0",
    "typescript": "5.5.3"
  }
}
```

### `frontend/tsconfig.json`
```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["dom", "dom.iterable", "esnext"],
    "allowJs": true,
    "skipLibCheck": true,
    "strict": true,
    "noEmit": true,
    "esModuleInterop": true,
    "module": "esnext",
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "jsx": "preserve",
    "incremental": true,
    "types": ["bun-types"],
    "plugins": [{ "name": "next" }],
    "paths": { "@/*": ["./*"] }
  },
  "include": ["next-env.d.ts", "**/*.ts", "**/*.tsx", ".next/types/**/*.ts"],
  "exclude": ["node_modules"]
}
```

### `frontend/next.config.ts`
```ts
import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  reactStrictMode: true,
};

export default nextConfig;
```

### `frontend/bunfig.toml`
```toml
[test]
preload = []
```

### `frontend/.env.local.example`
```bash
NEXT_PUBLIC_API_URL=http://localhost:8080
```

---

### `frontend/lib/validation.ts`
```ts
export type ValidationResult = { ok: true } | { ok: false; error: string };

/**
 * Password rule (mirrors backend):
 *  - At least 8 characters
 *  - At least 1 number
 */
export function validatePassword(pw: string): ValidationResult {
  if (pw.length < 8) {
    return { ok: false, error: "Password must be at least 8 characters" };
  }
  if (!/\d/.test(pw)) {
    return { ok: false, error: "Password must contain at least 1 number" };
  }
  return { ok: true };
}

export function validateUsername(u: string): ValidationResult {
  if (u.trim().length === 0) {
    return { ok: false, error: "Username must not be empty" };
  }
  return { ok: true };
}
```

---

### `frontend/lib/api.ts`
```ts
const BASE = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export type AuthResponse = {
  message?: string;
  token?: string;
  user?: string;
  error?: string;
};

async function postAuth(path: string, body: unknown): Promise<AuthResponse> {
  const res = await fetch(`${BASE}${path}`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  return (await res.json()) as AuthResponse;
}

export const api = {
  register: (username: string, password: string) =>
    postAuth("/api/register", { username, password }),
  login: (username: string, password: string) =>
    postAuth("/api/login", { username, password }),
  me: async (token: string): Promise<AuthResponse> => {
    const res = await fetch(`${BASE}/api/me`, {
      headers: { Authorization: `Bearer ${token}` },
    });
    return (await res.json()) as AuthResponse;
  },
};
```

---

### `frontend/app/layout.tsx`
```tsx
import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "DevOpsAgents",
  description: "Sprint 1 - Authentication",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en">
      <body>{children}</body>
    </html>
  );
}
```

### `frontend/app/globals.css`
```css
* { box-sizing: border-box; }
body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  margin: 0;
  background: #0f172a;
  color: #e2e8f0;
}
.container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 1rem;
}
.card {
  background: #1e293b;
  padding: 2rem;
  border-radius: 12px;
  width: 100%;
  max-width: 400px;
  box-shadow: 0 10px 30px rgba(0,0,0,0.3);
}
.card h1 { margin: 0 0 1.5rem; font-size: 1.5rem; }
.field { margin-bottom: 1rem; }
.field label { display: block; margin-bottom: 0.5rem; font-size: 0.9rem; }
.field input {
  width: 100%;
  padding: 0.6rem 0.8rem;
  border-radius: 6px;
  border: 1px solid #334155;
  background: #0f172a;
  color: #e2e8f0;
  font-size: 1rem;
}
.actions { display: flex; gap: 0.5rem; margin-top: 1rem; }
button {
  flex: 1;
  padding: 0.7rem;
  border: none;
  border-radius: 6px;
  font-weight: 600;
  cursor: pointer;
  font-size: 0.95rem;
}
button.primary { background: #3b82f6; color: white; }
button.secondary { background: #475569; color: white; }
button:disabled { opacity: 0.5; cursor: not-allowed; }
.message { margin-top: 1rem; padding: 0.6rem; border-radius: 6px; font-size: 0.9rem; }
.message.success { background: #064e3b; color: #6ee7b7; }
.message.error { background: #7f1d1d; color: #fca5a5; }
```

---

### `frontend/app/page.tsx` (Login Page)
```tsx
"use client";

import { useState, FormEvent } from "react";
import { useRouter } from "next/navigation";
import { api } from "@/lib/api";
import { validatePassword, validateUsername } from "@/lib/validation";

type Status = { kind: "success" | "error"; text: string } | null;

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [status, setStatus] = useState<Status>(null);
  const [loading, setLoading] = useState(false);

  async function handleSubmit(mode: "login" | "register", e: FormEvent) {
    e.preventDefault();
    setStatus(null);

    const u = validateUsername(username);
    if (!u.ok) return setStatus({ kind: "error", text: u.error });

    if (mode === "register") {
      const p = validatePassword(password);
      if (!p.ok) return setStatus({ kind: "error", text: p.error });
    }

    setLoading(true);
    try {
      const res =
        mode === "register"
          ? await api.register(username, password)
          : await api.login(username, password);

      if (res.error || !res.token) {
        setStatus({ kind: "error", text: res.error ?? "Unknown error" });
        return;
      }

      localStorage.setItem("token", res.token);
      localStorage.setItem("user", res.user ?? username);
      setStatus({
        kind: "success",
        text: mode === "register" ? "Registered successfully!" : "Logged in!",
      });
      setTimeout(() => router.push("/dashboard"), 600);
    } catch (err) {
      setStatus({ kind: "error", text: "Network error. Is backend running?" });
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="container">
      <form className="card" onSubmit={(e) => e.preventDefault()}>
        <h1>DevOpsAgents - Login</h1>

        <div className="field">
          <label htmlFor="username">Username</label>
          <input
            id="username"
            type="text"
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoComplete="username"
          />
        </div>

        <div className="field">
          <label htmlFor="password">Password</label>
          <input
            id="password"
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
          />
          <small style={{ color: "#94a3b8" }}>
            Min 8 characters, at least 1 number
          </small>
        </div>

        <div className="actions">
          <button
            className="secondary"
            disabled={loading}
            onClick={(e) => handleSubmit("register", e)}
          >
            Register
          </button>
          <button
            className="primary"
            type="submit"
            disabled={loading}
            onClick={(e) => handleSubmit("login", e)}
          >
            Login
          </button>
        </div>

        {status && (
          <div className={`message ${status.kind}`} role="alert">
            {status.text}
          </div>
        )}
      </form>
    </main>
  );
}
```

---

### `frontend/app/dashboard/page.tsx`
```tsx
"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";

export default function Dashboard() {
  const router = useRouter();
  const [user, setUser] = useState<string | null>(null);

  useEffect(() => {
    const token = localStorage.getItem("token");
    const u = localStorage.getItem("user");
    if (!token) {
      router.replace("/");
      return;
    }
    setUser(u);
  }, [router]);

  function logout() {
    localStorage.removeItem("token");
    localStorage.removeItem("user");
    router.replace("/");
  }

  return (
    <main className="container">
      <div className="card">
        <h1>Welcome, {user ?? "..."} 👋</h1>
        <p>You are now authenticated.</p>
        <button className="primary" onClick={logout}>
          Logout
        </button>
      </div>
    </main>
  );
}
```

---

## 🧪 Frontend Tests

### `frontend/__tests__/validation.test.ts`
```ts
import { describe, expect, test } from "bun:test";
import { validatePassword, validateUsername } from "../lib/validation";

describe("validatePassword", () => {
  test("rejects passwords shorter than 8 chars", () => {
    const r = validatePassword("ab12");
    expect(r.ok).toBe(false);
    if (!r.ok) expect(r.error).toMatch(/8 characters/);
  });

  test("rejects passwords without a number", () => {
    const r = validatePassword("abcdefgh");
    expect(r.ok).toBe(false);
    if (!r.ok) expect(r.error).toMatch(/1 number/);
  });

  test("accepts valid password", () => {
    expect(validatePassword("abcdefg1").ok).toBe(true);
    expect(validatePassword("P@ssw0rd!").ok).toBe(true);
  });

  test("rejects empty password", () => {
    expect(validatePassword("").ok).toBe(false);
  });
});

describe("validateUsername", () => {
  test("rejects empty / whitespace", () => {
    expect(validateUsername("").ok).toBe(false);
    expect(validateUsername("   ").ok).toBe(false);
  });
  test("accepts non-empty", () => {
    expect(validateUsername("kong").ok).toBe(true);
  });
});
```

---

## 🌐 E2E Test

### `test/e2e/auth.e2e.test.ts`
```ts
/**
 * E2E test against a running backend at http://localhost:8080
 * Run:
 *   1) cd backend && go run .
 *   2) cd test && bun test
 */
import { describe, expect, test } from "bun:test";

const BASE = process.env.API_URL ?? "http://localhost:8080";

function uniqueUser() {
  return `user_${Date.now()}_${Math.floor(Math.random() * 1000)}`;
}

describe("auth e2e", () => {
  test("health endpoint", async () => {
    const res = await fetch(`${BASE}/api/health`);
    expect(res.status).toBe(200);
  });

  test("register -> login -> me", async () => {
    const username = uniqueUser();
    const password = "passw0rd";

    const reg = await fetch(`${BASE}/api/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });
    expect(reg.status).toBe(201);
    const regBody = await reg.json();
    expect(regBody.token).toBeTruthy();

    const login = await fetch(`${BASE}/api/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password }),
    });
    expect(login.status).toBe(200);
    const loginBody = await login.json();
    expect(loginBody.token).toBeTruthy();

    const me = await fetch(`${BASE}/api/me`, {
      headers: { Authorization: `Bearer ${loginBody.token}` },
    });
    expect(me.status).toBe(200);
    const meBody = await me.json();
    expect(meBody.user).toBe(username);
  });

  test("rejects weak password", async () => {
    const res = await fetch(`${BASE}/api/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username: uniqueUser(), password: "short" }),
    });
    expect(res.status).toBe(400);
  });

  test("rejects wrong password", async () => {
    const username = uniqueUser();
    await fetch(`${BASE}/api/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password: "passw0rd" }),
    });
    const res = await fetch(`${BASE}/api/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username, password: "wrongpw1" }),
    });
    expect(res.status).toBe(401);
  });
});
```

---

## 📖 README.md (อัปเดต)

```markdown
# DevOpsAgents

## Sprint 1 — Authentication (Core Product)

### Stack
- **Backend:** Go 1.22 + SQLite (modernc.org/sqlite, pure-Go)
- **Frontend:** Next.js 14 (App Router) + TypeScript + Bun
- **Auth:** bcrypt + JWT (HS256)

### Password Policy
- ≥ 8 characters
- ≥ 1 number

---

## Run Backend
```bash
cd backend
cp .env.example .env
go mod tidy
go run .
# listening on :8080
```

## Run Frontend
```bash
cd frontend
cp .env.local.example .env.local
bun install
bun run dev
# http://localhost:3000
```

## Tests
```bash
# Backend unit tests
cd backend && go test ./... -v

# Frontend unit tests (Bun's built-in test runner)
cd frontend && bun test

# E2E (requires backend running)
cd test && bun test
```

### API
| Method | Path           | Description           |
|--------|----------------|-----------------------|
| POST   | /api/register  | Create user → token   |
| POST   | /api/login     | Verify creds → token  |
| GET    | /api/me        | Validate JWT          |
| GET    | /api/health    | Health check          |
```

---

## ✅ Sprint 1 Checklist

| Requirement | Implementation |
|---|---|
| Login Page (username + password) | `frontend/app/page.tsx` |
| Register / Login button | Two buttons in same form |
| Success / error message | `<div className="message">` ด้วย `Status` state |
| Password ≥ 8 chars | `ValidatePassword` (Go) + `validatePassword` (TS) |
| Password ≥ 1 number | เช็คทั้งฝั่ง backend และ frontend |
| Minimal database | SQLite (single file, ตารางเดียว `users`) — pure-Go ไม่ต้อง CGO |
| Authentication Logic | bcrypt hashing + JWT (24h) |
| Testing | Go unit tests + Bun unit tests + E2E |

### ขั้นตอนถัดไปที่แนะนำสำหรับ Sprint 2
1. Refresh tokens / session expiration UX
2. Rate-limiting middleware (anti brute-force)
3. CI workflow (GitHub Actions): `go test` + `bun test`
4. Containerize ด้วย Docker Compose

ลอง run แล้วบอกได้ครับว่าเจอ error ตรงไหน — เดี๋ยวช่วย debug ต่อให้ 🚀