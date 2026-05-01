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

| Method | Path          | Description          |
| ------ | ------------- | -------------------- |
| POST   | /api/register | Create user → token  |
| POST   | /api/login    | Verify creds → token |
| GET    | /api/me       | Validate JWT         |
| GET    | /api/health   | Health check         |
