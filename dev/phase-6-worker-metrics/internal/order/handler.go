package order

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"

	"order-service-worker/internal/auth"

	apperrors "order-service-worker/internal/errors"
)

// Handler exposes the order endpoints (architecture §11). It stays thin: decode,
// call service, map result. Business rules live in the service.
type Handler struct {
	service *Service
}

// NewHandler builds the order HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register mounts the order routes onto mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/orders", h.Create)
	mux.HandleFunc("GET /api/orders", h.List)
	mux.HandleFunc("GET /api/orders/{id}", h.Get)
	mux.HandleFunc("PATCH /api/orders/{id}/cancel", h.Cancel)
	mux.HandleFunc("PATCH /api/orders/{id}/ship", h.Ship)
}

// Create handles POST /api/orders.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFromContext(r.Context())
	if !ok {
		apperrors.Write(w, apperrors.Unauthorized("Authentication required"))
		return
	}
	var input CreateOrderInput
	if err := decodeJSON(r, &input); err != nil {
		apperrors.Write(w, err)
		return
	}
	out, err := h.service.Create(r.Context(), identity.UserID, input)
	if err != nil {
		apperrors.Write(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// Get handles GET /api/orders/{id}.
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

// List handles GET /api/orders with filtering and pagination.
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

// Cancel handles PATCH /api/orders/{id}/cancel.
func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		apperrors.Write(w, err)
		return
	}
	out, err := h.service.Cancel(r.Context(), id)
	if err != nil {
		apperrors.Write(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Ship handles PATCH /api/orders/{id}/ship.
func (h *Handler) Ship(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		apperrors.Write(w, err)
		return
	}
	out, err := h.service.Ship(r.Context(), id)
	if err != nil {
		apperrors.Write(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func parseFilter(r *http.Request) (OrderFilter, error) {
	q := r.URL.Query()
	filter := OrderFilter{}
	if raw := q.Get("customer_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return OrderFilter{}, apperrors.Validation("customer_id must be a valid uuid")
		}
		filter.CustomerID = &id
	}
	if raw := q.Get("status"); raw != "" {
		status, err := ParseOrderStatus(raw)
		if err != nil {
			return OrderFilter{}, apperrors.Validation("status must be a valid order status")
		}
		filter.Status = &status
	}
	if raw := q.Get("start_date"); raw != "" {
		t, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return OrderFilter{}, apperrors.Validation("start_date must be YYYY-MM-DD")
		}
		filter.StartDate = &t
	}
	if raw := q.Get("end_date"); raw != "" {
		t, err := time.Parse("2006-01-02", raw)
		if err != nil {
			return OrderFilter{}, apperrors.Validation("end_date must be YYYY-MM-DD")
		}
		filter.EndDate = &t
	}
	page, err := parseIntParam(q.Get("page"))
	if err != nil {
		return OrderFilter{}, apperrors.Validation("page must be an integer")
	}
	filter.Page = page
	pageSize, err := parseIntParam(q.Get("page_size"))
	if err != nil {
		return OrderFilter{}, apperrors.Validation("page_size must be an integer")
	}
	filter.PageSize = pageSize
	return filter, nil
}

func parseIntParam(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	return strconv.Atoi(raw)
}

func pathID(r *http.Request) (uuid.UUID, error) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		return uuid.Nil, apperrors.Validation("invalid order id")
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
