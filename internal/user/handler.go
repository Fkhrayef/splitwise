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
// @Summary      Create a new user
// @Description  Create a new user with username and email
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        request body CreateUserRequest true "User creation request"
// @Success      201 {object} response.APIResponse{data=UserResponse}
// @Failure      400 {object} response.APIResponse
// @Failure      409 {object} response.APIResponse
// @Router       /users [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.BadRequest(w, "Invalid request body")
		return
	}

	// TODO: Add validation using validator package

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
// @Summary      Get user by ID
// @Description  Get a single user by their ID
// @Tags         users
// @Produce      json
// @Param        id path int true "User ID"
// @Success      200 {object} response.APIResponse{data=UserResponse}
// @Failure      400 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Router       /users/{id} [get]
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
// @Summary      List all users
// @Description  Get a paginated list of all users
// @Tags         users
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        per_page query int false "Items per page" default(20)
// @Success      200 {object} response.APIResponse{data=[]UserResponse}
// @Router       /users [get]
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

	// Convert to response DTOs
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
// @Summary      Update a user
// @Description  Update user's username or avatar
// @Tags         users
// @Accept       json
// @Produce      json
// @Param        id path int true "User ID"
// @Param        request body UpdateUserRequest true "User update request"
// @Success      200 {object} response.APIResponse{data=UserResponse}
// @Failure      400 {object} response.APIResponse
// @Failure      404 {object} response.APIResponse
// @Router       /users/{id} [put]
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
// @Summary      Delete a user
// @Description  Delete a user by their ID
// @Tags         users
// @Produce      json
// @Param        id path int true "User ID"
// @Success      200 {object} response.APIResponse
// @Failure      400 {object} response.APIResponse
// @Router       /users/{id} [delete]
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
