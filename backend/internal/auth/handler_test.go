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
