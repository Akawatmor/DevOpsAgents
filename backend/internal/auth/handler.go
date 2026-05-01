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
