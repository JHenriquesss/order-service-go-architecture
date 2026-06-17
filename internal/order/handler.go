package order

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "order-service-go/internal/errors"
	"order-service-go/internal/middleware"
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

// Routes returns the order routes as a chi router, mounted at /api/orders by the
// server. Authentication and role authorization are applied around this mount.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	return r
}

// Create handles POST /api/orders.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	rawID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		apperrors.Write(w, apperrors.Unauthorized("Authentication required"))
		return
	}
	createdBy, err := uuid.Parse(rawID)
	if err != nil {
		apperrors.Write(w, apperrors.Unauthorized("Invalid authenticated user"))
		return
	}
	var input CreateOrderInput
	if err := decodeJSON(r, &input); err != nil {
		apperrors.Write(w, err)
		return
	}
	out, err := h.service.Create(r.Context(), createdBy, input)
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
	id, err := uuid.Parse(chi.URLParam(r, "id"))
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
