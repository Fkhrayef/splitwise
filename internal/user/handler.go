package user

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/fkhayef/splitwise/pkg/response"
)

// Handler handles HTTP requests for user operations
type Handler struct {
	service *Service
}

// NewHandler creates a new user handler with service dependency injected
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// Routes returns the router for user endpoints
func (h *Handler) Routes() chi.Router {
	r := chi.NewRouter()

	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.GetByID)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)

	return r
}

// Create handles POST /users
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	user, err := h.service.Create(r.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyInUse) {
			response.Conflict(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to create user")
		return
	}

	response.JSON(w, http.StatusCreated, user.ToResponse())
}

// GetByID handles GET /users/{id}
func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	user, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to get user")
		return
	}

	response.JSON(w, http.StatusOK, user.ToResponse())
}

// List handles GET /users
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}

	users, total, err := h.service.List(r.Context(), page, perPage)
	if err != nil {
		response.InternalError(w, "Failed to list users")
		return
	}

	userResponses := make([]*UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = user.ToResponse()
	}

	totalPages := (total + perPage - 1) / perPage
	meta := &response.Meta{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}

	response.JSONWithMeta(w, http.StatusOK, userResponses, meta)
}

// Update handles PUT /users/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	user, err := h.service.Update(r.Context(), id, &req)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			response.NotFound(w, err.Error())
			return
		}
		response.InternalError(w, "Failed to update user")
		return
	}

	response.JSON(w, http.StatusOK, user.ToResponse())
}

// Delete handles DELETE /users/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(w, "Invalid user ID")
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		response.InternalError(w, "Failed to delete user")
		return
	}

	response.JSON(w, http.StatusOK, map[string]string{"message": "User deleted successfully"})
}
