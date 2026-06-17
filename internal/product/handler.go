package product

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	apperrors "order-service-go/internal/errors"
)

// Handler exposes the five product endpoints (architecture Ã‚Â§11). It stays thin:
// decode/parse the request shape, call the service, map the result. All business
// rules live in the service.
type Handler struct {
	service *Service
}

// NewHandler builds the product HTTP handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes returns the product routes (architecture §11) as a chi router, mounted
// at /api/products by the server. Authentication and role authorization are
// applied by the server around this mount.
func (h *Handler) Routes() http.Handler {
	r := chi.NewRouter()
	r.Get("/", h.List)
	r.Post("/", h.Create)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Patch("/{id}/deactivate", h.Deactivate)
	return r
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

// List handles GET /api/products with filtering and pagination (architecture Ã‚Â§13).
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
	id, err := uuid.Parse(chi.URLParam(r, "id"))
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
