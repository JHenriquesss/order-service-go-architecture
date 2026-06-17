package product

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"

	apperrors "order-service-product/internal/errors"
)

// Handler exposes the five product endpoints (architecture §11). It stays thin:
// decode/parse the request shape, call the service, map the result. All business
// rules live in the service.
type Handler struct {
	service *Service
}

// NewHandler builds the product HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Register mounts the product routes (architecture §11) onto mux.
func (h *Handler) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/products", h.List)
	mux.HandleFunc("POST /api/products", h.Create)
	mux.HandleFunc("GET /api/products/{id}", h.Get)
	mux.HandleFunc("PUT /api/products/{id}", h.Update)
	mux.HandleFunc("PATCH /api/products/{id}/deactivate", h.Deactivate)
}

// Create handles POST /api/products.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input CreateProductInput
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

// Update handles PUT /api/products/{id}.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		apperrors.Write(w, err)
		return
	}
	var input UpdateProductInput
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

// Deactivate handles PATCH /api/products/{id}/deactivate.
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

// Get handles GET /api/products/{id}.
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

// List handles GET /api/products with filtering and pagination (architecture §13).
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

func parseFilter(r *http.Request) (ProductFilter, error) {
	q := r.URL.Query()
	filter := ProductFilter{
		Name: q.Get("name"),
		SKU:  q.Get("sku"),
	}
	if raw := q.Get("active"); raw != "" {
		active, err := strconv.ParseBool(raw)
		if err != nil {
			return ProductFilter{}, apperrors.Validation("active must be a boolean")
		}
		filter.Active = &active
	}
	page, err := parseIntParam(q.Get("page"))
	if err != nil {
		return ProductFilter{}, apperrors.Validation("page must be an integer")
	}
	filter.Page = page
	pageSize, err := parseIntParam(q.Get("page_size"))
	if err != nil {
		return ProductFilter{}, apperrors.Validation("page_size must be an integer")
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
		return uuid.Nil, apperrors.Validation("invalid product id")
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
