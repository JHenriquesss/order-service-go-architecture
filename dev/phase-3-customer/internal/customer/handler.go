package customer

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	apperrors "order-service-customer/internal/errors"
)

// Handler exposes the five customer endpoints (architecture §11). It stays thin:
// decode/parse the request shape, call the service, map the result. All business
// rules live in the service.
type Handler struct {
	service *Service
}

// NewHandler builds the customer HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register mounts the customer routes (architecture §11) onto mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/customers", h.List)
	mux.HandleFunc("POST /api/customers", h.Create)
	mux.HandleFunc("GET /api/customers/{id}", h.Get)
	mux.HandleFunc("PUT /api/customers/{id}", h.Update)
	mux.HandleFunc("PATCH /api/customers/{id}/deactivate", h.Deactivate)
}

// Create handles POST /api/customers.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateCustomerInput
	if err := decodeJSON(r, &input); err != nil {
		apperrors.Write(w, err)
		return
	}
	out, err := h.service.Create(r.Context(), input)
	if err != nil {
		apperrors.Write(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// Update handles PUT /api/customers/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		apperrors.Write(w, err)
		return
	}
	var input UpdateCustomerInput
	if err := decodeJSON(r, &input); err != nil {
		apperrors.Write(w, err)
		return
	}
	out, err := h.service.Update(r.Context(), id, input)
	if err != nil {
		apperrors.Write(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Deactivate handles PATCH /api/customers/{id}/deactivate.
func (h *Handler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		apperrors.Write(w, err)
		return
	}
	if err := h.service.Deactivate(r.Context(), id); err != nil {
		apperrors.Write(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Get handles GET /api/customers/{id}.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		apperrors.Write(w, err)
		return
	}
	out, err := h.service.FindByID(r.Context(), id)
	if err != nil {
		apperrors.Write(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// List handles GET /api/customers with filtering and pagination (architecture §13).
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	filter, err := parseFilter(r)
	if err != nil {
		apperrors.Write(w, err)
		return
	}
	page, err := h.service.List(r.Context(), filter)
	if err != nil {
		apperrors.Write(w, err)
		return
	}
	writeJSON(w, http.StatusOK, page)
}

func parseFilter(r *http.Request) (CustomerFilter, error) {
	q := r.URL.Query()
	filter := CustomerFilter{
		Name:     q.Get("name"),
		Document: q.Get("document"),
	}
	if raw := q.Get("active"); raw != "" {
		active, err := strconv.ParseBool(raw)
		if err != nil {
			return CustomerFilter{}, apperrors.Validation("active must be a boolean")
		}
		filter.Active = &active
	}
	page, err := parseIntParam(q.Get("page"))
	if err != nil {
		return CustomerFilter{}, apperrors.Validation("page must be an integer")
	}
	filter.Page = page
	pageSize, err := parseIntParam(q.Get("page_size"))
	if err != nil {
		return CustomerFilter{}, apperrors.Validation("page_size must be an integer")
	}
	filter.PageSize = pageSize
	return filter, nil
}

// parseIntParam parses an optional integer query param. Empty returns 0, which
// the service interprets as "use the default".
func parseIntParam(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return uuid.Nil, apperrors.Validation("invalid customer id")
	}
	return id, nil
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
