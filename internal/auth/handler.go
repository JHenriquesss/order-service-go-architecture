package auth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	apperrors "order-service-go/internal/errors"
)

// Handler exposes the authentication endpoints. It stays thin: decode, call the
// service, map the result. All business rules live in the service.
type Handler struct {
	service *Service
}

// NewHandler builds the auth HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes mounts POST /login and POST /register under the caller's mount point
// (mounted at /api/auth by the router).
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/login", h.Login)
	r.Post("/register", h.Register)
	return r
}

// Login handles POST /api/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := decodeJSON(r, &req); err != nil {
		apperrors.Write(w, err)
		return
	}
	resp, err := h.service.Login(r.Context(), req)
	if err != nil {
		apperrors.Write(w, err)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// Register handles POST /api/auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := decodeJSON(r, &req); err != nil {
		apperrors.Write(w, err)
		return
	}
	resp, err := h.service.Register(r.Context(), req)
	if err != nil {
		apperrors.Write(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, resp)
}

func decodeJSON(r *http.Request, v any) error {
	if err := json.NewDecoder(r.Body).Decode(v); err != nil {
		return apperrors.Validation("invalid request body")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
